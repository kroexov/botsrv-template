package botsrv

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"gold-botsrv/pkg/db"
	"gold-botsrv/pkg/timetables"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/vmkteam/embedlog"
)

const (
	startCommand    = "/start"
	tasksCommand    = "/tasks"
	addTaskCommand  = "/add_task"
	generateCommand = "/generate"
	settingsCommand = "/settings"

	callbackPrefixDays   = "days"
	callbackPrefixSlots  = "slots"
	callbackReturnToDays = "return_days"
)

var days = map[int]string{
	0: "Воскресенье",
	1: "Понедельник",
	2: "Вторник",
	3: "Среда",
	4: "Четверг",
	5: "Пятница",
	6: "Суббота",
}

var commonTimeSlots = []int{6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22}

type Config struct {
	Token string
}

type BotManager struct {
	embedlog.Logger
	dbo    db.DB
	cr     db.TimetablesRepo
	places *sync.Map
	tm     *timetables.TimeTableManager
}

func NewBotManager(logger embedlog.Logger, dbo db.DB) *BotManager {
	return &BotManager{
		Logger: logger,
		dbo:    dbo,
		cr:     db.NewTimetablesRepo(dbo),
		places: new(sync.Map),
		tm:     timetables.NewTimeTableManager(dbo, logger),
	}
}

// RegisterBotHandlers is a function to register all telegram bot handlers
func (bm *BotManager) RegisterBotHandlers(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, startCommand, bot.MatchTypePrefix, bm.StartHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, tasksCommand, bot.MatchTypePrefix, bm.TasksHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, addTaskCommand, bot.MatchTypePrefix, bm.AddTaskHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, generateCommand, bot.MatchTypePrefix, bm.GenerateHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, settingsCommand, bot.MatchTypePrefix, bm.SettingsHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, callbackReturnToDays, bot.MatchTypePrefix, bm.ReturnToDaysHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, callbackPrefixDays, bot.MatchTypePrefix, bm.DaysHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, callbackPrefixSlots, bot.MatchTypePrefix, bm.ChangeSlotHandler)
}

// DefaultHandler is a handler if no match for user call is found
//
//nolint:errcheck,govet
func (bm *BotManager) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user, err := bm.cr.UserByID(ctx, int(update.Message.From.ID))
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	if user != nil && user.StatusID == db.StatusWaitTask {
		task, err := ParseTaskFromText(update.Message.Text, int(update.Message.From.ID))
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   err.Error(),
			})
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "успех",
		})
		_, err = bm.cr.AddTask(ctx, task)
		if err != nil {
			bm.Errorf("%v", err)
			return
		}
		_, err = bm.cr.UpdateUser(ctx, &db.User{
			ID:       int(update.Message.From.ID),
			StatusID: db.StatusEnabled,
		}, db.WithColumns(db.Columns.User.StatusID))
		if err != nil {
			bm.Errorf("%v", err)
			return
		}

		return
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Default bot answer",
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

// ParseTaskFromText parses incoming text message to task
//
//nolint:perfsprint,
func ParseTaskFromText(text string, userTgID int) (*db.Task, error) {
	lines := strings.Split(strings.TrimSpace(text), "\n")

	if len(lines) < 4 {
		return nil, fmt.Errorf("неверный формат. Ожидается 4 строки, получено %d", len(lines))
	}

	task := &db.Task{
		UserTgID: &userTgID,
		StatusID: db.StatusEnabled,
	}

	task.Description = strings.TrimSpace(lines[0])
	if task.Description == "" {
		return nil, fmt.Errorf("описание не может быть пустым")
	}

	deadline, err := time.Parse("02.01.2006", strings.TrimSpace(lines[1]))
	if err != nil {
		return nil, fmt.Errorf("неверный формат даты. Ожидается DD.MM.YYYY: %w", err)
	}
	task.Deadline = deadline

	length, err := strconv.Atoi(strings.TrimSpace(lines[2]))
	if err != nil {
		return nil, fmt.Errorf("неверный формат длительности. Ожидается число минут: %w", err)
	}
	if length <= 0 {
		return nil, fmt.Errorf("длительность должна быть положительным числом")
	}
	task.Length = length

	priority, err := strconv.Atoi(strings.TrimSpace(lines[3]))
	if err != nil {
		return nil, fmt.Errorf("неверный формат приоритета. Ожидается число 1-10: %w", err)
	}
	if priority < 1 || priority > 10 {
		return nil, fmt.Errorf("приоритет должен быть от 1 до 10")
	}
	task.Priority = priority

	return task, nil
}

// StartHandler is a handler to startCommand
func (bm *BotManager) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: `
Нажмите /settings чтобы настроить расписание
Нажмите /tasks чтобы посмотреть свои задачи
Нажмите /add_task чтобы добавить новую задачу
Нажмите /generate чтобы сгенерировать дедлайны
`,
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	user, err := bm.cr.UserByID(ctx, int(update.Message.From.ID))
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	if user == nil {
		_, err = bm.cr.AddUser(ctx, &db.User{
			ID:       int(update.Message.From.ID),
			StatusID: db.StatusEnabled,
		})
		if err != nil {
			bm.Errorf("%v", err)
			return
		}
	}
}

func (bm *BotManager) SettingsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Настройки расписания",
		ReplyMarkup: generateSettingsDays(),
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) ReturnToDaysHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      update.CallbackQuery.From.ID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        "Настройки расписания",
		ReplyMarkup: generateSettingsDays(),
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) DaysHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	data := strings.Split(update.CallbackQuery.Data, "_")
	if len(data) < 2 {
		return
	}

	dayNumber, err := strconv.Atoi(data[1])
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	settings, err := bm.tm.UserSettings(ctx, int(update.CallbackQuery.From.ID))
	if err != nil {
		bm.Errorf("%v", err)
		return
	} else if settings == nil {
		return
	}

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      update.CallbackQuery.From.ID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        days[dayNumber],
		ReplyMarkup: generateSettingsSlots(settings, dayNumber),
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) ChangeSlotHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	data := strings.Split(update.CallbackQuery.Data, "_")
	if len(data) < 4 {
		return
	}
	userID := int(update.CallbackQuery.From.ID)

	settings, err := bm.tm.UserSettings(ctx, userID)
	if err != nil {
		bm.Errorf("%v", err)
		return
	} else if settings == nil {
		return
	}

	dayNumber, slotNumber, checked, err := parseParams(data)
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	regenerateSlots(settings, dayNumber, slotNumber, checked)

	_, err = bm.cr.UpdateUser(ctx, &db.User{
		ID:        userID,
		TimeSlots: *settings,
	}, db.WithColumns(db.Columns.User.TimeSlots))
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      update.CallbackQuery.From.ID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		ReplyMarkup: generateSettingsSlots(settings, dayNumber),
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func parseParams(data []string) (int, int, bool, error) {
	dayNumber, err := strconv.Atoi(data[1])
	if err != nil {
		return 0, 0, false, err
	}
	slotNumber, err := strconv.Atoi(data[2])
	if err != nil {
		return 0, 0, false, err
	}
	checked, err := strconv.ParseBool(data[3])
	if err != nil {
		return 0, 0, false, err
	}
	return dayNumber, slotNumber, checked, nil
}

func (bm *BotManager) GenerateHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	tasks, err := bm.tm.GenerateTimeTable(ctx, int(update.Message.From.ID))
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	res, err := FormatTasksMessage(tasks)
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   res,
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) TasksHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	tasks, err := bm.cr.TasksByFilters(ctx, &db.TaskSearch{UserTgID: pointer(int(update.Message.From.ID))}, db.PagerNoLimit)
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	res, err := FormatTasksMessage(tasks)
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   res,
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) AddTaskHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: `
Отправьте задачу в формате
{Описание}
{Дата дедлайна DD.MM.YYYY}
{Длительность в минутах}
{Приоритет 1-10, выше = важнее}

Пример:
Сделать ДЗ физика
07.12.2025
120
1
`,
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	_, err = bm.cr.UpdateUser(ctx, &db.User{
		ID:       int(update.Message.From.ID),
		StatusID: db.StatusWaitTask,
	}, db.WithColumns(db.Columns.User.StatusID))
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func regenerateSlots(settings *db.UserTimeSlots, dayNumber int, slotNumber int, checked bool) {
	if settings.WeekDays == nil {
		settings.WeekDays = make(map[int][]db.TimeSlot)
	}

	checkedTimeSlots := generateCheckedTimeSlots(settings, dayNumber, slotNumber, checked)

	slices.SortFunc(checkedTimeSlots, func(a, b db.TimeSlot) int {
		return a.StartHour - b.StartHour
	})

	var res []db.TimeSlot

	if len(checkedTimeSlots) == 0 {
		settings.WeekDays[dayNumber] = res
		return
	}

	current := checkedTimeSlots[0]

	for i := 1; i < len(checkedTimeSlots); i++ {
		if current.EndHour == checkedTimeSlots[i].StartHour {
			current.EndHour = checkedTimeSlots[i].EndHour
		} else {
			res = append(res, current)
			current = checkedTimeSlots[i]
		}
	}

	res = append(res, current)

	settings.WeekDays[dayNumber] = res
}

//nolint:prealloc
func generateCheckedTimeSlots(settings *db.UserTimeSlots, dayNumber int, slotNumber int, checked bool) []db.TimeSlot {
	var checkedTimeSlots []db.TimeSlot

	for _, slot := range settings.WeekDays[dayNumber] {
		for i := slot.StartHour; i < slot.EndHour; i++ {
			if i == slotNumber && checked {
				continue
			}
			checkedTimeSlots = append(checkedTimeSlots, db.TimeSlot{
				StartHour: i,
				EndHour:   i + 1,
			})
		}
	}

	if !checked {
		checkedTimeSlots = append(checkedTimeSlots, db.TimeSlot{
			StartHour: slotNumber,
			EndHour:   slotNumber + 1,
		})
	}
	return checkedTimeSlots
}

//nolint:prealloc
func generateSettingsSlots(settings *db.UserTimeSlots, dayNumber int) models.InlineKeyboardMarkup {
	var res [][]models.InlineKeyboardButton
	res = append(res, []models.InlineKeyboardButton{
		{
			Text:         "Назад",
			CallbackData: callbackReturnToDays,
		},
	})
	for _, commonSlot := range commonTimeSlots {
		var checked bool
		prefix := "❌"

		for _, slot := range settings.WeekDays[dayNumber] {
			if slot.StartHour <= commonSlot && slot.EndHour > commonSlot {
				checked = true
				prefix = "✅"
				break
			}
		}
		res = append(res, []models.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("%s %d:00-%d:00", prefix, commonSlot, commonSlot+1),
				CallbackData: fmt.Sprintf("%s_%d_%d_%v", callbackPrefixSlots, dayNumber, commonSlot, checked),
			},
		})
	}

	return models.InlineKeyboardMarkup{InlineKeyboard: res}
}

// nolint:prealloc
func generateSettingsDays() models.InlineKeyboardMarkup {
	var res [][]models.InlineKeyboardButton
	for i := range 7 {
		res = append(res, []models.InlineKeyboardButton{
			{
				Text:         days[i],
				CallbackData: fmt.Sprintf("%s_%d", callbackPrefixDays, i),
			},
		})
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: res}
}

// nolint:unused
func pointer[T any](in T) *T {
	return &in
}
