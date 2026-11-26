package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory  = 64 * 1024
	argonTime    = 1
	argonThreads = 4
	argonSaltLen = 16
	argonKeyLen  = 32
)

var errInvalidHash = errors.New("некорректный формат хеша пароля")

// HashPassword возвращает Argon2id-хеш пароля в формате PHC.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("сгенерировать соль пароля: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// VerifyPassword сравнивает пароль с PHC-хешем.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errInvalidHash
	}

	memory, timeCost, threads, err := parseArgonParams(parts[3])
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errInvalidHash
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, errInvalidHash
	}

	actual := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(expected)))

	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func parseArgonParams(part string) (uint32, uint32, uint8, error) {
	var (
		memory   uint32
		timeCost uint32
		threads  uint8
	)

	for _, item := range strings.Split(part, ",") {
		key, value, found := strings.Cut(item, "=")
		if !found {
			return 0, 0, 0, errInvalidHash
		}

		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return 0, 0, 0, errInvalidHash
		}

		switch key {
		case "m":
			memory = uint32(parsed)
		case "t":
			timeCost = uint32(parsed)
		case "p":
			if parsed > 255 {
				return 0, 0, 0, errInvalidHash
			}
			threads = uint8(parsed)
		}
	}

	if memory == 0 || timeCost == 0 || threads == 0 {
		return 0, 0, 0, errInvalidHash
	}

	return memory, timeCost, threads, nil
}
