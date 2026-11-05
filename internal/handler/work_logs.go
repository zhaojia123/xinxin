package handler

import (
	"net/http"
	"strconv"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"friends-records/internal/token"
)

type WorkLogsPageData struct {
	ActiveMenu string
	Month      string
	Search     string
	Logs       []response.WorkLog
	Employees  []response.Employee
}

func (h *Handler) WorkLogsPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
	logs, err := h.Store.WorkLogs(r.Context(), month, r.URL.Query().Get("q"))
	if err != nil {
		fail(w, err, "小时工工时读取失败")
		return
	}
	employees, err := h.Store.Employees(r.Context())
	if err != nil {
		fail(w, err, "小时工列表读取失败")
		return
	}
	temporaryWorkers := make([]response.Employee, 0)
	for _, item := range employees {
		if item.EmploymentType == "兼职" && (item.PayBasis == "hourly" || item.PayBasis == "daily" || item.PayBasis == "half_daily") && item.Status != "已离职" {
			temporaryWorkers = append(temporaryWorkers, item)
		}
	}
	h.render(w, "work_logs.html", WorkLogsPageData{ActiveMenu: "work-logs", Month: month, Search: r.URL.Query().Get("q"), Logs: logs, Employees: temporaryWorkers})
}

func (h *Handler) WorkLogsAPI(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		id, _ := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
		switch r.Method {
		case http.MethodGet:
			month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
			items, err := h.Store.WorkLogs(r.Context(), month, r.URL.Query().Get("q"))
			if err != nil {
				fail(w, err, "小时工工时读取失败")
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "month": month})
		case http.MethodPost, http.MethodPut:
			if r.Method == http.MethodPost && id != 0 {
				httpx.Error(w, http.StatusBadRequest, "新增工时不能携带ID")
				return
			}
			var input request.WorkLogInput
			if !decodeJSON(w, r, &input) {
				return
			}
			if err := input.Validate(); err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			saved, err := h.Store.SaveWorkLogs(r.Context(), id, actor, input)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": saved[0], "ids": saved, "saved_count": len(saved)})
		case http.MethodDelete:
			ids := []uint64{id}
			if id == 0 {
				var input struct {
					IDs []uint64 `json:"ids"`
				}
				if !decodeJSON(w, r, &input) {
					return
				}
				ids = input.IDs
			}
			deleted, err := h.Store.DeleteWorkLogs(r.Context(), ids, actor)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"deleted": deleted})
		default:
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)
		}
	}
}

// WorkLogsExport 导出指定月份逐日出勤，保留全天、半天和实际小时的标记。
func (h *Handler) WorkLogsExport(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := adminActor(w, r, tokens); !ok {
			return
		}
		if r.Method != http.MethodGet {
			httpx.MethodNotAllowed(w, http.MethodGet)
			return
		}
		month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
		items, err := h.Store.WorkLogs(r.Context(), month, r.URL.Query().Get("q"))
		if err != nil {
			fail(w, err, "临时工工时导出失败")
			return
		}
		records := make([][]exportCell, 0, len(items))
		for _, item := range items {
			records = append(records, []exportCell{exportText(item.WorkDate), exportText(item.Name), exportText(item.PayBasisLabel), exportText(item.WorkLabel), exportNumber(item.Hours), exportText(item.PayRate), exportNumber(item.Amount), exportText(item.Remark)})
		}
		if err := writeExcelXLSX(w, month+"临时工每日出勤.xlsx", monthTitle(month)+"临时工每日出勤表", "临时工出勤", []string{"日期", "员工姓名", "计薪方式", "出勤标记", "实际工时", "计薪标准", "当天金额", "备注"}, records, []int{90, 90, 90, 90, 80, 95, 95, 180}); err != nil {
			fail(w, err, "临时工工时导出失败")
		}
	}
}
