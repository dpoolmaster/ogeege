package migrations

import "testing"

func TestLatestVersion(t *testing.T) {
	version, err := LatestVersion()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if version != 11 {
		t.Fatalf("версия = %d, ожидалась 11", version)
	}
}
