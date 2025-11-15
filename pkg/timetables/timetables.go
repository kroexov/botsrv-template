package timetables

import (
	"context"
	"errors"
	"time"

	"gold-botsrv/pkg/db"

	"github.com/vmkteam/embedlog"
)

type TimeTableManager struct {
	tr  db.TimetablesRepo
	dbc db.DB
	embedlog.Logger
}

func NewTimeTableManager(dbc db.DB, logger embedlog.Logger) *TimeTableManager {
	return &TimeTableManager{dbc: dbc, Logger: logger, tr: db.NewTimetablesRepo(dbc)}
}

var ErrNotFound = errors.New("not found")

func (tm *TimeTableManager) GenerateTimeTable(ctx context.Context, userId int) ([]db.Task, error) {
	user, err := tm.tr.UserByID(ctx, userId)

	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, ErrNotFound
	}

	// find all tasks, sort by deadline asc first, priority desc second
	tasks, err := tm.tr.TasksByFilters(ctx, &db.TaskSearch{UserTgID: &user.ID}, db.PagerNoLimit,
		db.WithSort(
			db.NewSortField(db.Columns.Task.Deadline, false),
			db.NewSortField(db.Columns.Task.Priority, true)))

	if err != nil {
		return nil, err
	}

	timeTable := User(*user).TimeTable()

	setTasksStartTime(tasks, timeTable)

	for i, task := range tasks {
		if task.StartAt == nil {
			tasks[i].StatusID = db.StatusDisabled
		}
		_, err = tm.tr.UpdateTask(ctx, &task,
			db.WithColumns(db.Columns.Task.StartAt, db.Columns.Task.StatusID))
		if err != nil {
			return nil, err
		}
	}

	return tasks, nil
}

func setTasksStartTime(tasks []db.Task, timeTable []TimeSlot) {
taskCycle:
	for i, task := range tasks {
		for j := 0; j <= len(timeTable)-1; j++ {
			// check deadline, skip if too late
			if timeTable[j].Start.After(task.Deadline) {
				continue
			}

			// try to check if task can be placed here; if yes, shift slot start
			shift := timeTable[j].Start.Add(time.Minute * time.Duration(task.Length))
			if shift.Before(timeTable[j].End) {
				start := timeTable[j].Start
				tasks[i].StartAt = &start
				timeTable[j].Start = shift
				continue taskCycle
			}
		}
	}
}

func (tm *TimeTableManager) UserSettings(ctx context.Context, userId int) (*db.UserTimeSlots, error) {
	user, err := tm.tr.UserByID(ctx, userId)

	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, ErrNotFound
	}

	return &user.TimeSlots, nil
}
