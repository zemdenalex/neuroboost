package handlers

import "time"

// Dates in these tests are relative to today: a fixed «14.10» was a date in the
// future when the tests were written and, from 15.10, one the bot (rightly)
// reads as next year's. The zone is the test account's, Europe/Moscow.
func moscowDay(offsetDays int) (ddmm string, midnight time.Time) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	n := time.Now().In(loc)
	d := time.Date(n.Year(), n.Month(), n.Day()+offsetDays, 0, 0, 0, 0, loc)
	return d.Format("02.01"), d
}
