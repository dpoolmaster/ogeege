package migrations

import "testing"

func TestLatestVersion(t *testing.T) {
	version, err := LatestVersion()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if version != 5 {
		t.Fatalf("версия = %d, ожидалась 5", version)
	}
}
