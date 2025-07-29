package botsrv

import (
	"botsrv/pkg/db"
	"botsrv/pkg/embedlog"
	"context"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
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
	dbo db.DB
}

func NewBotManager(logger embedlog.Logger, dbo db.DB) *BotManager {
	return &BotManager{
		Logger: logger,
		dbo:    dbo,
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
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Кошмарики, я ничего не поняла....",
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Привет! Я-Ленабот",
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
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Введите название места, куда нам обязательно нужно съездить!",
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) PlacesHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

}
