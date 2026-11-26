package migrations

import "testing"

func TestLatestVersion(t *testing.T) {
	version, err := LatestVersion()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if version != 7 {
		t.Fatalf("версия = %d, ожидалась 7", version)
	}
}
