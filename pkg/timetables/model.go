package timetables

import (
	"gold-botsrv/pkg/db"
	"slices"
	"time"
)

const timeSlotIncrement = 6

type Task db.Task

type User db.User

type TimeSlot struct {
	Start time.Time
	End   time.Time
}

// TimeTable returns full TimeTable for this week for current user
func (u User) TimeTable() []TimeSlot {
	t := time.Now()
	today := t.Weekday()
	var res []TimeSlot
	// fill timeslots for all week
	for weekday, slots := range u.TimeSlots.WeekDays {
		for _, slot := range slots {
			start := time.Date(t.Year(), t.Month(), t.Day()+daysUntilTargetDay(int(today), weekday),
				slot.StartHour, 0, 0, 0, time.Local)

			end := time.Date(t.Year(), t.Month(), t.Day()+daysUntilTargetDay(int(today), weekday),
				slot.EndHour, 0, 0, 0, time.Local)

			res = append(res, TimeSlot{
				Start: start,
				End:   end,
			})
		}
	}

	slices.SortFunc(res, func(a, b TimeSlot) int {
		if a.Start.Before(b.Start) {
			return -1
		}
		return 1
	})

	return res
}

// daysUntilTargetDay counts how much days is left before target weekDay
func daysUntilTargetDay(currentDay, targetDay int) int {
	if targetDay > currentDay {
		return targetDay - currentDay
	} else {
		return 7 - currentDay + targetDay
	}
}
