package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"friends-records/internal/token"
)

func TestRequireAdminSessionProtectsPagesAndAPIs(t *testing.T) {
	tokens := token.New("12345678901234567890123456789012")
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := requireAdminSession(tokens, next)

	api := httptest.NewRecorder()
	handler.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/admin/suppliers", nil))
	if api.Code != http.StatusUnauthorized {
		t.Fatalf("未登录 API 应返回401，实际为%d", api.Code)
	}

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/admin/supplier-costs", nil))
	if page.Code != http.StatusSeeOther || page.Header().Get("Location") != "/admin/login" {
		t.Fatalf("未登录页面应跳转登录页，实际状态=%d，地址=%q", page.Code, page.Header().Get("Location"))
	}

	value, err := tokens.Issue("admin", 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	authorized := httptest.NewRequest(http.MethodGet, "/api/admin/suppliers", nil)
	authorized.AddCookie(&http.Cookie{Name: "admin_token", Value: value})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, authorized)
	if response.Code != http.StatusNoContent || len(response.Result().Cookies()) != 1 {
		t.Fatalf("有效会话应放行并续期，状态=%d，Cookie=%d", response.Code, len(response.Result().Cookies()))
	}
}
