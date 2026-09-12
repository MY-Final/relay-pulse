package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"monitor/internal/config"
)

func newAdminAuthTestRouter(t *testing.T) (*gin.Engine, *Handler) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{
		config: &config.AppConfig{
			Admin: config.AdminConfig{
				Enabled:            true,
				Username:           "admin",
				PasswordHash:       string(hash),
				SessionSecret:      "test-session-secret",
				SessionTTLDuration: time.Hour,
			},
			Onboarding: config.OnboardingConfig{AdminToken: "legacy-token"},
		},
		adminLimiter:  newAdminLoginLimiter(),
		adminSessions: newAdminSessionRevocations(),
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/admin/login", h.AdminLogin)
	r.GET("/api/admin/me", h.AdminMe)
	r.POST("/api/admin/logout", h.AdminLogout)
	r.GET("/api/admin/protected", func(c *gin.Context) {
		if !h.checkAdminToken(c) {
			return
		}
		c.Status(http.StatusNoContent)
	})
	r.POST("/api/admin/protected", func(c *gin.Context) {
		if !h.checkAdminToken(c) {
			return
		}
		c.Status(http.StatusNoContent)
	})
	return r, h
}

func loginAdminTest(t *testing.T, r http.Handler) (*http.Cookie, string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(`{"username":"admin","password":"correct-password"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("login Cache-Control = %q, want no-store", got)
	}
	var body struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.CSRF == "" {
		t.Fatal("login response missing csrf token")
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != adminSessionCookie {
		t.Fatalf("login response missing session cookie: %#v", cookies)
	}
	return cookies[0], body.CSRF
}

func TestAdminLoginSessionAndCSRF(t *testing.T) {
	r, _ := newAdminAuthTestRouter(t)
	cookie, csrf := loginAdminTest(t, r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	req.AddCookie(cookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"authenticated":true`) {
		t.Fatalf("me status = %d, body = %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("me Cache-Control = %q, want no-store", got)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/admin/protected", nil)
	req.AddCookie(cookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d, body = %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/admin/protected", nil)
	req.AddCookie(cookie)
	req.Header.Set("X-CSRF-Token", csrf)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("valid CSRF status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAdminLoginRejectsWrongCredentialsUniformly(t *testing.T) {
	r, _ := newAdminAuthTestRouter(t)
	for _, body := range []string{
		`{"username":"admin","password":"wrong"}`,
		`{"username":"other","password":"wrong"}`,
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "用户名或密码错误") {
			t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
		}
	}
}

func TestAdminSessionTamperAndExpiryAreRejected(t *testing.T) {
	r, h := newAdminAuthTestRouter(t)
	cookie, _ := loginAdminTest(t, r)

	tampered := *cookie
	tampered.Value = "x" + tampered.Value[1:]
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	req.AddCookie(&tampered)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tampered session status = %d, body = %s", w.Code, w.Body.String())
	}

	expired, err := h.signAdminSession(adminSessionClaims{
		Username: "admin",
		IssuedAt: time.Now().Add(-2 * time.Hour).Unix(),
		Expires:  time.Now().Add(-time.Minute).Unix(),
		CSRF:     "expired-csrf",
	}, h.config.Admin.SessionSecret)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	req.AddCookie(&http.Cookie{Name: adminSessionCookie, Value: expired})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestAdminLogoutRevokesSession(t *testing.T) {
	r, _ := newAdminAuthTestRouter(t)
	cookie, csrf := loginAdminTest(t, r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/logout", nil)
	req.AddCookie(cookie)
	req.Header.Set("X-CSRF-Token", csrf)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("logout status = %d, body = %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	req.AddCookie(cookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestLegacyBearerTokenRemainsCompatible(t *testing.T) {
	r, _ := newAdminAuthTestRouter(t)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/admin/protected", nil)
		req.Header.Set("Authorization", "Bearer legacy-token")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("legacy %s status = %d, body = %s", method, w.Code, w.Body.String())
		}
	}
}
