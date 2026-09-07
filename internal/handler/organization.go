package handler

import (
	"friends-records/api/response"
	"friends-records/internal/httpx"
	"net/http"
)

type OrganizationPageData struct {
	ActiveMenu  string
	Departments []response.Department
	Positions   []response.Position
	Summary     response.OrganizationSummary
}

func (h *Handler) OrganizationPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	departments, err := h.Store.Departments(r.Context())
	if err != nil {
		fail(w, err, "部门数据读取失败")
		return
	}
	positions, err := h.Store.Positions(r.Context())
	if err != nil {
		fail(w, err, "岗位数据读取失败")
		return
	}
	summary, err := h.Store.OrganizationSummary(r.Context())
	if err != nil {
		fail(w, err, "组织架构统计读取失败")
		return
	}
	h.render(w, "organization.html", OrganizationPageData{"organization", departments, positions, summary})
}
func (h *Handler) OrganizationAPI(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	departments, err := h.Store.Departments(r.Context())
	if err != nil {
		fail(w, err, "部门数据读取失败")
		return
	}
	positions, err := h.Store.Positions(r.Context())
	if err != nil {
		fail(w, err, "岗位数据读取失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"departments": departments, "positions": positions})
}
