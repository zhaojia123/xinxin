package router

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"friends-records/config"
	"friends-records/internal/handler"
	modelmysql "friends-records/internal/models/mysql"
	"friends-records/internal/service"
)

// registerUploadRoutes 只注册文件上传和文件读取路由。
func registerUploadRoutes(mux *http.ServeMux, h *handler.Handler, store *modelmysql.Store, cfg config.Config) {
	upload := service.HealthCertificate{Store: store, RootDir: cfg.Upload.Dir, MaxBytes: cfg.Upload.MaxSizeMB << 20}

	// POST /api/admin/upload/health-certificate：上传员工健康证照片并保存发证、到期日期。
	mux.HandleFunc("/api/admin/upload/health-certificate", h.HealthCertificateUpload(upload))
	// GET /uploads/*：读取已经上传的健康证图片。
	mux.HandleFunc("/uploads/", serveUploadedFile(cfg.Upload.Dir))
}

func serveUploadedFile(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		relative := strings.TrimPrefix(r.URL.Path, "/uploads/")
		cleaned := strings.TrimPrefix(path.Clean("/"+relative), "/")
		if relative == "" || cleaned != relative || strings.HasSuffix(relative, "/") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(root, filepath.FromSlash(cleaned)))
	}
}
