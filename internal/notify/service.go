package notify

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"text/template"
	"time"

	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/ids"
	"oge-ege/internal/shared/mailer"
)

const (
	processBatchSize = 20
	initialBackoff   = 10 * time.Second
	maxErrorLength   = 500
)

// TxManager выполняет функцию в транзакции.
type TxManager interface {
	WithTx(ctx context.Context, fn func(q db.Querier) error) error
}

// Service ставит письма в outbox и отправляет их пачками.
type Service struct {
	repo   Repository
	tx     TxManager
	sender mailer.Sender
	clock  clock.Clock
	logger *slog.Logger
}

// NewService создаёт сервис notify.
func NewService(repo Repository, tx TxManager, sender mailer.Sender, clk clock.Clock, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		tx:     tx,
		sender: sender,
		clock:  clk,
		logger: logger,
	}
}

// Enqueue сохраняет письмо в outbox, не отправляя его синхронно.
func (s *Service) Enqueue(ctx context.Context, q db.Querier, msg Message) error {
	if strings.TrimSpace(msg.Recipient) == "" {
		return errors.New("не задан получатель письма")
	}
	if strings.TrimSpace(msg.Template) == "" {
		return errors.New("не задан шаблон письма")
	}
	if msg.Channel == "" {
		msg.Channel = ChannelEmail
	}

	if _, err := s.repo.GetActiveTemplate(ctx, q, msg.Template); err != nil {
		return err
	}

	id, err := ids.New()
	if err != nil {
		return err
	}

	now := s.clock.Now()
	item := &OutboxItem{
		ID:          id,
		UserID:      msg.UserID,
		Channel:     msg.Channel,
		Recipient:   msg.Recipient,
		Template:    msg.Template,
		Payload:     msg.Payload,
		Status:      StatusPending,
		Attempts:    0,
		ScheduledAt: now,
		DedupKey:    msg.DedupKey,
	}

	if err := s.repo.InsertOutbox(ctx, q, item); err != nil {
		return fmt.Errorf("поставить письмо в outbox: %w", err)
	}

	return nil
}

// ProcessDue отправляет пачку писем, срок которых наступил.
func (s *Service) ProcessDue(ctx context.Context) (int, error) {
	processed := 0

	err := s.tx.WithTx(ctx, func(q db.Querier) error {
		items, err := s.repo.FetchDue(ctx, q, s.clock.Now(), processBatchSize)
		if err != nil {
			return err
		}

		for i := range items {
			if items[i].Status != StatusPending {
				continue
			}

			s.processItem(ctx, q, &items[i])
			processed++
		}

		return nil
	})
	if err != nil {
		return processed, fmt.Errorf("обработать outbox: %w", err)
	}

	return processed, nil
}

func (s *Service) processItem(ctx context.Context, q db.Querier, item *OutboxItem) {
	attempts := item.Attempts + 1

	templateRecord, err := s.repo.GetActiveTemplate(ctx, q, item.Template)
	if err != nil {
		s.fail(ctx, q, item, attempts, "шаблон письма недоступен")
		return
	}

	subject, body, err := renderTemplate(templateRecord, item.Payload)
	if err != nil {
		s.fail(ctx, q, item, attempts, "не удалось сформировать письмо")
		return
	}

	sendErr := s.sender.Send(ctx, mailer.Message{
		To:      item.Recipient,
		Subject: subject,
		Body:    body,
	})
	if sendErr == nil {
		if err := s.repo.MarkSent(ctx, q, item.ID, s.clock.Now(), attempts); err != nil {
			s.logger.Error("не удалось отметить письмо отправленным", "outbox_id", item.ID, "template", item.Template, "error", err)
			return
		}

		s.logger.Info("письмо отправлено", "outbox_id", item.ID, "template", item.Template, "attempts", attempts, "status", string(StatusSent))
		return
	}

	lastError := sanitizeError(sendErr, item.Recipient)
	s.fail(ctx, q, item, attempts, lastError)
}

func (s *Service) fail(ctx context.Context, q db.Querier, item *OutboxItem, attempts int, lastError string) {
	if attempts >= MaxAttempts {
		if err := s.repo.MarkFailed(ctx, q, item.ID, attempts, lastError); err != nil {
			s.logger.Error("не удалось отметить письмо неуспешным", "outbox_id", item.ID, "template", item.Template, "error", err)
			return
		}

		s.logger.Warn("письмо не отправлено, попытки исчерпаны", "outbox_id", item.ID, "template", item.Template, "attempts", attempts, "status", string(StatusFailed))
		return
	}

	scheduledAt := s.clock.Now().Add(backoff(attempts))
	if err := s.repo.Reschedule(ctx, q, item.ID, attempts, scheduledAt, lastError); err != nil {
		s.logger.Error("не удалось перенести отправку письма", "outbox_id", item.ID, "template", item.Template, "error", err)
		return
	}

	s.logger.Warn("отправка письма отложена", "outbox_id", item.ID, "template", item.Template, "attempts", attempts, "status", string(StatusPending))
}

func renderTemplate(record *Template, payload map[string]any) (string, string, error) {
	subject, err := renderString(record.Subject, payload)
	if err != nil {
		return "", "", err
	}

	body, err := renderString(record.BodyMD, payload)
	if err != nil {
		return "", "", err
	}

	return subject, body, nil
}

func renderString(source string, payload map[string]any) (string, error) {
	parsed, err := template.New("email").Option("missingkey=zero").Parse(source)
	if err != nil {
		return "", fmt.Errorf("разобрать шаблон письма: %w", err)
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, payload); err != nil {
		return "", fmt.Errorf("применить шаблон письма: %w", err)
	}

	return buf.String(), nil
}

func backoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}

	delay := initialBackoff
	for i := 1; i < attempts; i++ {
		delay *= 2
		if delay > time.Hour {
			return time.Hour
		}
	}

	return delay
}

func sanitizeError(err error, recipient string) string {
	message := err.Error()
	if recipient != "" {
		message = strings.ReplaceAll(message, recipient, "[email]")
	}
	message = redactLinks(message)
	if len(message) > maxErrorLength {
		message = message[:maxErrorLength]
	}

	return message
}

func redactLinks(message string) string {
	fields := strings.Fields(message)
	for i, field := range fields {
		if strings.HasPrefix(field, "http://") || strings.HasPrefix(field, "https://") {
			fields[i] = "[ссылка]"
		}
	}

	return strings.Join(fields, " ")
}
