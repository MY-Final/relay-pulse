package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"monitor/internal/config"
)

const adminSessionCookie = "relaypulse_admin_session"

type adminSessionClaims struct {
	Username string `json:"username"`
	IssuedAt int64  `json:"iat"`
	Expires  int64  `json:"exp"`
	CSRF     string `json:"csrf"`
}

type adminLoginLimiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
}

// adminSessionRevocations 记录本进程内主动登出的会话。会话主体仍是无状态 Cookie，
// 这里只保留撤销指纹，不新增数据库表；服务重启后旧 Cookie 仍会按签名和过期时间自然失效。
type adminSessionRevocations struct {
	mu      sync.Mutex
	entries map[[sha256.Size]byte]time.Time
}

func newAdminSessionRevocations() *adminSessionRevocations {
	return &adminSessionRevocations{entries: make(map[[sha256.Size]byte]time.Time)}
}

func (r *adminSessionRevocations) revoke(raw string, expires time.Time, now time.Time) {
	if r == nil || raw == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, deadline := range r.entries {
		if !deadline.After(now) {
			delete(r.entries, key)
		}
	}
	if expires.After(now) {
		r.entries[sha256.Sum256([]byte(raw))] = expires
	}
}

func (r *adminSessionRevocations) contains(raw string, now time.Time) bool {
	if r == nil || raw == "" {
		return false
	}
	key := sha256.Sum256([]byte(raw))
	r.mu.Lock()
	defer r.mu.Unlock()
	deadline, ok := r.entries[key]
	if ok && !deadline.After(now) {
		delete(r.entries, key)
		return false
	}
	return ok
}

func newAdminLoginLimiter() *adminLoginLimiter {
	return &adminLoginLimiter{entries: make(map[string][]time.Time)}
}

func (l *adminLoginLimiter) allow(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-15 * time.Minute)
	items := l.entries[ip]
	kept := items[:0]
	for _, item := range items {
		if item.After(cutoff) {
			kept = append(kept, item)
		}
	}
	if len(kept) >= 5 {
		l.entries[ip] = kept
		return false
	}
	l.entries[ip] = append(kept, now)
	return true
}

func (l *adminLoginLimiter) reset(ip string) {
	l.mu.Lock()
	delete(l.entries, ip)
	l.mu.Unlock()
}

func (h *Handler) adminConfigSnapshot() config.AdminConfig {
	h.cfgMu.RLock()
	defer h.cfgMu.RUnlock()
	if h.config == nil {
		return config.AdminConfig{}
	}
	return h.config.Admin
}

// AdminLogin 使用配置中的 bcrypt 哈希创建无状态签名会话。
func (h *Handler) AdminLogin(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	cfg := h.adminConfigSnapshot()
	if !cfg.Enabled {
		apiError(c, http.StatusServiceUnavailable, ErrCodeServiceUnavailable, "管理员账号登录未启用")
		return
	}

	if h.adminLimiter != nil && !h.adminLimiter.allow(c.ClientIP(), time.Now()) {
		apiError(c, http.StatusTooManyRequests, ErrCodeRateLimited, "登录尝试过于频繁，请稍后重试")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "用户名或密码错误")
		return
	}
	usernameMatch := subtle.ConstantTimeCompare([]byte(strings.TrimSpace(req.Username)), []byte(cfg.Username)) == 1
	passwordErr := bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(req.Password))
	if !usernameMatch || passwordErr != nil {
		apiError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "用户名或密码错误")
		return
	}

	if h.adminLimiter != nil {
		h.adminLimiter.reset(c.ClientIP())
	}
	ttl := cfg.SessionTTLDuration
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	now := time.Now()
	csrf, err := randomAdminToken(32)
	if err != nil {
		apiError(c, http.StatusInternalServerError, ErrCodeInternalError, "创建登录会话失败")
		return
	}
	claims := adminSessionClaims{
		Username: cfg.Username,
		IssuedAt: now.Unix(),
		Expires:  now.Add(ttl).Unix(),
		CSRF:     csrf,
	}
	session, err := h.signAdminSession(claims, cfg.SessionSecret)
	if err != nil {
		apiError(c, http.StatusInternalServerError, ErrCodeInternalError, "创建登录会话失败")
		return
	}

	h.setAdminSessionCookie(c, session, int(ttl.Seconds()))
	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"username":      claims.Username,
		"csrf_token":    claims.CSRF,
	})
}

// AdminMe 返回当前登录状态，供页面刷新后恢复会话。
func (h *Handler) AdminMe(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	mode, claims, ok := h.authenticateAdmin(c, false)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"username":      claims.Username,
		"auth_mode":     mode,
		"csrf_token":    claims.CSRF,
	})
}

// AdminLogout 清除浏览器中的管理员会话 Cookie。
func (h *Handler) AdminLogout(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	raw, _ := c.Cookie(adminSessionCookie)
	_, claims, ok := h.authenticateAdmin(c, true)
	if !ok {
		return
	}
	if raw != "" && claims != nil && claims.Expires > 0 && h.adminSessions != nil {
		h.adminSessions.revoke(raw, time.Unix(claims.Expires, 0), time.Now())
	}
	cookie := &http.Cookie{
		Name:     adminSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   adminCookieSecure(c),
	}
	http.SetCookie(c.Writer, cookie)
	c.JSON(http.StatusOK, gin.H{"authenticated": false})
}

// checkAdminToken 保留旧方法名以减少现有管理接口改动；实际验证同时支持
// HttpOnly 会话 Cookie 和旧 Bearer Token。
func (h *Handler) checkAdminToken(c *gin.Context) bool {
	_, _, ok := h.authenticateAdmin(c, true)
	return ok
}

func (h *Handler) authenticateAdmin(c *gin.Context, checkCSRF bool) (string, *adminSessionClaims, bool) {
	cfg := h.adminConfigSnapshot()

	if cfg.Enabled {
		if raw, err := c.Cookie(adminSessionCookie); err == nil && raw != "" {
			claims, valid := h.verifyAdminSession(raw, cfg)
			if valid {
				if h.adminSessions != nil && h.adminSessions.contains(raw, time.Now()) {
					apiError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "管理员会话无效")
					return "", nil, false
				}
				if claims.Username != cfg.Username {
					apiError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "管理员会话无效")
					return "", nil, false
				}
				if checkCSRF && isAdminMutation(c.Request.Method) &&
					subtle.ConstantTimeCompare([]byte(c.GetHeader("X-CSRF-Token")), []byte(claims.CSRF)) != 1 {
					apiError(c, http.StatusForbidden, ErrCodeForbidden, "缺少有效的 CSRF token")
					return "", nil, false
				}
				return "session", &claims, true
			}
		}
	}

	adminToken := ""
	h.cfgMu.RLock()
	if h.config != nil {
		adminToken = h.config.Onboarding.AdminToken
	}
	h.cfgMu.RUnlock()
	if adminToken == "" {
		if cfg.Enabled {
			apiError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "需要管理员登录")
		} else {
			apiError(c, http.StatusServiceUnavailable, ErrCodeServiceUnavailable, "管理后台暂不可用")
		}
		return "", nil, false
	}

	authHeader := c.GetHeader("Authorization")
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		apiError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "需要管理员登录")
		return "", nil, false
	}
	token := strings.TrimPrefix(authHeader, bearerPrefix)
	if subtle.ConstantTimeCompare([]byte(token), []byte(adminToken)) != 1 {
		apiError(c, http.StatusForbidden, ErrCodeForbidden, "管理员 token 无效")
		return "", nil, false
	}
	return "bearer", &adminSessionClaims{Username: cfg.Username}, true
}

func isAdminMutation(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func randomAdminToken(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func (h *Handler) signAdminSession(claims adminSessionClaims, secret string) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(encoded))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + signature, nil
}

func (h *Handler) verifyAdminSession(raw string, cfg config.AdminConfig) (adminSessionClaims, bool) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || cfg.SessionSecret == "" {
		return adminSessionClaims{}, false
	}
	mac := hmac.New(sha256.New, []byte(cfg.SessionSecret))
	_, _ = mac.Write([]byte(parts[0]))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || subtle.ConstantTimeCompare(got, want) != 1 {
		return adminSessionClaims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return adminSessionClaims{}, false
	}
	var claims adminSessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Username == "" || claims.CSRF == "" {
		return adminSessionClaims{}, false
	}
	if claims.Expires <= time.Now().Unix() || claims.IssuedAt > time.Now().Add(2*time.Minute).Unix() {
		return adminSessionClaims{}, false
	}
	return claims, true
}

func adminCookieSecure(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https")
}

func (h *Handler) setAdminSessionCookie(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   adminCookieSecure(c),
		SameSite: http.SameSiteLaxMode,
	})
}
