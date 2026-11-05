package handler

import (
	"net/http"
	"strconv"

	"friends-records/internal/httpx"
	"friends-records/internal/token"
)

// AdminDataExport 为后台已有列表提供统一的标准 XLSX 导出。
func (h *Handler) AdminDataExport(tokens *token.Manager, kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := adminActor(w, r, tokens); !ok {
			return
		}
		if r.Method != http.MethodGet {
			httpx.MethodNotAllowed(w, http.MethodGet)
			return
		}
		var filename, title, sheet string
		var headers []string
		var records [][]exportCell
		var widths []int
		switch kind {
		case "employees":
			items, err := h.Store.Employees(r.Context())
			if err != nil {
				fail(w, err, "员工表导出失败")
				return
			}
			filename, title, sheet, headers, widths = "员工档案.xlsx", "员工档案表", "员工档案", []string{"序号", "工号", "姓名", "部门", "岗位", "状态", "入职日期", "联系电话", "薪资"}, []int{48, 110, 90, 100, 100, 80, 95, 110, 90}
			for i, item := range items {
				records = append(records, []exportCell{exportInteger(itoa(i + 1)), exportText(item.EmployeeNo), exportText(item.Name), exportText(item.Department), exportText(item.Position), exportText(item.Status), exportText(item.JoinedOn), exportText(item.Mobile), exportNumber(item.SalaryValue)})
			}
		case "changes":
			items, err := h.Store.EmploymentChanges(r.Context())
			if err != nil {
				fail(w, err, "异动记录导出失败")
				return
			}
			items = filterChanges(items, r.URL.Query().Get("month"), r.URL.Query().Get("change_type"), r.URL.Query().Get("q"))
			filename, title, sheet, headers, widths = "人事异动记录.xlsx", "人事异动记录", "人事异动", []string{"序号", "工号", "姓名", "异动类型", "变更前", "变更后", "生效日期", "原因", "操作人"}, []int{48, 100, 90, 90, 120, 120, 95, 180, 90}
			for i, item := range items {
				records = append(records, []exportCell{exportInteger(itoa(i + 1)), exportText(item.EmployeeNo), exportText(item.Name), exportText(item.Type), exportText(item.Before), exportText(item.After), exportText(item.EffectiveOn), exportText(item.Reason), exportText(item.Operator)})
			}
		case "salary":
			items, err := h.Store.SalaryAdjustments(r.Context())
			if err != nil {
				fail(w, err, "调薪记录导出失败")
				return
			}
			items = filterSalary(items, r.URL.Query().Get("month"), r.URL.Query().Get("q"))
			filename, title, sheet, headers, widths = "调薪记录.xlsx", "调薪记录", "调薪记录", []string{"序号", "工号", "姓名", "调整前", "调整后", "调整金额", "生效日期", "原因", "操作人"}, []int{48, 100, 90, 90, 90, 90, 95, 180, 90}
			for i, item := range items {
				records = append(records, []exportCell{exportInteger(itoa(i + 1)), exportText(item.EmployeeNo), exportText(item.Name), exportNumber(item.Before), exportNumber(item.After), exportNumber(item.Increase), exportText(item.EffectiveOn), exportText(item.Reason), exportText(item.Operator)})
			}
		case "audit":
			items, err := h.Store.AuditLogs(r.Context())
			if err != nil {
				fail(w, err, "操作日志导出失败")
				return
			}
			filename, title, sheet, headers, widths = "操作日志.xlsx", "操作日志", "操作日志", []string{"序号", "时间", "操作人", "模块", "动作", "对象", "明细", "IP", "结果"}, []int{48, 145, 90, 100, 90, 150, 200, 120, 75}
			for i, item := range items {
				records = append(records, []exportCell{exportInteger(itoa(i + 1)), exportText(item.Time), exportText(item.Operator), exportText(item.Module), exportText(item.Action), exportText(item.Target), exportText(item.Detail), exportText(item.IP), exportText(item.Result)})
			}
		case "ledger":
			start, end, err := ledgerDateRange(r)
			if err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			categoryID, err := ledgerCategoryID(r)
			if err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			items, err := h.Store.LedgerRangeFiltered(r.Context(), start, end, categoryID)
			if err != nil {
				fail(w, err, "台账导出失败")
				return
			}
			filename, title, sheet, headers, widths = "台账流水.xlsx", "台账流水", "台账流水", []string{"序号", "日期", "部门", "摘要", "对方单位/账户", "分类", "账户", "流入", "流出", "结存"}, []int{48, 95, 100, 180, 150, 100, 100, 90, 90, 90}
			for i, item := range items {
				records = append(records, []exportCell{exportInteger(itoa(i + 1)), exportText(item.Date), exportText(item.Department), exportText(item.Summary), exportText(item.Counterparty), exportText(item.Category), exportText(item.Account), exportNumber(item.Income), exportNumber(item.Expense), exportNumber(item.Balance)})
			}
		default:
			http.NotFound(w, r)
			return
		}
		if err := writeExcelXLSX(w, filename, title, sheet, headers, records, widths); err != nil {
			fail(w, err, title+"导出失败")
		}
	}
}

func itoa(value int) string { return strconv.Itoa(value) }
