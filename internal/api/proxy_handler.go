package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"monitor/internal/config"
	"monitor/internal/modelvendor"
)

type adminProxyProfileView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URLMasked string `json:"url_masked"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

func proxyProfileView(profile config.ProxyProfile) adminProxyProfileView {
	return adminProxyProfileView{
		ID:        profile.ID,
		Name:      profile.Name,
		URLMasked: maskProxyProfileURL(profile.URL),
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
}

func maskProxyProfileURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "********"
	}
	parsed.User = nil
	return parsed.Scheme + "://" + parsed.Host
}

func (h *Handler) getProxyStore(c *gin.Context) *config.ProxyProfileStore {
	store := h.proxyStore
	if store == nil {
		apiError(c, http.StatusServiceUnavailable, ErrCodeFeatureDisabled, "代理配置管理未启用")
	}
	return store
}

// GetModelVendors 返回模型厂商词表。它不依赖 onboarding，供管理员通道表单使用。
// GET /api/model-vendors
func (h *Handler) GetModelVendors(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"model_vendors": modelvendor.Options()})
}

// AdminListProxyProfiles 列出代理配置（地址脱敏）。
// GET /api/admin/proxies
func (h *Handler) AdminListProxyProfiles(c *gin.Context) {
	if !h.checkAdminToken(c) {
		return
	}
	store := h.getProxyStore(c)
	if store == nil {
		return
	}
	profiles, err := store.List()
	if err != nil {
		apiError(c, http.StatusInternalServerError, ErrCodeInternalError, "读取代理配置失败")
		return
	}
	views := make([]adminProxyProfileView, 0, len(profiles))
	for _, profile := range profiles {
		views = append(views, proxyProfileView(profile))
	}
	c.JSON(http.StatusOK, gin.H{"proxies": views})
}

// AdminCreateProxyProfile 创建代理配置。
// POST /api/admin/proxies
func (h *Handler) AdminCreateProxyProfile(c *gin.Context) {
	if !h.checkAdminToken(c) {
		return
	}
	store := h.getProxyStore(c)
	if store == nil {
		return
	}
	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusBadRequest, ErrCodeInvalidParam, "请求参数无效")
		return
	}
	profile, err := store.Create(req.Name, req.URL)
	if err != nil {
		apiError(c, http.StatusBadRequest, ErrCodeInvalidParam, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"proxy": proxyProfileView(*profile)})
}

// AdminUpdateProxyProfile 更新代理配置。
// PUT /api/admin/proxies/:id
func (h *Handler) AdminUpdateProxyProfile(c *gin.Context) {
	if !h.checkAdminToken(c) {
		return
	}
	store := h.getProxyStore(c)
	if store == nil {
		return
	}
	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusBadRequest, ErrCodeInvalidParam, "请求参数无效")
		return
	}
	profile, err := store.Update(c.Param("id"), req.Name, req.URL)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		apiError(c, status, ErrCodeInvalidParam, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"proxy": proxyProfileView(*profile)})
}

// AdminDeleteProxyProfile 删除代理配置。被监测通道引用时拒绝删除，避免热更新后配置失效。
// DELETE /api/admin/proxies/:id
func (h *Handler) AdminDeleteProxyProfile(c *gin.Context) {
	if !h.checkAdminToken(c) {
		return
	}
	store := h.getProxyStore(c)
	if store == nil {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if h.proxyProfileReferenced(id) {
		apiError(c, http.StatusConflict, ErrCodeInvalidParam, "代理配置仍被监测通道引用，请先改为直连或其他代理")
		return
	}
	if err := store.Delete(id); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		apiError(c, status, ErrCodeInvalidParam, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) proxyProfileReferenced(id string) bool {
	if h.monitorStore == nil || strings.TrimSpace(id) == "" {
		return false
	}
	summaries, err := h.monitorStore.List()
	if err != nil {
		return true
	}
	for _, summary := range summaries {
		file, err := h.monitorStore.Get(summary.Key)
		if err != nil || file == nil {
			return true
		}
		for _, monitor := range file.Monitors {
			if strings.TrimSpace(monitor.ProxyProfile) == id {
				return true
			}
		}
	}
	return false
}

func (h *Handler) validateMonitorProxyProfiles(monitors []config.ServiceConfig) error {
	for _, monitor := range monitors {
		id := strings.TrimSpace(monitor.ProxyProfile)
		if id == "" {
			continue
		}
		if h.proxyStore == nil {
			return fmt.Errorf("代理配置管理未启用")
		}
		profile, err := h.proxyStore.Get(id)
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("代理配置不存在: %s", id)
		}
	}
	return nil
}
