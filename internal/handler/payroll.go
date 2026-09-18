package handler

import (
	"net/http"
	"strconv"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"friends-records/internal/service"
)

type PayrollPageData struct {
	ActiveMenu string
	Payroll    []response.PayrollRecord
	Summary    response.PayrollSummary
	Employees  []mysql.AdminOption
}

func (h *Handler) PayrollPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
	items, err := h.Store.Payroll(r.Context(), month)
	if err != nil {
		fail(w, err, "工资记录读取失败")
		return
	}
	summary, err := h.Store.PayrollSummary(r.Context(), month)
	if err != nil {
		fail(w, err, "工资统计读取失败")
		return
	}
	options, err := h.Store.AdminOptions(r.Context())
	if err != nil {
		fail(w, err, "员工选项读取失败")
		return
	}
	people, _ := options["employees"].([]mysql.AdminOption)
	h.render(w, "payroll.html", PayrollPageData{ActiveMenu: "payroll", Payroll: items, Summary: summary, Employees: people})
}
func (h *Handler) PayrollAPI(payroll service.Payroll) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
			items, err := h.Store.Payroll(r.Context(), month)
			if err != nil {
				fail(w, err, "工资记录读取失败")
				return
			}
			summary, err := h.Store.PayrollSummary(r.Context(), month)
			if err != nil {
				fail(w, err, "工资统计读取失败")
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "summary": summary})
		case http.MethodPut:
			id, _ := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
			if id == 0 {
				httpx.Error(w, http.StatusBadRequest, "工资明细ID不正确")
				return
			}
			var input request.PayrollUpdate
			if !decodeJSON(w, r, &input) {
				return
			}
			net, err := payroll.Update(r.Context(), id, input)
			if err != nil {
				fail(w, err, "工资明细保存失败")
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "net_salary": mysql.Money(net)})
		default:
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPut)
		}
	}
}
func (h *Handler) PayrollGenerateAPI(payroll service.Payroll) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		var input struct {
			Month      string `json:"month"`
			EmployeeID uint64 `json:"employee_id"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		var batchID uint64
		var err error
		if input.EmployeeID > 0 {
			batchID, err = payroll.GenerateEmployee(r.Context(), input.Month, input.EmployeeID)
		} else {
			batchID, err = payroll.Generate(r.Context(), input.Month)
		}
		if err != nil {
			fail(w, err, "生成月度工资失败")
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"batch_id": batchID, "month": mysql.NormalizeMonth(input.Month)})
	}
}
func (h *Handler) PayrollPayAPI(payroll service.Payroll) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		var input request.PayrollPay
		if !decodeJSON(w, r, &input) {
			return
		}
		var entryID uint64
		var err error
		if input.ItemID > 0 {
			entryID, err = payroll.PayEmployee(r.Context(), input)
		} else {
			entryID, err = payroll.Pay(r.Context(), input)
		}
		if err != nil {
			fail(w, err, "工资发放失败")
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"ledger_entry_id": entryID})
	}
}
func (h *Handler) PayrollConfirmAPI(payroll service.Payroll) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		var input struct {
			BatchID uint64 `json:"batch_id"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		if input.BatchID == 0 {
			httpx.Error(w, http.StatusBadRequest, "工资批次ID不正确")
			return
		}
		if err := payroll.Confirm(r.Context(), input.BatchID); err != nil {
			fail(w, err, "工资确认失败")
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"batch_id": input.BatchID})
	}
}
