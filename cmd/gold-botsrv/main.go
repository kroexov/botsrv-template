package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gold-botsrv/pkg/app"
	"gold-botsrv/pkg/db"

	"github.com/BurntSushi/toml"
	"github.com/getsentry/sentry-go"
	"github.com/go-pg/pg/v10"
	"github.com/namsral/flag"
	"github.com/vmkteam/appkit"
	"github.com/vmkteam/embedlog"
)

const appName = "gold-botsrv"

var (
	fs           = flag.NewFlagSetWithEnvPrefix(os.Args[0], strings.ToUpper(appName), 0)
	flConfigPath = fs.String("config", "config.toml", "Path to config file")
	flVerbose    = fs.Bool("verbose", false, "enable debug output")
	flJSONLogs   = fs.Bool("json", false, "enable json output")
	flDev        = fs.Bool("dev", false, "enable dev mode")
	cfg          app.Config
)

func main() {
	flag.DefaultConfigFlagname = "config.flag"
	exitOnError(fs.Parse(os.Args[1:]))

	// setup logger
	sl, ctx := embedlog.NewLogger(*flVerbose, *flJSONLogs), context.Background()
	if *flDev {
		sl = embedlog.NewDevLogger()
	}
	slog.SetDefault(sl.Log()) // set default logger
	ql := db.NewQueryLogger(sl)
	pg.SetLogger(ql)

	version := appkit.Version()
	sl.Print(ctx, "starting", "app", appName, "version", version)
	if _, err := toml.DecodeFile(*flConfigPath, &cfg); err != nil {
		exitOnError(err)
	}

	// check db connection
	pgdb := pg.Connect(cfg.Database)
	dbc := db.New(pgdb)

	v, err := dbc.Version()
	exitOnError(err)
	sl.Print(ctx, "connected to db", "version", v)

	// log all sql queries
	if *flDev {
		pgdb.AddQueryHook(ql)
	}

	// create & run app
	a := app.New(appName, sl, cfg, dbc, pgdb)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// run app and send panic to sentry
	go func() {
		defer func() {
			if err := recover(); err != nil {
				sentry.CurrentHub().Recover(err)
				sentry.Flush(time.Second * 3)
				panic(err)
			}
		}()

		er := a.Run(ctx)
		if errors.Is(er, http.ErrServerClosed) {
			er = nil
		}

		// exit after run failed
		a.PrintOrErr(ctx, "server stopped", er)
		quit <- syscall.SIGTERM
	}()

	<-quit

	if err = a.Shutdown(5 * time.Second); err != nil {
		a.Error(ctx, "shutting down service", "err", err)
	}
}

// exitOnError calls log.Fatal if err wasn't nil.
func exitOnError(err error) {
	if err != nil {
		//nolint:sloglint,noctx
		slog.Error(err.Error())
		os.Exit(1)
	}
}
