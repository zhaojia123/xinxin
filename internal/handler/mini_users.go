package handler

import (
	"net/http"

	"friends-records/api/request"
	"friends-records/internal/httpx"
	"friends-records/internal/token"
)

func (h *Handler) MiniUsersPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	h.render(w, "mini_users.html", map[string]string{"ActiveMenu": "mini-users"})
}

func (h *Handler) AdminMiniUsersAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := adminActor(w, r, tokens); !ok {
			return
		}
		if r.Method == http.MethodGet {
			users, modules, err := h.Store.AdminMiniUsers(r.Context())
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"users": users, "modules": modules})
			return
		}
		if r.Method != http.MethodPut {
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPut)
			return
		}
		id, ok := adminID(w, r, true)
		if !ok {
			return
		}
		var input request.MiniUserPermissionInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.Store.UpdateMiniUserPermissions(r.Context(), id, input); err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
	}
}
