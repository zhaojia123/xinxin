package handler

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"

	"friends-records/internal/apperror"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
)

type Handler struct {
	Store     *mysql.Store
	Templates *template.Template
}

func New(store *mysql.Store, templates *template.Template) *Handler {
	return &Handler{store, templates}
}
func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("渲染后台页面%s失败：%v", name, apperror.Wrap(err, "HTML模板渲染失败"))
	}
}
func getOnly(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet {
		return true
	}
	httpx.MethodNotAllowed(w, http.MethodGet)
	return false
}
func decodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求内容格式不正确")
		return false
	}
	return true
}
func fail(w http.ResponseWriter, err error, message string) {
	log.Printf("%s：%v", message, err)
	status := http.StatusInternalServerError
	if errors.Is(err, mysql.ErrNotConfigured) {
		status = http.StatusServiceUnavailable
	}
	if errors.Is(err, mysql.ErrNotFound) {
		status = http.StatusNotFound
	}
	httpx.ErrorWithCause(w, status, message, err)
}
