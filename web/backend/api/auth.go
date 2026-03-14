package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
	"github.com/sipeed/picoclaw/web/backend/middleware"
)

func (h *Handler) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", h.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", h.handleLogout)
	mux.HandleFunc("GET /api/auth/status", h.handleAuthStatus)
	mux.HandleFunc("PUT /api/auth/password", h.handleChangePassword)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	cfg, err := h.loadLauncherConfig()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	if req.Username != cfg.AuthUsername || !launcherconfig.CheckPassword(cfg.AuthPasswordHash, req.Password) {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	secret, err := hex.DecodeString(cfg.AuthCookieSecret)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	secure := r.TLS != nil
	cookie := middleware.CreateSessionCookie(req.Username, secret, middleware.SessionTTL(), secure)
	http.SetCookie(w, cookie)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"authenticated": true})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, middleware.ClearSessionCookie())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
}

func (h *Handler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.loadLauncherConfig()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	authenticated := false
	if cfg.AuthEnabled {
		secret, err := hex.DecodeString(cfg.AuthCookieSecret)
		if err == nil {
			cookie, err := r.Cookie("picoclaw_session")
			if err == nil {
				_, authenticated = middleware.UsernameFromSession(cookie.Value, secret)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"authenticated": authenticated,
		"auth_enabled":  cfg.AuthEnabled,
	})
}

func (h *Handler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.NewPassword == "" {
		http.Error(w, `{"error":"new password required"}`, http.StatusBadRequest)
		return
	}

	cfg, err := h.loadLauncherConfig()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	if !launcherconfig.CheckPassword(cfg.AuthPasswordHash, req.CurrentPassword) {
		http.Error(w, `{"error":"current password is incorrect"}`, http.StatusUnauthorized)
		return
	}

	hash, err := launcherconfig.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	cfg.AuthPasswordHash = hash
	if err := launcherconfig.Save(h.launcherConfigPath(), cfg); err != nil {
		http.Error(w, `{"error":"failed to save config"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"updated": true})
}
