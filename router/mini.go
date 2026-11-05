package router

import (
	"context"
	"net/http"

	"friends-records/config"
	"friends-records/internal/handler"
	modelmysql "friends-records/internal/models/mysql"
	"friends-records/internal/service"
	"friends-records/internal/token"
)

// registerMiniRoutes 只注册微信小程序使用的 /api/mini 接口。
func registerMiniRoutes(mux *http.ServeMux, h *handler.Handler, store *modelmysql.Store, cfg config.Config, tokens *token.Manager) {
	var exchanger service.CodeExchanger = service.NewWeChatClient(cfg.WeChat.AppID, cfg.WeChat.AppSecret)
	if cfg.WeChat.MockLogin {
		exchanger = service.MockWeChatClient{}
	}
	mini := service.Mini{Store: store, Exchanger: exchanger, Tokens: tokens, AutoRegister: cfg.WeChat.AutoRegister}
	// POST /api/mini/login：使用 wx.login 返回的 code 登录小程序。
	mux.HandleFunc("/api/mini/login", h.MiniLogin(mini))
	employeeAccess := func(ctx context.Context, id uint64, action string) (bool, error) {
		return store.MiniPermissionAction(ctx, id, "employees", action)
	}
	attendanceAccess := func(ctx context.Context, id uint64, action string) (bool, error) {
		return store.MiniPermissionAction(ctx, id, "attendance", action)
	}
	// 小程序只提供员工资料和请假/调休，其他经营数据仍只在后台维护。
	mux.HandleFunc("/api/mini/employees", handler.RequireMiniCRUD(tokens, store.MiniEnabled, employeeAccess, h.MiniEmployeesAPI))
	mux.HandleFunc("/api/mini/attendance", handler.RequireMiniCRUD(tokens, store.MiniEnabled, attendanceAccess, h.MiniAttendanceAPI))
	mux.HandleFunc("/api/mini/options", handler.RequireMiniPermission(tokens, store.MiniEnabled, func(ctx context.Context, id uint64) (bool, error) {
		return store.MiniPermissionAction(ctx, id, "employees", "view")
	}, h.MiniOptionsAPI))
}
