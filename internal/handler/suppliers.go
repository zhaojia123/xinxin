package handler

import (
	"net/http"

	"friends-records/api/request"
	"friends-records/internal/httpx"
	"friends-records/internal/token"
)

// SuppliersAPI 管理供货商资料；档口对账从这里选择供货商。
func (h *Handler) SuppliersAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := adminID(w, r, false)
		if !ok {
			return
		}
		switch r.Method {
		case http.MethodGet:
			items, err := h.Store.Suppliers(r.Context())
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"records": items})
		case http.MethodPost, http.MethodPut:
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			if r.Method == http.MethodPost && id != 0 {
				httpx.Error(w, http.StatusBadRequest, "新增供货商不能携带ID")
				return
			}
			var input request.SupplierInput
			if !decodeJSON(w, r, &input) {
				return
			}
			if err := input.Validate(); err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			saved, err := h.Store.SaveSupplier(r.Context(), id, actor, input)
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
			if id == 0 {
				httpx.Error(w, http.StatusBadRequest, "删除供货商必须提供ID")
				return
			}
			if err := h.Store.DeleteSupplier(r.Context(), id, actor); err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
		default:
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)
		}
	}
}
