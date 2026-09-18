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
	ledgerAccess := func(ctx context.Context, id uint64, action string) (bool, error) {
		return store.MiniPermissionAction(ctx, id, "ledger", action)
	}
	purchaseAccess := func(ctx context.Context, id uint64, action string) (bool, error) {
		return store.MiniPermissionAction(ctx, id, "purchases", action)
	}
	employeeAccess := func(ctx context.Context, id uint64, action string) (bool, error) {
		return store.MiniPermissionAction(ctx, id, "employees", action)
	}
	anyAccess := store.MiniAnyPermission
	// 每个业务接口都校验对应模块权限，避免仅隐藏导航而仍可直接写入其他模块。
	mux.HandleFunc("/api/mini/ledger", handler.RequireMiniCRUD(tokens, store.MiniEnabled, ledgerAccess, h.MiniLedgerAPI))
	mux.HandleFunc("/api/mini/purchases", handler.RequireMiniCRUD(tokens, store.MiniEnabled, purchaseAccess, h.MiniPurchasesAPI))
	mux.HandleFunc("/api/mini/purchases/export", handler.RequireMiniPermission(tokens, store.MiniEnabled, func(ctx context.Context, id uint64) (bool, error) {
		return store.MiniPermissionAction(ctx, id, "purchases", "view")
	}, h.MiniPurchaseExport))
	mux.HandleFunc("/api/mini/employees", handler.RequireMiniCRUD(tokens, store.MiniEnabled, employeeAccess, h.MiniEmployeesAPI))
	mux.HandleFunc("/api/mini/options", handler.RequireMiniPermission(tokens, store.MiniEnabled, anyAccess, h.MiniOptionsAPI))
}
