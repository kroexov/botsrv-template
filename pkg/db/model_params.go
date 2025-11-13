package db

type UserTimeSlots struct {
	// WeekDays represent days of the week from 0 (Sunday) to 6 (Saturday) with array of TimeSlot
	WeekDays map[int][]TimeSlot `json:"weekDays"`
}

// TimeSlot has StartHour, EndHour of 6..22, each representing 1 hour from 6:00 AM to 10:00 PM,
type TimeSlot struct {
	StartHour int `json:"startHour"`
	EndHour   int `json:"endHour"`
}
