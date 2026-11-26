package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const tokenBytes = 32

// NewToken генерирует непрозрачный токен и его SHA-256 хеш.
func NewToken() (string, []byte, error) {
	random := make([]byte, tokenBytes)
	if _, err := rand.Read(random); err != nil {
		return "", nil, fmt.Errorf("сгенерировать токен: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(random)

	return token, HashToken(token), nil
}

// HashToken возвращает SHA-256 хеш токена для хранения.
func HashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))

	return hash[:]
}
