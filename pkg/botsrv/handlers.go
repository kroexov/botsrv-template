package botsrv

import (
	"context"
	"sync"

	"gold-botsrv/pkg/db"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/vmkteam/embedlog"
)

const (
	startCommand = "/start"
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

// RegisterBotHandlers is a function to register all telegram bot handlers
func (bm *BotManager) RegisterBotHandlers(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, startCommand, bot.MatchTypePrefix, bm.StartHandler)
}

// DefaultHandler is a handler if no match for user call is found
func (bm *BotManager) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.InlineQuery != nil && update.InlineQuery.From != nil {
		if err := bm.answerInlineQuery(ctx, b, update); err != nil {
			bm.Errorf("%v", err)
		}
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Default bot answer",
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

// answerInlineQuery is a function to handle all inline bot calls
func (bm *BotManager) answerInlineQuery(ctx context.Context, b *bot.Bot, update *models.Update) error {
	_, err := b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: update.InlineQuery.ID,
		Results: []models.InlineQueryResult{
			&models.InlineQueryResultArticle{
				ID:    "1",
				Title: "Nothing!",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "Default inline query answer",
				},
				ReplyMarkup:  nil,
				ThumbnailURL: "https://img.freepik.com/premium-vector/vector-loading-icon-vector-editable-element-design_898898-180.jpg",
			},
		},
		CacheTime:  1,
		IsPersonal: true,
	})
	if err != nil {
		return err
	}

	return nil
}

// StartHandler is a handler to startCommand
func (bm *BotManager) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Start command bot answer",
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

// nolint:unused
func pointer[T any](in T) *T {
	return &in
}
