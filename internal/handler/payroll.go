package handler

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	activePeople := make([]mysql.AdminOption, 0, len(people))
	for _, person := range people {
		if person.EmploymentType == payrollType && (person.Status == "active" || person.Status == "probation") {
			activePeople = append(activePeople, person)
		}
	}
	// 工资统一按固定30天计薪，默认公休3天，对应27个计薪日。
	workDays := 30 - h.Store.MonthlyRestDays
	if workDays < 1 {
		workDays = 1
	}
	summary = payrollSummaryForType(summary, items)
	pageTitle, pageDescription, activeMenu := "正式工工资", "仅生成当前在职正式工；离职员工工资请按需手动补录", "payroll-formal"
	if payrollType == "part_time" {
		pageTitle, pageDescription, activeMenu = "临时工工资", "先登记每日工时，再按时薪、日薪或半天工资自动汇总本月工资", "payroll-temporary"
	}
	h.render(w, "payroll.html", PayrollPageData{ActiveMenu: activeMenu, Payroll: items, Summary: summary, Employees: activePeople, RestDays: h.Store.MonthlyRestDays, WorkDays: workDays, StatusFilter: statusFilter, Search: search, PayrollType: payrollType, PageTitle: pageTitle, PageDescription: pageDescription})
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
		// 手动实发保存后，以最终实发金额作为本页面“本月工资合计”，确保顶部汇总立即联动。
		gross += parse(item.NetSalary)
		deductions += parse(item.AttendanceDeduction) + parse(item.OtherDeduction) + parse(item.SocialSecurity) + parse(item.Tax)
		net += parse(item.NetSalary)
		if item.Status == "已发放" {
			summary.PaidCount++
		}
	}
	summary.GrossAmount, summary.DeductionAmount, summary.NetAmount = mysql.Money(gross), mysql.Money(deductions), mysql.Money(net)
	return summary
}

// PayrollExport 导出当前月份工资明细。正式工使用带月份标题的 Excel 签收表。
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
		if payrollType == "part_time" {
			headers := []string{"序号", "员工姓名", "计薪方式", "本月工时", "奖金补贴", "应发工资", "其他扣款"}
			widths := []int{48, 100, 100, 82, 92, 100, 100}
			records := make([][]exportCell, 0, len(items)+1)
			var hours, bonus, gross, deduction float64
			for index, item := range items {
				base, _ := strconv.ParseFloat(strings.TrimPrefix(item.BaseSalary, "¥"), 64)
				itemBonus, _ := strconv.ParseFloat(strings.TrimPrefix(item.Bonus, "¥"), 64)
				itemHours, _ := strconv.ParseFloat(item.WorkHours, 64)
				itemDeduction, _ := strconv.ParseFloat(strings.TrimPrefix(item.OtherDeduction, "¥"), 64)
				records = append(records, []exportCell{exportInteger(strconv.Itoa(index + 1)), exportText(item.Name), exportText(payrollPayBasisLabel(item.PayBasis)), exportNumber(item.WorkHours), exportNumber(item.Bonus), exportNumber(mysql.Money(base + itemBonus)), exportNumber(item.OtherDeduction)})
				hours += itemHours
				bonus += itemBonus
				gross += base + itemBonus
				deduction += itemDeduction
			}
			records = append(records, []exportCell{exportText("合计"), exportText(""), exportText(""), exportNumber(mysql.Money(hours)), exportNumber(mysql.Money(bonus)), exportNumber(mysql.Money(gross)), exportNumber(mysql.Money(deduction))})
			if err := writeExcelXLSX(w, "临时工工资_"+month+".xlsx", monthTitle(month)+"临时工工资表", "临时工工资表", headers, records, widths); err != nil {
				fail(w, err, "工资表导出失败")
			}
			return
		}

		headers := []string{"序号", "岗位", "姓名", "入职时间", "薪资/月", "离职时间", "应勤天数", "实勤天数", "应发工资", "社会保险", "住房公积金", "补贴", "实发合计", "员工签字", "备注"}
		widths := []int{48, 82, 82, 90, 82, 90, 76, 76, 88, 88, 92, 78, 92, 110, 180}
		records := make([][]exportCell, 0, len(items)+1)
		var gross, socialSecurity, subsidy, net float64
		for i, item := range items {
			records = append(records, []exportCell{
				exportInteger(strconv.Itoa(i + 1)),
				exportText(item.Position),
				exportText(item.Name),
				exportText(item.JoinedOn),
				exportNumber(item.EntrySalary),
				exportText(item.LeftOn),
				exportInteger(strconv.Itoa(item.RequiredAttendanceDays)),
				exportInteger(payrollActualDays(item)),
				exportNumber(item.BaseSalary),
				exportNumber(item.SocialSecurity),
				exportNumber("0"),
				exportNumber(item.Bonus),
				exportNumber(item.NetSalary),
				exportText(""),
				exportText(""),
			})
			gross += payrollExportAmount(item.BaseSalary)
			socialSecurity += payrollExportAmount(item.SocialSecurity)
			subsidy += payrollExportAmount(item.Bonus)
			net += payrollExportAmount(item.NetSalary)
		}
		records = append(records, []exportCell{
			exportText("合计"), exportText(""), exportText(""), exportText(""), exportText(""), exportText(""), exportText(""), exportText(""),
			exportNumber(mysql.Money(gross)), exportNumber(mysql.Money(socialSecurity)), exportNumber("0"), exportNumber(mysql.Money(subsidy)), exportNumber(mysql.Money(net)), exportText(""), exportText(""),
		})
		canteenName := ""
		if settings, settingsErr := h.Store.StallSettings(r.Context(), month); settingsErr == nil {
			canteenName = settings.CanteenName
		}
		if err := writeExcelXLSX(w, "正式工资签收表_"+month+".xlsx", payrollExportTitle(month, canteenName), "正式工资签收表", headers, records, widths); err != nil {
			fail(w, err, "工资表导出失败")
		}
	}
}

// payrollExportAmount 将页面展示用金额转换为导出汇总金额。
func payrollExportAmount(value string) float64 {
	amount, _ := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(value), "¥"), 64)
	return amount
}

// payrollExportTitle 使用食堂名称和月份生成工资条签收表标题。
func payrollExportTitle(month, canteenName string) string {
	parts := strings.Split(month, "-")
	if len(parts) == 2 && strings.TrimSpace(canteenName) != "" {
		monthNumber, err := strconv.Atoi(parts[1])
		if err == nil {
			return strings.TrimSpace(canteenName) + "员工" + strconv.Itoa(monthNumber) + "月份工资条签收表（后厨）"
		}
	}
	return monthTitle(month) + "正式员工工资签收表"
}

// payrollActualDays 按工资生成的实际在岗区间计算出勤。
// 整月在岗时，公休额度内视为满勤；入职或离职的非整月则按实际在岗日减公休和请假。
func payrollActualDays(item response.PayrollRecord) string {
	leave, _ := strconv.ParseFloat(strings.TrimPrefix(item.LeaveDays, "¥"), 64)
	rest, _ := strconv.ParseFloat(strings.TrimPrefix(item.PublicRestDays, "¥"), 64)
	actual := float64(item.RequiredAttendanceDays)
	monthStart, err := time.Parse("2006-01", item.Month)
	if err != nil {
		allowedRest := math.Max(0, 30-float64(item.RequiredAttendanceDays))
		return strconv.Itoa(int(math.Floor(math.Max(0, actual-leave-math.Max(0, rest-allowedRest)))))
	}
	monthEnd := monthStart.AddDate(0, 1, -1)
	workStart, workEnd := monthStart, monthEnd
	if joinedOn, err := time.Parse("2006-01-02", item.JoinedOn); err == nil && joinedOn.After(workStart) {
		workStart = joinedOn
	}
	if leftOn, err := time.Parse("2006-01-02", item.LeftOn); err == nil && leftOn.Before(workEnd) {
		workEnd = leftOn
	}
	if workEnd.Before(workStart) {
		return "0"
	}
	if !workStart.Equal(monthStart) || !workEnd.Equal(monthEnd) {
		actual = float64(int(workEnd.Sub(workStart).Hours()/24) + 1)
		actual -= rest + leave
	} else {
		allowedRest := math.Max(0, 30-float64(item.RequiredAttendanceDays))
		excessRest := math.Max(0, rest-allowedRest)
		actual -= leave + excessRest
	}
	return strconv.Itoa(int(math.Floor(actual)))
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
			PayrollType string   `json:"payroll_type"`
			EmployeeID  uint64   `json:"employee_id"`
			EmployeeIDs []uint64 `json:"employee_ids"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		if input.PayrollType != "full_time" && input.PayrollType != "part_time" {
			httpx.Error(w, http.StatusBadRequest, "工资类型不正确")
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
				batchID, err = payroll.GenerateEmployee(r.Context(), input.Month, input.PayrollType, employeeID)
				if err != nil {
					break
				}
			}
		} else {
			batchID, err = payroll.Generate(r.Context(), input.Month, input.PayrollType)
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

// PayrollSettlementRevokeAPI 撤销一类员工工资的汇总流水并恢复待确认状态。
func (h *Handler) PayrollSettlementRevokeAPI(payroll service.Payroll, tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		var input request.PayrollSettlementRevokeInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := payroll.RevokeSettlement(r.Context(), actor, input); err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"month": input.Month, "payroll_type": input.PayrollType})
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
