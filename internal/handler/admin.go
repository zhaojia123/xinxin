package handler

import (
	"net/http"
	"strings"

	"friends-records/internal/httpx"
	"friends-records/internal/service"
)

type LoginPageData struct{ Error, Username string }

func (h *Handler) AdminRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin" {
		http.NotFound(w, r)
		return
	}
	if !getOnly(w, r) {
		return
	}
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}
func (h *Handler) AdminLoginPage(admin service.Admin) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.render(w, "login.html", LoginPageData{})
			return
		}
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
			return
		}
		if err := r.ParseForm(); err != nil {
			h.render(w, "login.html", LoginPageData{Error: "提交内容格式不正确"})
			return
		}
		username := strings.TrimSpace(r.FormValue("username"))
		_, err := admin.Login(r.Context(), username, r.FormValue("password"))
		if service.IsInvalidCredentials(err) {
			h.render(w, "login.html", LoginPageData{Error: "用户名或密码错误", Username: username})
			return
		}
		if err != nil {
			fail(w, err, "登录服务暂时不可用")
			return
		}
		http.Redirect(w, r, "/admin/employees", http.StatusSeeOther)
	}
}
