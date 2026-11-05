package handler

import (
	"net/http"
	"strings"
	"time"

	"friends-records/api/request"
	"friends-records/internal/httpx"
	"friends-records/internal/service"
	"friends-records/internal/token"
)

type LoginPageData struct{ Error, Username string }
type PasswordPageData struct{ ActiveMenu, Error, Success string }

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

// AdminPasswordPage 允许登录中的管理员修改自己的密码；成功后立即退出当前会话。
func (h *Handler) AdminPasswordPage(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.render(w, "admin_password.html", PasswordPageData{ActiveMenu: "admin-password"})
			return
		}
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		if err := r.ParseForm(); err != nil {
			h.render(w, "admin_password.html", PasswordPageData{ActiveMenu: "admin-password", Error: "提交内容格式不正确"})
			return
		}
		input := request.AdminPasswordInput{CurrentPassword: r.FormValue("current_password"), NewPassword: r.FormValue("new_password"), ConfirmPassword: r.FormValue("confirm_password")}
		if err := input.Validate(); err != nil {
			h.render(w, "admin_password.html", PasswordPageData{ActiveMenu: "admin-password", Error: err.Error()})
			return
		}
		if err := h.Store.ChangeAdminPassword(r.Context(), actor, input.CurrentPassword, input.NewPassword); err != nil {
			if service.IsInvalidCredentials(err) {
				h.render(w, "admin_password.html", PasswordPageData{ActiveMenu: "admin-password", Error: "当前密码不正确"})
				return
			}
			fail(w, err, "修改后台密码失败")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "admin_token", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	}
}

// AdminLogout 清除后台会话，供手动退出和浏览器无操作超时使用。
func (h *Handler) AdminLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "admin_token", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) AdminLoginPage(admin service.Admin, tokens *token.Manager) http.HandlerFunc {
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
		user, err := admin.Login(r.Context(), username, r.FormValue("password"))
		if service.IsInvalidCredentials(err) {
			h.render(w, "login.html", LoginPageData{Error: "用户名或密码错误", Username: username})
			return
		}
		if err != nil {
			fail(w, err, "登录服务暂时不可用")
			return
		}
		// 后台会话采用两小时滑动过期，页面每次操作会由路由中间件续期。
		value, err := tokens.Issue("admin", user.ID, 2*time.Hour)
		if err != nil {
			fail(w, err, "生成后台登录凭证失败")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "admin_token", Value: value, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 2 * 60 * 60})
		http.Redirect(w, r, "/admin/employees", http.StatusSeeOther)
	}
}
