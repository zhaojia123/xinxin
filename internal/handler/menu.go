package handler

import (
	"net/http"
	"strconv"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/httpx"
	"friends-records/internal/token"
)

type MenuPageData struct {
	ActiveMenu    string
	Keyword       string
	PriceCategory string
	ItemCategory  string
	MinPrice      string
	MaxPrice      string
	Sort          string
	Menu          response.MenuPage
}

func (h *Handler) MenuPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	sort := r.URL.Query().Get("sort")
	menu, err := h.Store.MenuItems(r.Context(), page, pageSize, r.URL.Query().Get("q"), r.URL.Query().Get("price_category"), r.URL.Query().Get("item_category"), r.URL.Query().Get("min_price"), r.URL.Query().Get("max_price"), sort)
	if err != nil {
		fail(w, err, "菜单数据读取失败")
		return
	}
	h.render(w, "menu.html", MenuPageData{ActiveMenu: "menu", Keyword: r.URL.Query().Get("q"), PriceCategory: r.URL.Query().Get("price_category"), ItemCategory: r.URL.Query().Get("item_category"), MinPrice: r.URL.Query().Get("min_price"), MaxPrice: r.URL.Query().Get("max_price"), Sort: sort, Menu: menu})
}

func (h *Handler) MenuAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
			menu, err := h.Store.MenuItems(r.Context(), page, pageSize, r.URL.Query().Get("q"), r.URL.Query().Get("price_category"), r.URL.Query().Get("item_category"), r.URL.Query().Get("min_price"), r.URL.Query().Get("max_price"), r.URL.Query().Get("sort"))
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, menu)
		case http.MethodPost, http.MethodPut:
			id, ok := adminID(w, r, r.Method == http.MethodPut)
			if !ok {
				return
			}
			var input request.MenuItemInput
			if !decodeJSON(w, r, &input) {
				return
			}
			if err := input.Validate(); err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			saved, err := h.Store.SaveMenuItem(r.Context(), id, actor, input)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": saved})
		case http.MethodDelete:
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			id, ok := adminID(w, r, true)
			if !ok {
				return
			}
			if err := h.Store.DeleteMenuItem(r.Context(), id, actor); err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
		default:
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)
		}
	}
}
