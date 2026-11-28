package learning

import "time"

// WeekStart возвращает понедельник недели в UTC.
func WeekStart(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	day := t.AddDate(0, 0, -(weekday - 1))

	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
}

// DateOnly обнуляет время у даты в UTC.
func DateOnly(t time.Time) time.Time {
	t = t.UTC()

	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// DaysBetween возвращает число целых дней между датами.
func DaysBetween(from, to time.Time) int {
	return int(DateOnly(to).Sub(DateOnly(from)).Hours() / 24)
}

// WeeksBetween возвращает число недель между понедельниками.
func WeeksBetween(from, to time.Time) int {
	return int(WeekStart(to).Sub(WeekStart(from)).Hours() / 24 / 7)
}
