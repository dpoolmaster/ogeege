package publicsite

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/shared/clock"
)

// DemoCookieName — имя cookie демо-диагностики.
const DemoCookieName = "demo_session"

// DemoTTL — срок жизни демо-сессии.
const DemoTTL = 30 * time.Minute

// DemoPayload содержит состояние демо-диагностики.
type DemoPayload struct {
	ExamYearID uuid.UUID   `json:"exam_year_id"`
	TaskIDs    []uuid.UUID `json:"task_ids"`
	IssuedAt   time.Time   `json:"issued_at"`
	Nonce      string      `json:"nonce"`
}

// DemoSigner подписывает и проверяет cookie демо-диагностики.
type DemoSigner struct {
	secret []byte
	ttl    time.Duration
	clock  clock.Clock
}

// NewDemoSigner создаёт подписыватель демо-сессии.
func NewDemoSigner(secret string, ttl time.Duration, clk clock.Clock) *DemoSigner {
	if ttl <= 0 {
		ttl = DemoTTL
	}
	if clk == nil {
		clk = clock.New()
	}

	return &DemoSigner{secret: []byte(secret), ttl: ttl, clock: clk}
}

// Sign возвращает подписанный токен.
func (s *DemoSigner) Sign(payload *DemoPayload) (string, error) {
	if payload == nil {
		return "", errors.New("пустые данные демо")
	}
	if payload.IssuedAt.IsZero() {
		payload.IssuedAt = s.clock.Now()
	}
	if payload.Nonce == "" {
		nonce, err := randomNonce()
		if err != nil {
			return "", err
		}
		payload.Nonce = nonce
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("сериализовать данные демо: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)

	return encoded + "." + s.signature(encoded), nil
}

// Verify проверяет подпись и срок действия токена.
func (s *DemoSigner) Verify(token string) (*DemoPayload, error) {
	encoded, signature, found := strings.Cut(strings.TrimSpace(token), ".")
	if !found || encoded == "" || signature == "" {
		return nil, ErrInvalidSession
	}
	if subtle.ConstantTimeCompare([]byte(signature), []byte(s.signature(encoded))) != 1 {
		return nil, ErrInvalidSession
	}

	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ErrInvalidSession
	}
	var payload DemoPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, ErrInvalidSession
	}
	if payload.IssuedAt.IsZero() || len(payload.TaskIDs) == 0 {
		return nil, ErrInvalidSession
	}
	if s.clock.Now().After(payload.IssuedAt.Add(s.ttl)) {
		return nil, ErrInvalidSession
	}

	return &payload, nil
}

func (s *DemoSigner) signature(encoded string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(encoded))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("сгенерировать nonce демо: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
