package middleware

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testSecret() []byte {
	b := make([]byte, 32)
	rand.Read(b)
	return b
}

func TestSessionAuth_UnauthenticatedAPI_Returns401(t *testing.T) {
	secret := testSecret()
	handler := SessionAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/gateway/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestSessionAuth_NonAPIPath_PassThrough(t *testing.T) {
	secret := testSecret()
	handler := SessionAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestSessionAuth_ExemptPaths_PassThrough(t *testing.T) {
	secret := testSecret()
	handler := SessionAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	for _, tc := range []struct {
		method, path string
	}{
		{"POST", "/api/auth/login"},
		{"GET", "/api/auth/status"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("%s %s: expected 200, got %d", tc.method, tc.path, rec.Code)
		}
	}
}

func TestSessionAuth_ValidCookie_PassThrough(t *testing.T) {
	secret := testSecret()
	handler := SessionAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	cookie := CreateSessionCookie("admin", secret, 1*time.Hour, false)
	req := httptest.NewRequest("GET", "/api/gateway/status", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestSessionAuth_ExpiredCookie_Returns401(t *testing.T) {
	secret := testSecret()
	handler := SessionAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	cookie := CreateSessionCookie("admin", secret, -1*time.Hour, false)
	req := httptest.NewRequest("GET", "/api/gateway/status", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestSessionAuth_TamperedCookie_Returns401(t *testing.T) {
	secret := testSecret()
	handler := SessionAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	cookie := CreateSessionCookie("admin", secret, 1*time.Hour, false)
	cookie.Value = cookie.Value + "tampered"
	req := httptest.NewRequest("GET", "/api/gateway/status", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestUsernameFromSession(t *testing.T) {
	secret := testSecret()
	cookie := CreateSessionCookie("testuser", secret, 1*time.Hour, false)
	username, ok := UsernameFromSession(cookie.Value, secret)
	if !ok || username != "testuser" {
		t.Errorf("expected testuser, got %q (ok=%v)", username, ok)
	}
}
