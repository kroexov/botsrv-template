package db

type UserTimeSlots struct {
	WeekDays []WeekDay `json:"weekDays"`
}

// WeekDay represents day of the week from 1 (Monday) to 7 (Sunday) with array of TimeSlot
type WeekDay struct {
	DayNumber int        `json:"dayNumber"`
	Slots     []TimeSlot `json:"slots"`
}

// TimeSlot has id of 1..16, each representing 1 hour slot from 6:00 am to 10:00 pm
type TimeSlot struct {
	Id int `json:"id"`
}
