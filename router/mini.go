package router

import (
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
	// 启用的小程序用户共同管理当前项目的台账及员工档案。
	mux.HandleFunc("/api/mini/ledger", handler.RequireMini(tokens, store.MiniEnabled, h.MiniLedgerAPI))
	mux.HandleFunc("/api/mini/employees", handler.RequireMini(tokens, store.MiniEnabled, h.MiniEmployeesAPI))
	mux.HandleFunc("/api/mini/options", handler.RequireMini(tokens, store.MiniEnabled, h.MiniOptionsAPI))
}
