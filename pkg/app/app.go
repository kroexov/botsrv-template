package app

import (
	"context"
	"github.com/go-telegram/bot"
	"gold-botsrv/pkg/botsrv"
	"time"

	"gold-botsrv/pkg/db"

	"github.com/go-pg/pg/v10"
	monitor "github.com/hypnoglow/go-pg-monitor"
	"github.com/labstack/echo/v4"
	"github.com/vmkteam/appkit"
	"github.com/vmkteam/embedlog"
)

type Config struct {
	Database *pg.Options
	Server   struct {
		Host    string
		Port    int
		IsDevel bool
	}
	Bot botsrv.Config
}

type App struct {
	embedlog.Logger
	appName string
	cfg     Config
	db      db.DB
	dbc     *pg.DB
	mon     *monitor.Monitor
	echo    *echo.Echo

	b  *bot.Bot
	bm *botsrv.BotManager
}

func New(appName string, sl embedlog.Logger, cfg Config, db db.DB, dbc *pg.DB) *App {
	a := &App{
		appName: appName,
		cfg:     cfg,
		db:      db,
		dbc:     dbc,
		echo:    appkit.NewEcho(),
		Logger:  sl,
	}

	// add services and managers
	a.bm = botsrv.NewBotManager(a.Logger, a.db)

	opts := []bot.Option{bot.WithDefaultHandler(a.bm.DefaultHandler)}
	b, err := bot.New(cfg.Bot.Token, opts...)
	if err != nil {
		panic(err)
	}
	a.b = b

	return a
}

// Run is a function that runs application.
func (a *App) Run(ctx context.Context) error {
	a.registerMetrics()
	a.registerHandlers()
	a.registerDebugHandlers()
	a.registerMetadata()
	a.bm.RegisterBotHandlers(a.b)

	// start tg bot
	go a.b.Start(context.TODO())

	return a.runHTTPServer(ctx, a.cfg.Server.Host, a.cfg.Server.Port)
}

// Shutdown is a function that gracefully stops HTTP server.
func (a *App) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	a.mon.Close()

	return a.echo.Shutdown(ctx)
}

// registerMetadata is a function that registers meta info from service. Must be updated.
func (a *App) registerMetadata() {
	opts := appkit.MetadataOpts{
		HasPublicAPI:  true,
		HasPrivateAPI: true,
		DBs: []appkit.DBMetadata{
			appkit.NewDBMetadata(a.cfg.Database.Database, a.cfg.Database.PoolSize, false),
		},
		Services: []appkit.ServiceMetadata{
			// NewServiceMetadata("srv", MetadataServiceTypeAsync),
		},
	}

	md := appkit.NewMetadataManager(opts)
	md.RegisterMetrics()

	a.echo.GET("/debug/metadata", md.Handler)
}
