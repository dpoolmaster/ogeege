// Package mailer отправляет электронную почту через SMTP.
package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

const defaultTimeout = 10 * time.Second

// Message описывает письмо для отправки.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender отправляет письмо; интерфейс позволяет подменить SMTP в тестах.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Options содержит настройки SMTP-подключения.
type Options struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	TLS      bool
	Timeout  time.Duration
}

// SMTPSender отправляет письма через SMTP-сервер.
type SMTPSender struct {
	opts Options
}

// NewSMTP создаёт отправитель писем.
func NewSMTP(opts Options) *SMTPSender {
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}

	return &SMTPSender{opts: opts}
}

// Send отправляет одно письмо.
func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	if err := validateMessage(s.opts.From, msg); err != nil {
		return err
	}

	addr := net.JoinHostPort(s.opts.Host, fmt.Sprintf("%d", s.opts.Port))

	conn, err := s.dial(ctx, addr)
	if err != nil {
		return fmt.Errorf("подключиться к SMTP: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	_ = conn.SetDeadline(time.Now().Add(s.opts.Timeout))

	client, err := smtp.NewClient(conn, s.opts.Host)
	if err != nil {
		return fmt.Errorf("создать SMTP-клиент: %w", err)
	}
	defer func() {
		_ = client.Close()
	}()

	if s.opts.User != "" {
		auth := smtp.PlainAuth("", s.opts.User, s.opts.Password, s.opts.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("аутентифицироваться в SMTP: %w", err)
		}
	}

	if err := client.Mail(s.opts.From); err != nil {
		return fmt.Errorf("указать отправителя: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("указать получателя: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("начать передачу письма: %w", err)
	}

	if _, err := writer.Write(buildMessage(s.opts.From, msg)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("записать письмо: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("завершить передачу письма: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("завершить SMTP-сеанс: %w", err)
	}

	return nil
}

func (s *SMTPSender) dial(ctx context.Context, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: s.opts.Timeout}

	if s.opts.TLS {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			ServerName: s.opts.Host,
			MinVersion: tls.VersionTLS12,
		})
		if err != nil {
			return nil, err
		}

		return conn, nil
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	return conn, nil
}

func validateMessage(from string, msg Message) error {
	if strings.TrimSpace(from) == "" {
		return errors.New("не задан адрес отправителя")
	}
	if strings.TrimSpace(msg.To) == "" {
		return errors.New("не задан адрес получателя")
	}
	if strings.TrimSpace(msg.Subject) == "" {
		return errors.New("не задана тема письма")
	}

	return nil
}

func buildMessage(from string, msg Message) []byte {
	var builder strings.Builder

	builder.WriteString("From: " + from + "\r\n")
	builder.WriteString("To: " + msg.To + "\r\n")
	builder.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", msg.Subject) + "\r\n")
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	builder.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(msg.Body)
	builder.WriteString("\r\n")

	return []byte(builder.String())
}
