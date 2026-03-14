package launcherconfig

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash of the given password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a bcrypt hash with a plaintext password.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateCookieSecret returns a hex-encoded 32-byte random key.
func GenerateCookieSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateRandomPassword returns a 16-character alphanumeric password.
func GenerateRandomPassword() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[n.Int64()]
	}
	return string(b)
}

// EnsureAuthBootstrapped populates empty auth fields with generated defaults.
// Returns the generated plaintext password and whether the config was changed.
func EnsureAuthBootstrapped(cfg *Config) (generatedPassword string, changed bool) {
	if cfg.AuthUsername != "" && cfg.AuthPasswordHash != "" && cfg.AuthCookieSecret != "" {
		return "", false
	}

	if cfg.AuthUsername == "" {
		cfg.AuthUsername = "admin"
	}

	password := ""
	if cfg.AuthPasswordHash == "" {
		password = GenerateRandomPassword()
		hash, err := HashPassword(password)
		if err != nil {
			return "", false
		}
		cfg.AuthPasswordHash = hash
	}

	if cfg.AuthCookieSecret == "" {
		secret, err := GenerateCookieSecret()
		if err != nil {
			return "", false
		}
		cfg.AuthCookieSecret = secret
	}

	cfg.AuthEnabled = true
	return password, true
}
