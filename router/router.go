package router

import (
	"context"
	"database/sql"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"friends-records/config"
	"friends-records/internal/handler"
	"friends-records/internal/httpx"
	modelmysql "friends-records/internal/models/mysql"
	"friends-records/internal/token"
)

func New(cfg config.Config, db *sql.DB, templates *template.Template, static fs.FS, tokens *token.Manager) http.Handler {
	mux := http.NewServeMux()
	store := modelmysql.New(db)
	if cfg.App.MonthlyRestDays > 0 {
		store.MonthlyRestDays = cfg.App.MonthlyRestDays
	}
	handlers := handler.New(store, templates)

	// GET /assets/*：后台页面内嵌的 CSS 和 JavaScript。
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(static))))
	// GET /healthz：检查 HTTP 服务和 MySQL 连接状态。
	mux.HandleFunc("/healthz", health(db))
	registerAdminRoutes(mux, handlers, store, tokens)
	registerMiniRoutes(mux, handlers, store, cfg, tokens)
	registerUploadRoutes(mux, handlers, store, cfg, tokens)
	// GET /：跳转后台入口。
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})
	return httpx.Common(requireAdminSession(tokens, mux))
}

const adminSessionTTL = 2 * time.Hour

// requireAdminSession 统一保护后台页面和接口，避免单个处理器遗漏登录校验。
// 每次有效后台请求都会续期两小时，以实现无操作两小时后自动退出。
func requireAdminSession(tokens *token.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/admin/login" || (!strings.HasPrefix(path, "/admin/") && path != "/admin" && !strings.HasPrefix(path, "/api/admin/")) {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie("admin_token")
		if err != nil {
			denyAdminSession(w, r)
			return
		}
		claims, err := tokens.Parse(cookie.Value, "admin")
		if err != nil {
			denyAdminSession(w, r)
			return
		}
		value, err := tokens.Issue("admin", claims.SubjectID, adminSessionTTL)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "刷新登录状态失败")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "admin_token", Value: value, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: int(adminSessionTTL.Seconds())})
		next.ServeHTTP(w, r)
	})
}

func denyAdminSession(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/admin/") {
		httpx.Error(w, http.StatusUnauthorized, "请重新登录后台")
		return
	}
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func health(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.MethodNotAllowed(w, http.MethodGet)
			return
		}
		result := map[string]string{"status": "ok", "database": "not_configured"}
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := db.PingContext(ctx); err != nil {
				httpx.Error(w, http.StatusServiceUnavailable, "数据库连接不可用")
				return
			}
			result["database"] = "connected"
		}
		httpx.JSON(w, http.StatusOK, result)
	}
}
