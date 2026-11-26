package mailer

import (
	"context"
	"strings"
	"testing"
)

type recordingSender struct {
	messages []Message
	err      error
}

func (r *recordingSender) Send(_ context.Context, msg Message) error {
	r.messages = append(r.messages, msg)
	return r.err
}

func TestSenderSubstitution(t *testing.T) {
	var sender Sender = &recordingSender{}

	err := sender.Send(context.Background(), Message{
		To:      "student@example.ru",
		Subject: "Подтверждение",
		Body:    "Текст письма",
	})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	recorder, ok := sender.(*recordingSender)
	if !ok {
		t.Fatal("ожидался тестовый отправитель")
	}
	if len(recorder.messages) != 1 || recorder.messages[0].To != "student@example.ru" {
		t.Fatalf("письмо не зафиксировано: %+v", recorder.messages)
	}
}

func TestSMTPRejectsInvalidMessage(t *testing.T) {
	sender := NewSMTP(Options{Host: "localhost", Port: 1025, From: "no-reply@example.ru"})

	tests := []struct {
		name string
		msg  Message
		want string
	}{
		{name: "без получателя", msg: Message{Subject: "Тема"}, want: "получателя"},
		{name: "без темы", msg: Message{To: "a@b.ru"}, want: "тема"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sender.Send(context.Background(), tt.msg)
			if err == nil {
				t.Fatal("ожидалась ошибка")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ошибка %q не содержит %q", err.Error(), tt.want)
			}
		})
	}
}

func TestBuildMessage(t *testing.T) {
	raw := string(buildMessage("no-reply@example.ru", Message{
		To:      "student@example.ru",
		Subject: "Подтверждение",
		Body:    "Строка 1\nСтрока 2",
	}))

	for _, part := range []string{
		"From: no-reply@example.ru",
		"To: student@example.ru",
		"Content-Type: text/plain; charset=UTF-8",
		"Строка 1",
	} {
		if !strings.Contains(raw, part) {
			t.Fatalf("письмо не содержит %q: %s", part, raw)
		}
	}
}
