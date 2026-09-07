package router

import (
	"context"
	"database/sql"
	"html/template"
	"io/fs"
	"net/http"
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
	handlers := handler.New(store, templates)

	// GET /assets/*：后台页面内嵌的 CSS 和 JavaScript。
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(static))))
	// GET /healthz：检查 HTTP 服务和 MySQL 连接状态。
	mux.HandleFunc("/healthz", health(db))
	registerAdminRoutes(mux, handlers, store)
	registerMiniRoutes(mux, handlers, store, cfg, tokens)
	registerUploadRoutes(mux, handlers, store, cfg)
	// GET /：跳转后台入口。
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})
	return httpx.Common(mux)
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
