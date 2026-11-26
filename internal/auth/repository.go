package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oge-ege/internal/shared/db"
)

// Repository описывает хранилище модуля auth.
type Repository interface {
	CreateUser(ctx context.Context, q db.Querier, user *User) error
	GetUserByEmail(ctx context.Context, q db.Querier, email string) (*User, error)
	GetUserByID(ctx context.Context, q db.Querier, id uuid.UUID) (*User, error)
	MarkEmailVerified(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time) error
	UpdatePasswordHash(ctx context.Context, q db.Querier, id uuid.UUID, hash string, now time.Time) error
	AnonymizeUser(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time) error

	CreateEmailVerification(ctx context.Context, q db.Querier, verification *Verification) error
	GetEmailVerificationByHash(ctx context.Context, q db.Querier, tokenHash []byte) (*Verification, error)
	ConsumeEmailVerification(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time) error
	InvalidateEmailVerifications(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) error
	HasRecentEmailVerification(ctx context.Context, q db.Querier, userID uuid.UUID, since time.Time) (bool, error)

	CreatePasswordReset(ctx context.Context, q db.Querier, reset *Verification) error
	GetPasswordResetByHash(ctx context.Context, q db.Querier, tokenHash []byte) (*Verification, error)
	ConsumePasswordReset(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time) error
	InvalidatePasswordResets(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) error
	HasRecentPasswordReset(ctx context.Context, q db.Querier, userID uuid.UUID, since time.Time) (bool, error)

	InsertLoginAttempt(ctx context.Context, q db.Querier, attempt *LoginAttempt) error
	CountFailedLoginAttempts(ctx context.Context, q db.Querier, email, ip string, since time.Time) (int, error)
	ClearFailedLoginAttempts(ctx context.Context, q db.Querier, email, ip string) error

	CreateSession(ctx context.Context, q db.Querier, session *Session) error
	GetSessionByTokenHash(ctx context.Context, q db.Querier, tokenHash []byte) (*Session, error)
	TouchSession(ctx context.Context, q db.Querier, id uuid.UUID, lastSeenAt, expiresAt time.Time) error
	RevokeSession(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time) error
	RevokeUserSessions(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) error
	RevokeOtherUserSessions(ctx context.Context, q db.Querier, userID, keepSessionID uuid.UUID, now time.Time) error
}

const createUserQuery = `
INSERT INTO auth.users (id, email, password_hash, role, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $6)
`

const userColumns = `
    id,
    email,
    password_hash,
    role,
    status,
    email_verified_at,
    created_at,
    updated_at
`

const getUserByEmailQuery = `
SELECT ` + userColumns + `
FROM auth.users
WHERE lower(email) = lower($1)
`

const getUserByIDQuery = `
SELECT ` + userColumns + `
FROM auth.users
WHERE id = $1
`

const markEmailVerifiedQuery = `
UPDATE auth.users
SET status = 'active', email_verified_at = $2, updated_at = $2
WHERE id = $1
`

const updatePasswordHashQuery = `
UPDATE auth.users
SET password_hash = $2, updated_at = $3
WHERE id = $1
`

const anonymizeUserQuery = `
UPDATE auth.users
SET email = $2, password_hash = '!', status = 'anonymized', updated_at = $3
WHERE id = $1
`

const createEmailVerificationQuery = `
INSERT INTO auth.email_verifications (id, user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5)
`

const getEmailVerificationByHashQuery = `
SELECT id, user_id, token_hash, expires_at, consumed_at, created_at
FROM auth.email_verifications
WHERE token_hash = $1
`

const consumeEmailVerificationQuery = `
UPDATE auth.email_verifications
SET consumed_at = $2
WHERE id = $1 AND consumed_at IS NULL
`

const invalidateEmailVerificationsQuery = `
UPDATE auth.email_verifications
SET consumed_at = $2
WHERE user_id = $1 AND consumed_at IS NULL
`

const hasRecentEmailVerificationQuery = `
SELECT EXISTS (
    SELECT 1
    FROM auth.email_verifications
    WHERE user_id = $1 AND created_at > $2
)
`

const createPasswordResetQuery = `
INSERT INTO auth.password_resets (id, user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5)
`

const hasRecentPasswordResetQuery = `
SELECT EXISTS (
    SELECT 1
    FROM auth.password_resets
    WHERE user_id = $1 AND created_at > $2
)
`

const getPasswordResetByHashQuery = `
SELECT id, user_id, token_hash, expires_at, consumed_at, created_at
FROM auth.password_resets
WHERE token_hash = $1
`

const consumePasswordResetQuery = `
UPDATE auth.password_resets
SET consumed_at = $2
WHERE id = $1 AND consumed_at IS NULL
`

const invalidatePasswordResetsQuery = `
UPDATE auth.password_resets
SET consumed_at = $2
WHERE user_id = $1 AND consumed_at IS NULL
`

const insertLoginAttemptQuery = `
INSERT INTO auth.login_attempts (id, email, ip, success, created_at)
VALUES ($1, $2, NULLIF($3, '')::inet, $4, $5)
`

const countFailedLoginAttemptsQuery = `
SELECT count(*)
FROM auth.login_attempts
WHERE lower(email) = lower($1)
  AND ip IS NOT DISTINCT FROM NULLIF($2, '')::inet
  AND success = false
  AND created_at >= $3
`

const clearFailedLoginAttemptsQuery = `
DELETE FROM auth.login_attempts
WHERE lower(email) = lower($1)
  AND ip IS NOT DISTINCT FROM NULLIF($2, '')::inet
  AND success = false
`

const createSessionQuery = `
INSERT INTO auth.sessions (id, user_id, token_hash, user_agent, ip, expires_at, last_seen_at, created_at)
VALUES ($1, $2, $3, $4, NULLIF($5, '')::inet, $6, $7, $7)
`

const getSessionByTokenHashQuery = `
SELECT
    id,
    user_id,
    token_hash,
    COALESCE(user_agent, ''),
    COALESCE(host(ip), ''),
    expires_at,
    revoked_at,
    last_seen_at,
    created_at
FROM auth.sessions
WHERE token_hash = $1
`

const touchSessionQuery = `
UPDATE auth.sessions
SET last_seen_at = $2, expires_at = $3
WHERE id = $1
`

const revokeSessionQuery = `
UPDATE auth.sessions
SET revoked_at = $2
WHERE id = $1 AND revoked_at IS NULL
`

const revokeUserSessionsQuery = `
UPDATE auth.sessions
SET revoked_at = $2
WHERE user_id = $1 AND revoked_at IS NULL
`

const revokeOtherUserSessionsQuery = `
UPDATE auth.sessions
SET revoked_at = $3
WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL
`

// PgRepository работает с PostgreSQL.
type PgRepository struct{}

// NewPgRepository создаёт репозиторий auth.
func NewPgRepository() *PgRepository {
	return &PgRepository{}
}

// CreateUser сохраняет учётную запись.
func (r *PgRepository) CreateUser(ctx context.Context, q db.Querier, user *User) error {
	if _, err := q.Exec(ctx, createUserQuery,
		user.ID, user.Email, user.PasswordHash, string(user.Role), string(user.Status), user.CreatedAt,
	); err != nil {
		return fmt.Errorf("создать пользователя: %w", err)
	}

	return nil
}

// GetUserByEmail ищет пользователя по email без учёта регистра.
func (r *PgRepository) GetUserByEmail(ctx context.Context, q db.Querier, email string) (*User, error) {
	return scanUser(q.QueryRow(ctx, getUserByEmailQuery, email))
}

// GetUserByID ищет пользователя по идентификатору.
func (r *PgRepository) GetUserByID(ctx context.Context, q db.Querier, id uuid.UUID) (*User, error) {
	return scanUser(q.QueryRow(ctx, getUserByIDQuery, id))
}

// MarkEmailVerified переводит пользователя в активный статус.
func (r *PgRepository) MarkEmailVerified(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time) error {
	if _, err := q.Exec(ctx, markEmailVerifiedQuery, id, now); err != nil {
		return fmt.Errorf("подтвердить email: %w", err)
	}

	return nil
}

// UpdatePasswordHash обновляет хеш пароля.
func (r *PgRepository) UpdatePasswordHash(ctx context.Context, q db.Querier, id uuid.UUID, hash string, now time.Time) error {
	if _, err := q.Exec(ctx, updatePasswordHashQuery, id, hash, now); err != nil {
		return fmt.Errorf("обновить пароль: %w", err)
	}

	return nil
}

// AnonymizeUser обезличивает учётную запись.
func (r *PgRepository) AnonymizeUser(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time) error {
	email := fmt.Sprintf("deleted+%s@anonymized.invalid", id)
	if _, err := q.Exec(ctx, anonymizeUserQuery, id, email, now); err != nil {
		return fmt.Errorf("обезличить пользователя: %w", err)
	}

	return nil
}

// CreateEmailVerification сохраняет токен подтверждения.
func (r *PgRepository) CreateEmailVerification(ctx context.Context, q db.Querier, verification *Verification) error {
	if _, err := q.Exec(ctx, createEmailVerificationQuery,
		verification.ID, verification.UserID, verification.TokenHash, verification.ExpiresAt, verification.CreatedAt,
	); err != nil {
		return fmt.Errorf("создать токен подтверждения: %w", err)
	}

	return nil
}

// GetEmailVerificationByHash ищет токен подтверждения по хешу.
func (r *PgRepository) GetEmailVerificationByHash(ctx context.Context, q db.Querier, tokenHash []byte) (*Verification, error) {
	return scanVerification(q.QueryRow(ctx, getEmailVerificationByHashQuery, tokenHash))
}

// ConsumeEmailVerification помечает токен использованным.
func (r *PgRepository) ConsumeEmailVerification(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time) error {
	if _, err := q.Exec(ctx, consumeEmailVerificationQuery, id, now); err != nil {
		return fmt.Errorf("использовать токен подтверждения: %w", err)
	}

	return nil
}

// InvalidateEmailVerifications инвалидирует прежние токены подтверждения.
func (r *PgRepository) InvalidateEmailVerifications(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) error {
	if _, err := q.Exec(ctx, invalidateEmailVerificationsQuery, userID, now); err != nil {
		return fmt.Errorf("инвалидировать токены подтверждения: %w", err)
	}

	return nil
}

// HasRecentEmailVerification сообщает, отправлялось ли письмо подтверждения после since.
func (r *PgRepository) HasRecentEmailVerification(ctx context.Context, q db.Querier, userID uuid.UUID, since time.Time) (bool, error) {
	var recent bool
	if err := q.QueryRow(ctx, hasRecentEmailVerificationQuery, userID, since).Scan(&recent); err != nil {
		return false, fmt.Errorf("проверить недавнее письмо подтверждения: %w", err)
	}

	return recent, nil
}

// CreatePasswordReset сохраняет токен сброса пароля.
func (r *PgRepository) CreatePasswordReset(ctx context.Context, q db.Querier, reset *Verification) error {
	if _, err := q.Exec(ctx, createPasswordResetQuery,
		reset.ID, reset.UserID, reset.TokenHash, reset.ExpiresAt, reset.CreatedAt,
	); err != nil {
		return fmt.Errorf("создать токен сброса пароля: %w", err)
	}

	return nil
}

// GetPasswordResetByHash ищет токен сброса по хешу.
func (r *PgRepository) GetPasswordResetByHash(ctx context.Context, q db.Querier, tokenHash []byte) (*Verification, error) {
	return scanVerification(q.QueryRow(ctx, getPasswordResetByHashQuery, tokenHash))
}

// ConsumePasswordReset помечает токен сброса использованным.
func (r *PgRepository) ConsumePasswordReset(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time) error {
	if _, err := q.Exec(ctx, consumePasswordResetQuery, id, now); err != nil {
		return fmt.Errorf("использовать токен сброса: %w", err)
	}

	return nil
}

// InvalidatePasswordResets инвалидирует прежние токены сброса.
func (r *PgRepository) InvalidatePasswordResets(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) error {
	if _, err := q.Exec(ctx, invalidatePasswordResetsQuery, userID, now); err != nil {
		return fmt.Errorf("инвалидировать токены сброса: %w", err)
	}

	return nil
}

// HasRecentPasswordReset сообщает, отправлялось ли письмо сброса после since.
func (r *PgRepository) HasRecentPasswordReset(ctx context.Context, q db.Querier, userID uuid.UUID, since time.Time) (bool, error) {
	var recent bool
	if err := q.QueryRow(ctx, hasRecentPasswordResetQuery, userID, since).Scan(&recent); err != nil {
		return false, fmt.Errorf("проверить недавнее письмо сброса: %w", err)
	}

	return recent, nil
}

// InsertLoginAttempt сохраняет попытку входа.
func (r *PgRepository) InsertLoginAttempt(ctx context.Context, q db.Querier, attempt *LoginAttempt) error {
	if _, err := q.Exec(ctx, insertLoginAttemptQuery,
		attempt.ID, attempt.Email, attempt.IP, attempt.Success, attempt.CreatedAt,
	); err != nil {
		return fmt.Errorf("сохранить попытку входа: %w", err)
	}

	return nil
}

// CountFailedLoginAttempts считает неудачи за период.
func (r *PgRepository) CountFailedLoginAttempts(ctx context.Context, q db.Querier, email, ip string, since time.Time) (int, error) {
	var count int
	if err := q.QueryRow(ctx, countFailedLoginAttemptsQuery, email, ip, since).Scan(&count); err != nil {
		return 0, fmt.Errorf("посчитать неудачные попытки входа: %w", err)
	}

	return count, nil
}

// ClearFailedLoginAttempts сбрасывает счётчик неудач.
func (r *PgRepository) ClearFailedLoginAttempts(ctx context.Context, q db.Querier, email, ip string) error {
	if _, err := q.Exec(ctx, clearFailedLoginAttemptsQuery, email, ip); err != nil {
		return fmt.Errorf("сбросить неудачные попытки входа: %w", err)
	}

	return nil
}

// CreateSession сохраняет сессию.
func (r *PgRepository) CreateSession(ctx context.Context, q db.Querier, session *Session) error {
	if _, err := q.Exec(ctx, createSessionQuery,
		session.ID, session.UserID, session.TokenHash, session.UserAgent, session.IP,
		session.ExpiresAt, session.LastSeenAt,
	); err != nil {
		return fmt.Errorf("создать сессию: %w", err)
	}

	return nil
}

// GetSessionByTokenHash ищет сессию по хешу токена.
func (r *PgRepository) GetSessionByTokenHash(ctx context.Context, q db.Querier, tokenHash []byte) (*Session, error) {
	var session Session
	err := q.QueryRow(ctx, getSessionByTokenHashQuery, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.UserAgent,
		&session.IP,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.LastSeenAt,
		&session.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("сессия не найдена")
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать сессию: %w", err)
	}

	return &session, nil
}

// TouchSession продлевает сессию при активности.
func (r *PgRepository) TouchSession(ctx context.Context, q db.Querier, id uuid.UUID, lastSeenAt, expiresAt time.Time) error {
	if _, err := q.Exec(ctx, touchSessionQuery, id, lastSeenAt, expiresAt); err != nil {
		return fmt.Errorf("продлить сессию: %w", err)
	}

	return nil
}

// RevokeSession отзывает сессию.
func (r *PgRepository) RevokeSession(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time) error {
	if _, err := q.Exec(ctx, revokeSessionQuery, id, now); err != nil {
		return fmt.Errorf("отозвать сессию: %w", err)
	}

	return nil
}

// RevokeUserSessions отзывает все сессии пользователя.
func (r *PgRepository) RevokeUserSessions(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) error {
	if _, err := q.Exec(ctx, revokeUserSessionsQuery, userID, now); err != nil {
		return fmt.Errorf("отозвать сессии пользователя: %w", err)
	}

	return nil
}

// RevokeOtherUserSessions отзывает все сессии, кроме текущей.
func (r *PgRepository) RevokeOtherUserSessions(ctx context.Context, q db.Querier, userID, keepSessionID uuid.UUID, now time.Time) error {
	if _, err := q.Exec(ctx, revokeOtherUserSessionsQuery, userID, keepSessionID, now); err != nil {
		return fmt.Errorf("отозвать прочие сессии: %w", err)
	}

	return nil
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(row userScanner) (*User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.EmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("пользователь не найден")
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать пользователя: %w", err)
	}

	return &user, nil
}

func scanVerification(row userScanner) (*Verification, error) {
	var verification Verification
	err := row.Scan(
		&verification.ID,
		&verification.UserID,
		&verification.TokenHash,
		&verification.ExpiresAt,
		&verification.ConsumedAt,
		&verification.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("токен не найден")
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать токен: %w", err)
	}

	return &verification, nil
}
