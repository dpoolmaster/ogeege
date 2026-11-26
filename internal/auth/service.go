package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"oge-ege/internal/notify"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/ids"
)

const (
	// SessionDuration — срок жизни сессии.
	SessionDuration = 30 * 24 * time.Hour
	// VerificationTTL — срок жизни токена подтверждения email.
	VerificationTTL = 24 * time.Hour
	// PasswordResetTTL — срок жизни токена сброса пароля.
	PasswordResetTTL = time.Hour
	// EmailResendCooldown — минимальный интервал между письмами подтверждения и сброса.
	EmailResendCooldown = time.Minute
	// LoginAttemptWindow — окно подсчёта неудачных попыток.
	LoginAttemptWindow = 15 * time.Minute
	// MaxLoginAttempts — допустимое число неудачных попыток.
	MaxLoginAttempts = 5

	minPasswordLength = 10
	maxPasswordLength = 128

	// ConsentPersonalData — согласие на обработку персональных данных.
	ConsentPersonalData = "personal_data"
	// ConsentLegalRepresentative — согласие законного представителя.
	ConsentLegalRepresentative = "legal_representative"
	// DefaultConsentVersion — текущая версия документов согласий.
	DefaultConsentVersion = "1"
)

var (
	// ErrInvalidEmail возвращается при некорректном email.
	ErrInvalidEmail = errors.New("некорректный email")
	// ErrWeakPassword возвращается при слишком коротком пароле.
	ErrWeakPassword = fmt.Errorf("пароль должен содержать не менее %d символов", minPasswordLength)
	// ErrEmailTaken возвращается при занятом email.
	ErrEmailTaken = errors.New("email уже зарегистрирован")
	// ErrInvalidRole возвращается при недоступной публичной роли.
	ErrInvalidRole = errors.New("роль недоступна при регистрации")
	// ErrConsentRequired возвращается без обязательного согласия.
	ErrConsentRequired = errors.New("необходимо согласие на обработку персональных данных")
	// ErrInvalidCredentials возвращается при неверных данных входа.
	ErrInvalidCredentials = errors.New("неверный email или пароль")
	// ErrEmailNotVerified возвращается до подтверждения email.
	ErrEmailNotVerified = errors.New("подтвердите адрес электронной почты")
	// ErrAccountBlocked возвращается для заблокированной учётной записи.
	ErrAccountBlocked = errors.New("учётная запись заблокирована")
	// ErrTooManyAttempts возвращается при превышении лимита попыток.
	ErrTooManyAttempts = errors.New("слишком много неудачных попыток входа")
	// ErrInvalidToken возвращается для неизвестного токена.
	ErrInvalidToken = errors.New("токен недействителен")
	// ErrTokenExpired возвращается для истёкшего токена.
	ErrTokenExpired = errors.New("срок действия токена истёк")
	// ErrTokenUsed возвращается для уже использованного токена.
	ErrTokenUsed = errors.New("токен уже использован")
	// ErrInvalidCurrentPassword возвращается при неверном текущем пароле.
	ErrInvalidCurrentPassword = errors.New("неверный текущий пароль")
)

var dummyPasswordHash, _ = HashPassword("dummy-password-value")

// TxManager выполняет функцию в транзакции.
type TxManager interface {
	WithTx(ctx context.Context, fn func(q db.Querier) error) error
}

// EmailEnqueuer ставит письмо в outbox в транзакции вызывающего модуля.
type EmailEnqueuer interface {
	Enqueue(ctx context.Context, q db.Querier, msg notify.Message) error
}

// RegisterInput содержит данные регистрации.
type RegisterInput struct {
	Email    string
	Password string
	Role     Role
	Consents []ConsentGrant
	IP       string
}

// Service содержит сценарии аутентификации.
type Service struct {
	repo     Repository
	db       db.Querier
	tx       TxManager
	enqueuer EmailEnqueuer
	clock    clock.Clock
	baseURL  string
	logger   *slog.Logger
}

// NewService создаёт сервис auth.
func NewService(repo Repository, database db.Querier, tx TxManager, enqueuer EmailEnqueuer, clk clock.Clock, baseURL string, logger *slog.Logger) *Service {
	return &Service{
		repo:     repo,
		db:       database,
		tx:       tx,
		enqueuer: enqueuer,
		clock:    clk,
		baseURL:  strings.TrimRight(baseURL, "/"),
		logger:   logger,
	}
}

// ValidateEmail проверяет и нормализует email.
func ValidateEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 254 {
		return "", ErrInvalidEmail
	}

	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || !strings.Contains(value, "@") {
		return "", ErrInvalidEmail
	}

	return strings.ToLower(value), nil
}

// ValidatePassword проверяет длину пароля.
func ValidatePassword(value string) error {
	length := utf8.RuneCountInString(value)
	if length < minPasswordLength || length > maxPasswordLength {
		return ErrWeakPassword
	}

	return nil
}

// NormalizeConsents проверяет набор согласий и заполняет версии документов.
func NormalizeConsents(grants []ConsentGrant) ([]ConsentGrant, error) {
	normalized := make([]ConsentGrant, 0, len(grants))
	personalData := false

	for _, grant := range grants {
		if grant.Kind != ConsentPersonalData && grant.Kind != ConsentLegalRepresentative {
			return nil, fmt.Errorf("неизвестный вид согласия %q", grant.Kind)
		}
		if grant.DocumentVersion == "" {
			grant.DocumentVersion = DefaultConsentVersion
		}
		if grant.Kind == ConsentPersonalData && grant.Granted {
			personalData = true
		}
		normalized = append(normalized, grant)
	}

	if !personalData {
		return nil, ErrConsentRequired
	}

	return normalized, nil
}

// Register создаёт учётную запись в статусе «не подтверждён».
func (s *Service) Register(ctx context.Context, in RegisterInput) (*User, error) {
	email, err := ValidateEmail(in.Email)
	if err != nil {
		return nil, err
	}
	if err := ValidatePassword(in.Password); err != nil {
		return nil, err
	}
	if !in.Role.IsPublic() {
		return nil, ErrInvalidRole
	}
	if _, err := NormalizeConsents(in.Consents); err != nil {
		return nil, err
	}

	passwordHash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	token, tokenHash, err := NewToken()
	if err != nil {
		return nil, err
	}

	userID, err := ids.New()
	if err != nil {
		return nil, err
	}
	verificationID, err := ids.New()
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	user := &User{
		ID:           userID,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         in.Role,
		Status:       StatusUnverified,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	verification := &Verification{
		ID:        verificationID,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(VerificationTTL),
		CreatedAt: now,
	}

	err = s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.CreateUser(ctx, q, user); err != nil {
			return err
		}
		if err := s.repo.CreateEmailVerification(ctx, q, verification); err != nil {
			return err
		}

		return s.enqueuer.Enqueue(ctx, q, notify.Message{
			UserID:    userID,
			Channel:   notify.ChannelEmail,
			Recipient: email,
			Template:  notify.TemplateCodeVerification,
			Payload:   map[string]any{"link": s.verificationLink(token)},
			DedupKey:  "email_verification:" + fmt.Sprintf("%x", tokenHash),
		})
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("зарегистрировать пользователя: %w", err)
	}

	return user, nil
}

// VerifyEmail подтверждает email по одноразовому токену.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrInvalidToken
	}

	tokenHash := HashToken(token)
	now := s.clock.Now()

	return s.tx.WithTx(ctx, func(q db.Querier) error {
		verification, err := s.repo.GetEmailVerificationByHash(ctx, q, tokenHash)
		if err != nil {
			return ErrInvalidToken
		}
		if verification.ConsumedAt != nil {
			return ErrTokenUsed
		}
		if !now.Before(verification.ExpiresAt) {
			return ErrTokenExpired
		}

		if err := s.repo.ConsumeEmailVerification(ctx, q, verification.ID, now); err != nil {
			return err
		}

		return s.repo.MarkEmailVerified(ctx, q, verification.UserID, now)
	})
}

// ResendVerification повторно отправляет письмо подтверждения.
// Ответ нейтрален: отсутствие пользователя не раскрывается.
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	normalized, err := ValidateEmail(email)
	if err != nil {
		return nil
	}

	user, err := s.repo.GetUserByEmail(ctx, s.db, normalized)
	if err != nil || user.Status != StatusUnverified {
		return nil
	}

	now := s.clock.Now()

	return s.tx.WithTx(ctx, func(q db.Querier) error {
		recent, err := s.repo.HasRecentEmailVerification(ctx, q, user.ID, now.Add(-EmailResendCooldown))
		if err != nil {
			return err
		}
		if recent {
			return nil
		}

		token, tokenHash, err := NewToken()
		if err != nil {
			return err
		}
		verificationID, err := ids.New()
		if err != nil {
			return err
		}

		verification := &Verification{
			ID:        verificationID,
			UserID:    user.ID,
			TokenHash: tokenHash,
			ExpiresAt: now.Add(VerificationTTL),
			CreatedAt: now,
		}

		if err := s.repo.InvalidateEmailVerifications(ctx, q, user.ID, now); err != nil {
			return err
		}
		if err := s.repo.CreateEmailVerification(ctx, q, verification); err != nil {
			return err
		}

		return s.enqueuer.Enqueue(ctx, q, notify.Message{
			UserID:    user.ID,
			Channel:   notify.ChannelEmail,
			Recipient: user.Email,
			Template:  notify.TemplateCodeVerification,
			Payload:   map[string]any{"link": s.verificationLink(token)},
			DedupKey:  "email_verification:" + fmt.Sprintf("%x", tokenHash),
		})
	})
}

// Login проверяет данные и создаёт сессию.
func (s *Service) Login(ctx context.Context, email, password, ip, userAgent string) (*User, string, error) {
	normalized, err := ValidateEmail(email)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	now := s.clock.Now()
	failed, err := s.repo.CountFailedLoginAttempts(ctx, s.db, normalized, ip, now.Add(-LoginAttemptWindow))
	if err != nil {
		return nil, "", fmt.Errorf("проверить попытки входа: %w", err)
	}
	if failed >= MaxLoginAttempts {
		return nil, "", ErrTooManyAttempts
	}

	user, err := s.repo.GetUserByEmail(ctx, s.db, normalized)
	if err != nil {
		_, _ = VerifyPassword(password, dummyPasswordHash)
		_ = s.recordAttempt(ctx, normalized, ip, false)
		return nil, "", ErrInvalidCredentials
	}

	switch user.Status {
	case StatusUnverified:
		return nil, "", ErrEmailNotVerified
	case StatusBlocked, StatusAnonymized:
		_ = s.recordAttempt(ctx, normalized, ip, false)
		return nil, "", ErrAccountBlocked
	}

	ok, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !ok {
		_ = s.recordAttempt(ctx, normalized, ip, false)
		return nil, "", ErrInvalidCredentials
	}

	token, tokenHash, err := NewToken()
	if err != nil {
		return nil, "", err
	}
	sessionID, err := ids.New()
	if err != nil {
		return nil, "", err
	}

	attemptID, err := ids.New()
	if err != nil {
		return nil, "", err
	}

	session := &Session{
		ID:         sessionID,
		UserID:     user.ID,
		TokenHash:  tokenHash,
		UserAgent:  userAgent,
		IP:         ip,
		ExpiresAt:  now.Add(SessionDuration),
		LastSeenAt: now,
		CreatedAt:  now,
	}

	err = s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.CreateSession(ctx, q, session); err != nil {
			return err
		}
		if err := s.repo.ClearFailedLoginAttempts(ctx, q, normalized, ip); err != nil {
			return err
		}

		return s.repo.InsertLoginAttempt(ctx, q, &LoginAttempt{
			ID:        attemptID,
			Email:     normalized,
			IP:        ip,
			Success:   true,
			CreatedAt: now,
		})
	})
	if err != nil {
		return nil, "", fmt.Errorf("создать сессию: %w", err)
	}

	return user, token, nil
}

// Authenticate проверяет сессионный токен и продлевает сессию.
func (s *Service) Authenticate(ctx context.Context, token string) (*User, *Session, error) {
	if strings.TrimSpace(token) == "" {
		return nil, nil, ErrInvalidToken
	}

	session, err := s.repo.GetSessionByTokenHash(ctx, s.db, HashToken(token))
	if err != nil {
		return nil, nil, ErrInvalidToken
	}

	now := s.clock.Now()
	if session.RevokedAt != nil || !now.Before(session.ExpiresAt) {
		return nil, nil, ErrInvalidToken
	}

	user, err := s.repo.GetUserByID(ctx, s.db, session.UserID)
	if err != nil || user.Status != StatusActive {
		return nil, nil, ErrInvalidToken
	}

	expiresAt := now.Add(SessionDuration)
	if err := s.repo.TouchSession(ctx, s.db, session.ID, now, expiresAt); err != nil {
		s.logger.Warn("не удалось продлить сессию", "session_id", session.ID, "error", err)
	}
	session.LastSeenAt = now
	session.ExpiresAt = expiresAt

	return user, session, nil
}

// Logout отзывает текущую сессию.
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	now := s.clock.Now()

	return s.tx.WithTx(ctx, func(q db.Querier) error {
		return s.repo.RevokeSession(ctx, q, sessionID, now)
	})
}

// RequestPasswordReset ставит письмо сброса; ответ всегда нейтрален.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	normalized, err := ValidateEmail(email)
	if err != nil {
		return nil
	}

	user, err := s.repo.GetUserByEmail(ctx, s.db, normalized)
	if err != nil || user.Status == StatusAnonymized || user.Status == StatusBlocked {
		return nil
	}

	now := s.clock.Now()

	return s.tx.WithTx(ctx, func(q db.Querier) error {
		recent, err := s.repo.HasRecentPasswordReset(ctx, q, user.ID, now.Add(-EmailResendCooldown))
		if err != nil {
			return err
		}
		if recent {
			return nil
		}

		token, tokenHash, err := NewToken()
		if err != nil {
			return err
		}
		resetID, err := ids.New()
		if err != nil {
			return err
		}

		reset := &Verification{
			ID:        resetID,
			UserID:    user.ID,
			TokenHash: tokenHash,
			ExpiresAt: now.Add(PasswordResetTTL),
			CreatedAt: now,
		}

		if err := s.repo.InvalidatePasswordResets(ctx, q, user.ID, now); err != nil {
			return err
		}
		if err := s.repo.CreatePasswordReset(ctx, q, reset); err != nil {
			return err
		}

		return s.enqueuer.Enqueue(ctx, q, notify.Message{
			UserID:    user.ID,
			Channel:   notify.ChannelEmail,
			Recipient: user.Email,
			Template:  notify.TemplateCodePasswordReset,
			Payload:   map[string]any{"link": s.resetLink(token)},
			DedupKey:  "password_reset:" + fmt.Sprintf("%x", tokenHash),
		})
	})
}

// ResetPassword меняет пароль по одноразовому токену и отзывает сессии.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if strings.TrimSpace(token) == "" {
		return ErrInvalidToken
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	tokenHash := HashToken(token)
	now := s.clock.Now()

	return s.tx.WithTx(ctx, func(q db.Querier) error {
		reset, err := s.repo.GetPasswordResetByHash(ctx, q, tokenHash)
		if err != nil {
			return ErrInvalidToken
		}
		if reset.ConsumedAt != nil {
			return ErrTokenUsed
		}
		if !now.Before(reset.ExpiresAt) {
			return ErrTokenExpired
		}

		if err := s.repo.UpdatePasswordHash(ctx, q, reset.UserID, passwordHash, now); err != nil {
			return err
		}
		if err := s.repo.ConsumePasswordReset(ctx, q, reset.ID, now); err != nil {
			return err
		}

		return s.repo.RevokeUserSessions(ctx, q, reset.UserID, now)
	})
}

// ChangePassword меняет пароль аутентифицированного пользователя.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID uuid.UUID, currentPassword, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	user, err := s.repo.GetUserByID(ctx, s.db, userID)
	if err != nil {
		return ErrInvalidCredentials
	}

	ok, err := VerifyPassword(currentPassword, user.PasswordHash)
	if err != nil || !ok {
		return ErrInvalidCurrentPassword
	}

	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	now := s.clock.Now()

	return s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.UpdatePasswordHash(ctx, q, userID, passwordHash, now); err != nil {
			return err
		}

		return s.repo.RevokeOtherUserSessions(ctx, q, userID, sessionID, now)
	})
}

// AnonymizeAccount обезличивает учётную запись и отзывает сессии.
func (s *Service) AnonymizeAccount(ctx context.Context, userID uuid.UUID) error {
	now := s.clock.Now()

	return s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.AnonymizeUser(ctx, q, userID, now); err != nil {
			return err
		}
		if err := s.repo.InvalidateEmailVerifications(ctx, q, userID, now); err != nil {
			return err
		}
		if err := s.repo.InvalidatePasswordResets(ctx, q, userID, now); err != nil {
			return err
		}

		return s.repo.RevokeUserSessions(ctx, q, userID, now)
	})
}

// GetUser возвращает пользователя по идентификатору.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetUserByID(ctx, s.db, id)
}

// SearchUsers ищет пользователей по email или идентификатору.
func (s *Service) SearchUsers(ctx context.Context, query string, cursor uuid.UUID, limit int) ([]User, error) {
	return s.repo.SearchUsers(ctx, s.db, strings.TrimSpace(query), cursor, limit)
}

// SetUserRole меняет роль пользователя.
func (s *Service) SetUserRole(ctx context.Context, id uuid.UUID, role Role) (*User, error) {
	if !role.Valid() {
		return nil, ErrInvalidRole
	}
	if _, err := s.repo.GetUserByID(ctx, s.db, id); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	if err := s.repo.UpdateUserRole(ctx, s.db, id, role, now); err != nil {
		return nil, err
	}

	return s.repo.GetUserByID(ctx, s.db, id)
}

// SetUserStatus меняет статус пользователя и отзывает сессии при блокировке.
func (s *Service) SetUserStatus(ctx context.Context, id uuid.UUID, status Status) (*User, error) {
	if !status.Valid() {
		return nil, ErrInvalidRole
	}
	if _, err := s.repo.GetUserByID(ctx, s.db, id); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	err := s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.UpdateUserStatus(ctx, q, id, status, now); err != nil {
			return err
		}
		if status == StatusBlocked || status == StatusAnonymized {
			return s.repo.RevokeUserSessions(ctx, q, id, now)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.repo.GetUserByID(ctx, s.db, id)
}

func (s *Service) recordAttempt(ctx context.Context, email, ip string, success bool) error {
	id, err := ids.New()
	if err != nil {
		return err
	}

	return s.repo.InsertLoginAttempt(ctx, s.db, &LoginAttempt{
		ID:        id,
		Email:     email,
		IP:        ip,
		Success:   success,
		CreatedAt: s.clock.Now(),
	})
}

func (s *Service) verificationLink(token string) string {
	return s.baseURL + "/verify-email?token=" + token
}

func (s *Service) resetLink(token string) string {
	return s.baseURL + "/password/reset?token=" + token
}
