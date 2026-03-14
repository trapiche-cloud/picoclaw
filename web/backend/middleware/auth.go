package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	sessionCookieName = "picoclaw_session"
	sessionTTL        = 24 * time.Hour
)

type sessionPayload struct {
	Username string `json:"u"`
	Expiry   int64  `json:"e"`
}

// SessionAuth enforces cookie-based authentication on API routes.
// Non-API paths and exempt API paths are passed through.
func SessionAuth(cookieSecret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Non-API paths always pass through (SPA needs to load)
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Exempt auth endpoints
		if isAuthExempt(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Validate session cookie
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || !validateSession(cookie.Value, cookieSecret) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"authentication required"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAuthExempt(r *http.Request) bool {
	path := r.URL.Path
	if path == "/api/auth/login" && r.Method == http.MethodPost {
		return true
	}
	if path == "/api/auth/status" && r.Method == http.MethodGet {
		return true
	}
	return false
}

// CreateSessionCookie creates a signed session cookie.
func CreateSessionCookie(username string, secret []byte, ttl time.Duration, secure bool) *http.Cookie {
	payload := sessionPayload{
		Username: username,
		Expiry:   time.Now().Add(ttl).Unix(),
	}
	data, _ := json.Marshal(payload)
	sig := signData(data, secret)
	value := base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(sig)

	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	}
}

// ClearSessionCookie returns a cookie that clears the session.
func ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
}

// UsernameFromSession extracts the username from a valid session cookie value.
func UsernameFromSession(cookieValue string, secret []byte) (string, bool) {
	payload, ok := decodeAndVerify(cookieValue, secret)
	if !ok {
		return "", false
	}
	return payload.Username, true
}

func validateSession(cookieValue string, secret []byte) bool {
	payload, ok := decodeAndVerify(cookieValue, secret)
	if !ok {
		return false
	}
	return time.Now().Unix() < payload.Expiry
}

func decodeAndVerify(cookieValue string, secret []byte) (sessionPayload, bool) {
	var p sessionPayload
	parts := strings.SplitN(cookieValue, ".", 2)
	if len(parts) != 2 {
		return p, false
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return p, false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return p, false
	}

	expected := signData(data, secret)
	if !hmac.Equal(sig, expected) {
		return p, false
	}

	if err := json.Unmarshal(data, &p); err != nil {
		return p, false
	}
	return p, true
}

func signData(data, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return mac.Sum(nil)
}

// SessionTTL returns the default session TTL for external use.
func SessionTTL() time.Duration {
	return sessionTTL
}

// FormatSessionInfo returns a display string about the session cookie name.
func FormatSessionInfo() string {
	return fmt.Sprintf("cookie=%s ttl=%s", sessionCookieName, sessionTTL)
}
