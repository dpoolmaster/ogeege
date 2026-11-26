// Package clock предоставляет абстракцию текущего времени для тестов.
package clock

import "time"

// Clock возвращает текущее время.
type Clock interface {
	Now() time.Time
}

// System — реализация Clock на основе системного времени.
type System struct{}

// Now возвращает текущее системное время.
func (System) Now() time.Time {
	return time.Now()
}

// New возвращает системные часы.
func New() Clock {
	return System{}
}
