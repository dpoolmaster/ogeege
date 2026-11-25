package ids

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
)

func TestNewReturnsUUIDv7(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if id.Version() != 7 {
		t.Fatalf("версия UUID = %d, ожидалась 7", id.Version())
	}
}

func TestNewIsNonDecreasing(t *testing.T) {
	var previous uuid.UUID

	for i := 0; i < 100; i++ {
		id, err := New()
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if i > 0 && bytes.Compare(id[:], previous[:]) < 0 {
			t.Fatalf("идентификатор %s меньше предыдущего %s", id, previous)
		}
		previous = id
	}
}
