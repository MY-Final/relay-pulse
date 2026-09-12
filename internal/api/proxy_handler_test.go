package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"monitor/internal/modelvendor"
)

func TestGetModelVendorsDoesNotDependOnOnboarding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/model-vendors", nil)
	h.GetModelVendors(c)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200", w.Code)
	}
	var resp struct {
		Vendors []struct {
			Code string `json:"code"`
		} `json:"model_vendors"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Vendors) != len(modelvendor.Options()) {
		t.Fatalf("厂商数量 = %d, want %d", len(resp.Vendors), len(modelvendor.Options()))
	}
	if len(resp.Vendors) == 0 || resp.Vendors[0].Code == "" {
		t.Fatal("厂商接口不应返回空列表")
	}
}
