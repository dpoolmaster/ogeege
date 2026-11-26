package clock

import (
	"testing"
	"time"
)

type fixedClock struct {
	now time.Time
}

func (f fixedClock) Now() time.Time {
	return f.now
}

func TestFixedClock(t *testing.T) {
	want := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	var c Clock = fixedClock{now: want}

	if got := c.Now(); !got.Equal(want) {
		t.Fatalf("Now() = %s, ожидалось %s", got, want)
	}
}

func TestSystemClock(t *testing.T) {
	before := time.Now()
	got := New().Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Fatalf("Now() = %s вне диапазона %s..%s", got, before, after)
	}
}
