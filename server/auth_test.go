package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAuthUnconfigured(t *testing.T) {
	setupTestDB(t)
	os.Unsetenv("KB_AUTH_PASSWORD")
	os.Unsetenv("KB_POSTGRES_PASSWORD")
	defaultSessionStore.ClearAll()

	handler := RegisterRoutes()

	// 1. Status when no auth is configured
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	recStatus := httptest.NewRecorder()
	handler.ServeHTTP(recStatus, reqStatus)

	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/auth/status, got %d", recStatus.Code)
	}

	var statusRes authStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusRes); err != nil {
		t.Fatalf("failed to decode status: %v", err)
	}
	if !statusRes.Authenticated || statusRes.AuthRequired {
		t.Errorf("expected Authenticated=true, AuthRequired=false when no password set; got %+v", statusRes)
	}

	// 2. Protected routes should be accessible directly
	reqItems := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	recItems := httptest.NewRecorder()
	handler.ServeHTTP(recItems, reqItems)

	if recItems.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/items without auth when unconfigured, got %d", recItems.Code)
	}
}

func TestAuthLifecycle(t *testing.T) {
	setupTestDB(t)
	const testPass = "super_secure_kb_password_2026"
	t.Setenv("KB_AUTH_PASSWORD", testPass)
	defaultSessionStore.ClearAll()

	handler := RegisterRoutes()

	// 1. Unauthenticated request to /api/items should return 401
	reqUnauth := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	recUnauth := httptest.NewRecorder()
	handler.ServeHTTP(recUnauth, reqUnauth)

	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated /api/items, got %d: %s", recUnauth.Code, recUnauth.Body.String())
	}

	// 2. Auth status should indicate unauthenticated but required
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	recStatus := httptest.NewRecorder()
	handler.ServeHTTP(recStatus, reqStatus)

	var statusRes authStatusResponse
	json.Unmarshal(recStatus.Body.Bytes(), &statusRes)
	if statusRes.Authenticated || !statusRes.AuthRequired {
		t.Errorf("expected Authenticated=false, AuthRequired=true, got %+v", statusRes)
	}

	// 3. Login with wrong password should fail with 401
	wrongBody, _ := json.Marshal(map[string]string{"password": "wrong_password"})
	reqWrong := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(wrongBody))
	reqWrong.Header.Set("Content-Type", "application/json")
	recWrong := httptest.NewRecorder()
	handler.ServeHTTP(recWrong, reqWrong)

	if recWrong.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", recWrong.Code)
	}

	// 4. Login with correct password should succeed and set cookie
	correctBody, _ := json.Marshal(map[string]string{"password": testPass})
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(correctBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	recLogin := httptest.NewRecorder()
	handler.ServeHTTP(recLogin, reqLogin)

	if recLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 for correct login, got %d: %s", recLogin.Code, recLogin.Body.String())
	}

	cookies := recLogin.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("expected kb_session cookie to be set")
	}
	if !sessionCookie.HttpOnly {
		t.Errorf("expected session cookie to be HttpOnly")
	}

	// 5. Auth status with session cookie should return authenticated
	reqStatusAuth := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	reqStatusAuth.AddCookie(sessionCookie)
	recStatusAuth := httptest.NewRecorder()
	handler.ServeHTTP(recStatusAuth, reqStatusAuth)

	json.Unmarshal(recStatusAuth.Body.Bytes(), &statusRes)
	if !statusRes.Authenticated || !statusRes.AuthRequired {
		t.Errorf("expected Authenticated=true with cookie, got %+v", statusRes)
	}

	// 6. Access protected route with session cookie
	reqItemsAuth := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	reqItemsAuth.AddCookie(sessionCookie)
	recItemsAuth := httptest.NewRecorder()
	handler.ServeHTTP(recItemsAuth, reqItemsAuth)

	if recItemsAuth.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/items with session cookie, got %d: %s", recItemsAuth.Code, recItemsAuth.Body.String())
	}

	// 7. Logout should clear session
	reqLogout := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	reqLogout.AddCookie(sessionCookie)
	recLogout := httptest.NewRecorder()
	handler.ServeHTTP(recLogout, reqLogout)

	if recLogout.Code != http.StatusOK {
		t.Fatalf("expected 200 for logout, got %d", recLogout.Code)
	}

	// 8. Access protected route after logout should return 401
	reqAfterLogout := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	reqAfterLogout.AddCookie(sessionCookie)
	recAfterLogout := httptest.NewRecorder()
	handler.ServeHTTP(recAfterLogout, reqAfterLogout)

	if recAfterLogout.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 after logout, got %d", recAfterLogout.Code)
	}
}

func TestAuthBearerHeader(t *testing.T) {
	setupTestDB(t)
	const testPass = "bearer_token_test_pass"
	t.Setenv("KB_AUTH_PASSWORD", testPass)
	defaultSessionStore.ClearAll()

	token := defaultSessionStore.CreateSession()

	handler := RegisterRoutes()

	// Access with Bearer token header
	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with Authorization Bearer header, got %d: %s", rec.Code, rec.Body.String())
	}
}
