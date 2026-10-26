package handler

import (
	"encoding/csv"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"friends-records/internal/service"
	"friends-records/internal/token"
)

type PayrollPageData struct {
	ActiveMenu      string
	Payroll         []response.PayrollRecord
	Summary         response.PayrollSummary
	Employees       []mysql.AdminOption
	RestDays        int
	WorkDays        int
	StatusFilter    string
	Search          string
	PayrollType     string
	PageTitle       string
	PageDescription string
}

type SpecialPayrollPageData struct {
	ActiveMenu string
	Month      string
	Items      []response.SpecialPayrollRecord
	Summary    response.SpecialPayrollSummary
}

func (h *Handler) PayrollPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
	if r.URL.Query().Get("type") == "special" {
		items, summary, err := h.Store.SpecialPayrolls(r.Context(), month)
		if err != nil {
			fail(w, err, "特殊员工工资读取失败")
			return
		}
		h.render(w, "special_payroll.html", SpecialPayrollPageData{ActiveMenu: "payroll-special", Month: month, Items: items, Summary: summary})
		return
	}
	items, err := h.Store.Payroll(r.Context(), month)
	if err != nil {
		fail(w, err, "工资记录读取失败")
		return
	}
	payrollType := r.URL.Query().Get("type")
	if payrollType != "part_time" && payrollType != "full_time" {
		payrollType = "full_time"
	}
	statusFilter, search := r.URL.Query().Get("status"), r.URL.Query().Get("q")
	items = filterPayroll(items, statusFilter, search)
	items = filterPayrollType(items, payrollType)
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
	// 工资统一按固定30天计薪，默认公休3天，对应27个计薪日。
	workDays := 30 - h.Store.MonthlyRestDays
	if workDays < 1 {
		workDays = 1
	}
	summary = payrollSummaryForType(summary, items)
	pageTitle, pageDescription, activeMenu := "正式工工资", "请假、公休、入职及离职日期会自动参与正式工月度工资计算（离职当天计薪）", "payroll-formal"
	if payrollType == "part_time" {
		pageTitle, pageDescription, activeMenu = "临时工工资", "先登记每日工时，再按时薪、日薪或半天工资自动汇总本月工资", "payroll-temporary"
	}
	h.render(w, "payroll.html", PayrollPageData{ActiveMenu: activeMenu, Payroll: items, Summary: summary, Employees: people, RestDays: h.Store.MonthlyRestDays, WorkDays: workDays, StatusFilter: statusFilter, Search: search, PayrollType: payrollType, PageTitle: pageTitle, PageDescription: pageDescription})
}

func filterPayrollType(items []response.PayrollRecord, payrollType string) []response.PayrollRecord {
	result := make([]response.PayrollRecord, 0, len(items))
	for _, item := range items {
		if item.EmploymentType == payrollType {
			result = append(result, item)
		}
	}
	return result
}

// payrollSummaryForType 复用当前工资批次状态，只重算当前页面类型的金额和人数统计。
func payrollSummaryForType(summary response.PayrollSummary, items []response.PayrollRecord) response.PayrollSummary {
	var gross, deductions, net float64
	summary.EmployeeCount, summary.PaidCount = len(items), 0
	for _, item := range items {
		parse := func(value string) float64 {
			amount, _ := strconv.ParseFloat(strings.TrimLeft(value, "¥"), 64)
			return amount
		}
		gross += parse(item.BaseSalary)
		deductions += parse(item.AttendanceDeduction) + parse(item.OtherDeduction) + parse(item.SocialSecurity) + parse(item.Tax)
		net += parse(item.NetSalary)
		if item.Status == "已发放" {
			summary.PaidCount++
		}
	}
	summary.GrossAmount, summary.DeductionAmount, summary.NetAmount = mysql.Money(gross), mysql.Money(deductions), mysql.Money(net)
	return summary
}

// PayrollExport 导出当前月份工资明细，CSV 可直接用 Excel 打开。
func (h *Handler) PayrollExport(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := adminActor(w, r, tokens); !ok {
			return
		}
		if r.Method != http.MethodGet {
			httpx.MethodNotAllowed(w, http.MethodGet)
			return
		}
		month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
		items, err := h.Store.Payroll(r.Context(), month)
		if err != nil {
			fail(w, err, "工资表导出失败")
			return
		}
		payrollType := r.URL.Query().Get("type")
		if payrollType != "part_time" {
			payrollType = "full_time"
		}
		items = filterPayrollType(items, payrollType)
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		fileName := month + "正式工工资.csv"
		if payrollType == "part_time" {
			fileName = month + "临时工工资.csv"
		}
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(fileName))
		writer := csv.NewWriter(w)
		_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
		if payrollType == "part_time" {
			_ = writer.Write([]string{"员工姓名", "计薪方式", "本月工时", "奖金补贴", "应发工资", "其他扣款"})
			for _, item := range items {
				base, _ := strconv.ParseFloat(strings.TrimPrefix(item.BaseSalary, "¥"), 64)
				bonus, _ := strconv.ParseFloat(strings.TrimPrefix(item.Bonus, "¥"), 64)
				_ = writer.Write([]string{item.Name, payrollPayBasisLabel(item.PayBasis), item.WorkHours, item.Bonus, mysql.Money(base + bonus), item.OtherDeduction})
			}
			writer.Flush()
			return
		}
		_ = writer.Write([]string{"序号", "员工", "入职时间", "应出勤天数", "请假天数", "公休天数", "入职时工资", "本月基本工资", "奖金补贴", "考勤扣款", "其他扣款", "社保", "个税", "实发工资"})
		for i, item := range items {
			_ = writer.Write([]string{strconv.Itoa(i + 1), item.Name, item.JoinedOn, strconv.Itoa(item.RequiredAttendanceDays), item.LeaveDays, item.PublicRestDays, item.EntrySalary, item.BaseSalary, item.Bonus, item.AttendanceDeduction, item.OtherDeduction, item.SocialSecurity, item.Tax, item.NetSalary})
		}
		writer.Flush()
	}
}

func payrollPayBasisLabel(value string) string {
	switch value {
	case "hourly":
		return "按小时"
	case "daily":
		return "按天"
	case "half_daily":
		return "按半天"
	default:
		return "按月"
	}
}
func (h *Handler) PayrollAPI(payroll service.Payroll, tokens *token.Manager) http.HandlerFunc {
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
		case http.MethodDelete:
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			id, _ := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
			ids := []uint64{id}
			if id == 0 {
				var input struct {
					ItemIDs []uint64 `json:"item_ids"`
				}
				if !decodeJSON(w, r, &input) {
					return
				}
				ids = input.ItemIDs
			}
			deleted, err := payroll.Delete(r.Context(), ids, actor)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"deleted": deleted})
		default:
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPut, http.MethodDelete)
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
			Month       string   `json:"month"`
			EmployeeID  uint64   `json:"employee_id"`
			EmployeeIDs []uint64 `json:"employee_ids"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		var batchID uint64
		var err error
		if len(input.EmployeeIDs) > 0 || input.EmployeeID > 0 {
			ids := input.EmployeeIDs
			if input.EmployeeID > 0 {
				ids = append(ids, input.EmployeeID)
			}
			seen := make(map[uint64]struct{}, len(ids))
			for _, employeeID := range ids {
				if employeeID == 0 {
					continue
				}
				if _, ok := seen[employeeID]; ok {
					continue
				}
				seen[employeeID] = struct{}{}
				batchID, err = payroll.GenerateEmployee(r.Context(), input.Month, employeeID)
				if err != nil {
					break
				}
			}
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
		httpx.Error(w, http.StatusConflict, "请在正式工工资或临时工工资页面点击“确认并汇总入账”")
	}
}

func (h *Handler) TemporaryPayrollAPI(payroll service.Payroll) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		httpx.Error(w, http.StatusConflict, "请在临时工工资页面点击“确认并汇总入账”")
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
			ItemID  uint64 `json:"item_id"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		if input.ItemID == 0 && input.BatchID == 0 {
			httpx.Error(w, http.StatusBadRequest, "工资批次ID不正确")
			return
		}
		var err error
		if input.ItemID > 0 {
			err = payroll.ConfirmEmployee(r.Context(), input.ItemID)
		} else {
			err = payroll.Confirm(r.Context(), input.BatchID)
		}
		if err != nil {
			fail(w, err, "工资确认失败")
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"batch_id": input.BatchID, "item_id": input.ItemID})
	}
}

// PayrollSettlementAPI 按正式工或临时工类型确认本月工资，并生成一条汇总支出流水。
func (h *Handler) PayrollSettlementAPI(payroll service.Payroll, tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		var input request.PayrollSettlementInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		ledgerID, err := payroll.SettleType(r.Context(), actor, input)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"ledger_id": ledgerID})
	}
}

// SpecialPayrollAPI 维护未建档员工的特殊工资，并按月汇总入账。
func (h *Handler) SpecialPayrollAPI(payroll service.Payroll, tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, summary, err := payroll.SpecialList(r.Context(), mysql.NormalizeMonth(r.URL.Query().Get("month")))
			if err != nil {
				fail(w, err, "特殊员工工资读取失败")
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "summary": summary})
		case http.MethodPost, http.MethodPut:
			actor, ok := adminActor(w, r, tokens)
			if !ok {
				return
			}
			id, ok := adminID(w, r, r.Method == http.MethodPut)
			if !ok {
				return
			}
			var input request.SpecialPayrollInput
			if !decodeJSON(w, r, &input) {
				return
			}
			if err := input.Validate(); err != nil {
				httpx.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			saved, err := payroll.SaveSpecial(r.Context(), id, actor, input)
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
			if err := payroll.DeleteSpecial(r.Context(), id, actor); err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
		default:
			httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)
		}
	}
}

func (h *Handler) SpecialPayrollSettlementAPI(payroll service.Payroll, tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		var input request.SpecialPayrollSettlementInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		ledgerID, err := payroll.SettleSpecial(r.Context(), actor, input)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"ledger_id": ledgerID})
	}
}
