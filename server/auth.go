package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/utils"
)

const (
	SessionCookieName = "kb_session"
	SessionDuration   = 30 * 24 * time.Hour // 30-day persistent session
)

type Session struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

var defaultSessionStore = &SessionStore{
	sessions: make(map[string]Session),
}

// CreateSession generates a secure random 32-byte session token and stores it.
func (s *SessionStore) CreateSession() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback timestamp + random if rand.Read encounters system error
		b = []byte(time.Now().Format(time.RFC3339Nano) + hex.EncodeToString(b))
	}
	token := hex.EncodeToString(b)
	now := time.Now()

	s.sessions[token] = Session{
		Token:     token,
		CreatedAt: now,
		ExpiresAt: now.Add(SessionDuration),
	}
	return token
}

// ValidateSession returns true if the token is present and has not expired.
func (s *SessionStore) ValidateSession(token string) bool {
	if token == "" {
		return false
	}
	s.mu.RLock()
	session, exists := s.sessions[token]
	s.mu.RUnlock()

	if !exists {
		return false
	}
	if time.Now().After(session.ExpiresAt) {
		s.DeleteSession(token)
		return false
	}
	return true
}

// DeleteSession removes a session token from the store.
func (s *SessionStore) DeleteSession(token string) {
	if token == "" {
		return
	}
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// ClearAll clears all active sessions (useful for tests or password rotation).
func (s *SessionStore) ClearAll() {
	s.mu.Lock()
	s.sessions = make(map[string]Session)
	s.mu.Unlock()
}

// GetAuthPassword returns the configured access password from environment, config, or keyring.
func GetAuthPassword() (string, bool) {
	// 1. Explicit auth password from environment
	if envPass := os.Getenv("KB_AUTH_PASSWORD"); envPass != "" {
		return strings.TrimSpace(envPass), true
	}

	// 2. Server password in viper config
	if viperPass := viper.GetString("server.password"); viperPass != "" {
		return strings.TrimSpace(viperPass), true
	}

	// 3. PostgreSQL password from environment
	if envPgPass := os.Getenv("KB_POSTGRES_PASSWORD"); envPgPass != "" {
		return strings.TrimSpace(envPgPass), true
	}

	// 4. PostgreSQL password from OS Keyring / Local Secure Vault
	if secretPass, err := utils.GetSecret("postgres_password"); err == nil && secretPass != "" {
		return strings.TrimSpace(secretPass), true
	}

	return "", false
}

// IsAuthConfigured returns true if a password has been set.
func IsAuthConfigured() bool {
	_, configured := GetAuthPassword()
	return configured
}

func isSecureRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func extractSessionToken(r *http.Request) string {
	// 1. Check HttpOnly Cookie
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 2. Check Authorization Header: Bearer <token>
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}

// requireAuth is a middleware that validates the session before executing the handler.
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no password is configured on the machine, allow public access
		if !IsAuthConfigured() {
			next(w, r)
			return
		}

		token := extractSessionToken(r)
		if !defaultSessionStore.ValidateSession(token) {
			jsonError(w, http.StatusUnauthorized, "Authentication required. Please log in.")
			return
		}

		next(w, r)
	}
}

type loginRequest struct {
	Password string `json:"password"`
}

type authStatusResponse struct {
	Authenticated bool `json:"authenticated"`
	AuthRequired  bool `json:"auth_required"`
}

// handleAuthLogin authenticates a user against the DB/access password and sets a session cookie.
func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	expectedPass, isConfigured := GetAuthPassword()
	if !isConfigured {
		// If auth not configured, grant session immediately
		token := defaultSessionStore.CreateSession()
		setSessionCookie(w, r, token)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"authenticated": true,
			"auth_required": false,
			"message":       "Authentication not required",
		})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Constant-time password comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(req.Password)), []byte(expectedPass)) != 1 {
		// Add small delay to prevent brute-force timing
		time.Sleep(100 * time.Millisecond)
		jsonError(w, http.StatusUnauthorized, "Incorrect password. Please try again.")
		return
	}

	token := defaultSessionStore.CreateSession()
	setSessionCookie(w, r, token)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"auth_required": true,
		"message":       "Login successful",
	})
}

// handleAuthStatus checks whether the current client holds an active valid session.
func handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	isConfigured := IsAuthConfigured()
	if !isConfigured {
		jsonResponse(w, http.StatusOK, authStatusResponse{
			Authenticated: true,
			AuthRequired:  false,
		})
		return
	}

	token := extractSessionToken(r)
	isValid := defaultSessionStore.ValidateSession(token)

	jsonResponse(w, http.StatusOK, authStatusResponse{
		Authenticated: isValid,
		AuthRequired:  true,
	})
}

// handleAuthLogout invalidates the active session and clears the session cookie.
func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	token := extractSessionToken(r)
	if token != "" {
		defaultSessionStore.DeleteSession(token)
	}

	clearSessionCookie(w)
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"authenticated": false,
		"message":       "Logged out successfully",
	})
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(SessionDuration.Seconds()),
		Expires:  time.Now().Add(SessionDuration),
		Secure:   isSecureRequest(r),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
