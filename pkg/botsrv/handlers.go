package botsrv

import (
	"botsrv/pkg/db"
	"botsrv/pkg/embedlog"
	"bytes"
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"strconv"
	"sync"
	"text/template"
)

const (
	startCommand      = "/start"
	addPlaceCommand   = "/add_place"
	placesListCommand = "/places"
)

type Config struct {
	Token string
}

type BotManager struct {
	embedlog.Logger
	dbo    db.DB
	cr     db.CommonRepo
	places *sync.Map
}

func NewBotManager(logger embedlog.Logger, dbo db.DB) *BotManager {
	return &BotManager{
		Logger: logger,
		dbo:    dbo,
		cr:     db.NewCommonRepo(dbo),
		places: new(sync.Map),
	}
}

func (bm *BotManager) RegisterBotHandlers(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, startCommand, bot.MatchTypePrefix, bm.StartHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, addPlaceCommand, bot.MatchTypeExact, bm.AddPlaceHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, placesListCommand, bot.MatchTypeExact, bm.PlacesHandler)
}

func (bm *BotManager) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	user, err := bm.cr.OneUser(ctx, &db.UserSearch{ID: pointer(int(update.Message.From.ID))})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
	if user == nil {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Кошмарики, я ничего не поняла....",
		})
		if err != nil {
			bm.Errorf("%v", err)
			return
		}
		return
	}
	placeInterface, ok := bm.places.Load(user.ID)
	if !ok {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Кошмарики, я ничего не поняла....",
		})
		if err != nil {
			bm.Errorf("%v", err)
			return
		}
	}
	place := placeInterface.(*db.Place)
	switch {
	case place.PlaceName == "":
		_, err = bm.cr.UpdateUser(ctx, user)
		if err != nil {
			bm.Errorf("%v", err)
			return
		}
		place.PlaceName = update.Message.Text
		bm.places.Store(user.ID, place)
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Введите приоритет места, любое число, чем больше тем важнее!",
		})
		if err != nil {
			bm.Errorf("%v", err)
			return
		}
	default:
		priority, err := strconv.Atoi(update.Message.Text)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Это не число!",
			})
			return
		}
		place.PlacePriority = priority
		if _, err = bm.cr.AddPlace(ctx, place); err != nil {
			bm.Errorf("%v", err)
			return
		}
		bm.places.Delete(user.ID)
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Местечко успешно добавлено!",
		})
		if err != nil {
			bm.Errorf("%v", err)
			return
		}
	}
}

func (bm *BotManager) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	user, err := bm.cr.OneUser(ctx, &db.UserSearch{ID: pointer(int(update.Message.From.ID))})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
	if user == nil {
		user, err = bm.cr.AddUser(ctx, &db.User{
			ID:       int(update.Message.From.ID),
			Nickname: update.Message.From.Username,
			StatusID: 1,
		})
		if err != nil {
			return
		}
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text: fmt.Sprintf("Привет, @%s! Я-Ленабот!\nНажмите %s, чтобы добавить местечко куда нам нужно сходити!\nНажмите %s, чтобы увидеть весь список таких местечек",
				user.Nickname, addPlaceCommand, placesListCommand),
		})
		if err != nil {
			bm.Errorf("%v", err)
			return
		}
		return
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Привет, @%s! Давно не виделись!", user.Nickname),
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

}

func (bm *BotManager) AddPlaceHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	user, err := bm.cr.OneUser(ctx, &db.UserSearch{ID: pointer(int(update.Message.From.ID))})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
	if user == nil {
		bm.Errorf("user %d not found", update.Message.From.ID)
		return
	}
	bm.places.Store(user.ID, &db.Place{
		UserID: &user.ID,
	})
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Введите название места, куда нам обязательно нужно съездить!",
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

var placesTemplate = `{{- range $index, $place := . }}
{{- printf "\n%d. Название: %s, Приоритет: %d" (add $index 1) $place.PlaceName $place.PlacePriority}}
{{- end }}`

var funcMap = template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
}

func (bm *BotManager) PlacesHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	places, err := bm.cr.PlacesByFilters(ctx, &db.PlaceSearch{}, db.PagerNoLimit, db.WithSort(db.NewSortField(db.Columns.Place.PlacePriority, true)))
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
	if len(places) == 0 {
		return
	}
	tmplPlaces, err := template.New("winsList").Funcs(funcMap).Parse(placesTemplate)
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
	var bufPlaces bytes.Buffer
	if err = tmplPlaces.Execute(&bufPlaces, places); err != nil {
		bm.Errorf("%v", err)
		return
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Вот список всех мест\n" + bufPlaces.String(),
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func pointer[T any](in T) *T {
	return &in
}
