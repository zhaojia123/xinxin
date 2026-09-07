package handler

import (
	"errors"
	"net/http"

	"friends-records/api/request"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"friends-records/internal/service"
)

func (h *Handler) MiniLogin(mini service.Mini) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		var input request.MiniLogin
		if !decodeJSON(w, r, &input) {
			return
		}
		result, err := mini.Login(r.Context(), input.Code)
		if errors.Is(err, service.ErrMiniUserDisabled) {
			httpx.Error(w, http.StatusForbidden, "用户正在等待管理员启用")
			return
		}
		if errors.Is(err, mysql.ErrNotConfigured) {
			httpx.Error(w, http.StatusServiceUnavailable, "数据库尚未配置")
			return
		}
		if err != nil {
			fail(w, err, "微信登录失败")
			return
		}
		httpx.JSON(w, http.StatusOK, result)
	}
}
