package notify

import (
	"os"
	"strings"
	"testing"
)

func TestLeadNotificationTemplateMatchesMigration(t *testing.T) {
	body, err := os.ReadFile("../../migrations/000009_notify_lead_template.up.sql")
	if err != nil {
		t.Fatalf("прочитать миграцию шаблона: %v", err)
	}
	if !strings.Contains(string(body), "'"+TemplateCodeLeadNotification+"'") {
		t.Fatalf("код шаблона %q не совпадает с миграцией", TemplateCodeLeadNotification)
	}
}
