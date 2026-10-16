package handler

import (
	"encoding/csv"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"friends-records/internal/token"
	driver "github.com/go-sql-driver/mysql"
)

type EmployeesPageData struct {
	ActiveMenu      string
	Employees       []response.Employee
	Departments     []response.Department
	Positions       []response.Position
	Summary         response.EmployeeSummary
	HealthReminders []response.HealthCertificateReminder
	GlobalRestDays  int
}
type EmployeePageData struct {
	ActiveMenu, Mode, PageTitle string
	ID                          uint64
	Temporary                   bool
	Employee                    response.Employee
	Form                        request.EmployeeInput
	Error                       string
	Events                      []response.EmployeeEvent
	Departments                 []response.Department
	Positions                   []response.Position
}
type RecordsPageData struct {
	ActiveMenu         string
	Exceptions         []response.AttendanceRecord
	Statistics         response.AttendanceStatistics
	Changes            []response.EmploymentChange
	ChangeSummary      response.ChangeSummary
	Adjustments        []response.SalaryAdjustment
	SalarySummary      response.SalarySummary
	AuditLogs          []response.AuditLog
	AttendancePager    Pager
	ChangesPager       Pager
	AttendanceMonth    string
	AttendanceCategory string
	AttendanceType     string
	AttendanceStatus   string
	AttendanceSearch   string
	AttendanceTypes    []string
	AttendanceStatuses []string
	ChangeMonth        string
	ChangeType         string
	ChangeSearch       string
	ChangeTypes        []string
	SalaryMonth        string
	SalarySearch       string
}

type Pager struct {
	Page, PerPage, Total, PageCount int
	PrevPage, NextPage              int
	HasPrev, HasNext                bool
}

func pageNumber(r *http.Request) int {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	return page
}

func makePager(total, page, perPage int) Pager {
	if perPage < 1 {
		perPage = 20
	}
	pageCount := (total + perPage - 1) / perPage
	if pageCount == 0 {
		pageCount = 1
	}
	if page > pageCount {
		page = pageCount
	}
	return Pager{Page: page, PerPage: perPage, Total: total, PageCount: pageCount, PrevPage: page - 1, NextPage: page + 1, HasPrev: page > 1, HasNext: page < pageCount}
}

func pageAttendance(items []response.AttendanceRecord, page int) ([]response.AttendanceRecord, Pager) {
	pager := makePager(len(items), page, 20)
	start := (pager.Page - 1) * pager.PerPage
	if start > len(items) {
		start = len(items)
	}
	end := start + pager.PerPage
	if end > len(items) {
		end = len(items)
	}
	result := items[start:end]
	for i := range result {
		result[i].Seq = start + i + 1
	}
	return result, pager
}

func pageChanges(items []response.EmploymentChange, page int) ([]response.EmploymentChange, Pager) {
	pager := makePager(len(items), page, 20)
	start := (pager.Page - 1) * pager.PerPage
	if start > len(items) {
		start = len(items)
	}
	end := start + pager.PerPage
	if end > len(items) {
		end = len(items)
	}
	result := items[start:end]
	for i := range result {
		result[i].Seq = start + i + 1
	}
	return result, pager
}

func includesKeyword(keyword string, values ...string) bool {
	if keyword == "" {
		return true
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

func filterAttendance(items []response.AttendanceRecord, month, category, recordType, status, search string) []response.AttendanceRecord {
	month, search = strings.TrimSpace(month), strings.ToLower(strings.TrimSpace(search))
	result := make([]response.AttendanceRecord, 0, len(items))
	for _, item := range items {
		if month != "" && !strings.HasPrefix(item.Date, month) {
			continue
		}
		if category != "" {
			matched := (category == "leave" && item.Category == "请假" && item.Type != "特殊请假") ||
				(category == "public_rest" && item.Type == "公休") ||
				(category == "holiday" && item.Type == "节假日") ||
				(category == "special_leave" && item.Type == "特殊请假")
			if !matched {
				continue
			}
		}
		if recordType != "" && item.Type != recordType {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		if !includesKeyword(search, item.Name, item.EmployeeNo, item.Type, item.Reason, item.Date, item.Duration, item.Source) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func filterChanges(items []response.EmploymentChange, month, changeType, search string) []response.EmploymentChange {
	month, changeType, search = strings.TrimSpace(month), strings.TrimSpace(changeType), strings.ToLower(strings.TrimSpace(search))
	result := make([]response.EmploymentChange, 0, len(items))
	for _, item := range items {
		if month != "" && !strings.HasPrefix(item.EffectiveOn, month) {
			continue
		}
		if changeType != "" && item.Type != changeType {
			continue
		}
		if !includesKeyword(search, item.Name, item.EmployeeNo, item.Type, item.Before, item.After, item.Reason, item.Operator, item.EffectiveOn) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func filterSalary(items []response.SalaryAdjustment, month, search string) []response.SalaryAdjustment {
	month, search = strings.TrimSpace(month), strings.ToLower(strings.TrimSpace(search))
	result := make([]response.SalaryAdjustment, 0, len(items))
	for _, item := range items {
		if month != "" && !strings.HasPrefix(item.EffectiveOn, month) {
			continue
		}
		if !includesKeyword(search, item.Name, item.EmployeeNo, item.Before, item.After, item.Increase, item.Reason, item.Operator, item.EffectiveOn) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func filterPayroll(items []response.PayrollRecord, status, search string) []response.PayrollRecord {
	status, search = strings.TrimSpace(status), strings.ToLower(strings.TrimSpace(search))
	result := make([]response.PayrollRecord, 0, len(items))
	for _, item := range items {
		if status != "" && item.Status != status {
			continue
		}
		if !includesKeyword(search, item.Name, item.EmployeeNo, item.EntrySalary, item.BaseSalary, item.NetSalary) {
			continue
		}
		result = append(result, item)
	}
	for i := range result {
		result[i].Seq = i + 1
	}
	return result
}

func (h *Handler) EmployeesPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.Employees(r.Context())
	if err != nil {
		fail(w, err, "员工数据读取失败")
		return
	}
	summary, err := h.Store.EmployeeSummary(r.Context())
	if err != nil {
		fail(w, err, "员工统计读取失败")
		return
	}
	reminders, err := h.Store.HealthCertificateReminders(r.Context())
	if err != nil {
		fail(w, err, "健康证到期提醒读取失败")
		return
	}
	departments, err := h.Store.Departments(r.Context())
	if err != nil {
		fail(w, err, "部门筛选项读取失败")
		return
	}
	positions, err := h.Store.Positions(r.Context())
	if err != nil {
		fail(w, err, "岗位筛选项读取失败")
		return
	}
	h.render(w, "employees.html", EmployeesPageData{ActiveMenu: "employees", Employees: items, Departments: departments, Positions: positions, Summary: summary, HealthReminders: reminders, GlobalRestDays: h.Store.MonthlyRestDays})
}
func (h *Handler) EmployeeDetail(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	id, _ := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	item, found, err := h.Store.Employee(r.Context(), id)
	if err != nil {
		fail(w, err, "员工详情读取失败")
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	events, err := h.Store.EmployeeEvents(r.Context(), id)
	if err != nil {
		fail(w, err, "员工动态读取失败")
		return
	}
	h.render(w, "employee_detail.html", EmployeePageData{ActiveMenu: "employees", PageTitle: "员工详情", Employee: item, Events: events})
}
func (h *Handler) EmployeeForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodGet, http.MethodPost)
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
	temporary := r.URL.Query().Get("temporary") == "1"
	data := EmployeePageData{ActiveMenu: "employees", Mode: "new", PageTitle: "新增员工", Temporary: temporary, Departments: departments, Positions: positions}
	data.Form.Gender, data.Form.EmploymentStatus, data.Form.EmploymentType, data.Form.PayBasis, data.Form.CurrentSalary, data.Form.JoinedOn = "unknown", "probation", "full_time", "monthly", "", time.Now().Format("2006-01-02")
	if temporary {
		data.PageTitle = "新增临时工"
		data.Form.EmploymentStatus, data.Form.EmploymentType, data.Form.PayBasis = "active", "part_time", "hourly"
		data.Form.EmployeeNo = generatedEmployeeNo("TMP")
	}
	var id uint64
	var originalForm request.EmployeeInput
	if r.URL.Path == "/admin/employees/edit" {
		id, _ = strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
		data.ID = id
		item, err := h.Store.MiniEmployee(r.Context(), id)
		if err != nil {
			fail(w, err, "员工数据读取失败")
			return
		}
		item.EmployeeInput.Gender = request.NormalizeGender(item.EmployeeInput.Gender)
		data.Mode, data.PageTitle, data.Form = "edit", "编辑员工", item.EmployeeInput
		originalForm = item.EmployeeInput
		// 兼职档案使用精简临时工表单，正式员工仍保留完整档案字段。
		data.Temporary = temporary || item.EmployeeInput.EmploymentType == "part_time"
	}
	renderForm := func() {
		name := "employee_form.html"
		if data.Temporary {
			name = "temporary_employee_form.html"
		}
		h.render(w, name, data)
	}
	if r.Method == http.MethodGet {
		renderForm()
		return
	}
	// 员工表单带有文件上传区域，优先完整解析 multipart 字段，确保编辑时字段都能被读取。
	var parseErr error
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		parseErr = r.ParseMultipartForm(32 << 20)
	} else {
		parseErr = r.ParseForm()
	}
	if parseErr != nil {
		data.Error = "提交内容格式不正确"
		renderForm()
		return
	}
	data.Form = request.EmployeeInput{
		EmployeeNo: r.FormValue("employee_no"), Name: r.FormValue("name"), Gender: r.FormValue("gender"),
		IDCard: r.FormValue("id_card"), Mobile: r.FormValue("mobile"), EmploymentStatus: r.FormValue("employment_status"),
		EmploymentType: r.FormValue("employment_type"), JoinedOn: r.FormValue("joined_on"), RegularizedOn: r.FormValue("regularized_on"),
		LeftOn: r.FormValue("left_on"), EntrySalary: r.FormValue("entry_salary"), CurrentSalary: r.FormValue("current_salary"), PayBasis: r.FormValue("pay_basis"), Education: r.FormValue("education"),
		Hometown: r.FormValue("hometown"), Remark: r.FormValue("remark"),
	}
	data.Form.DepartmentID, _ = strconv.ParseUint(r.FormValue("department_id"), 10, 64)
	data.Form.PositionID, _ = strconv.ParseUint(r.FormValue("position_id"), 10, 64)
	data.Form.MonthlyRestDays, _ = strconv.Atoi(r.FormValue("monthly_rest_days"))
	data.Temporary = temporary || r.FormValue("temporary") == "1"
	if strings.TrimSpace(data.Form.EmployeeNo) == "" {
		data.Form.EmployeeNo = generatedEmployeeNo("EMP")
	}
	if data.Form.JoinedOn == "" {
		data.Form.JoinedOn = time.Now().Format("2006-01-02")
	}
	if data.Temporary {
		data.PageTitle = "新增临时工"
		data.Form.EmployeeNo = strings.TrimSpace(data.Form.EmployeeNo)
		if data.Form.EmployeeNo == "" {
			data.Form.EmployeeNo = generatedEmployeeNo("TMP")
		}
		data.Form.Gender = "unknown"
		data.Form.EmploymentStatus = "active"
		data.Form.EmploymentType = "part_time"
		// 临时工工资仅按每日工时汇总，不参与正式工公休规则。
		data.Form.MonthlyRestDays = 0
	}
	if id > 0 {
		// 编辑时只更新实际提交的字段；浏览器若只提交变更字段，其余资料沿用原值，避免误清空员工档案。
		preserve := func(name string, target *string, original string) {
			if !r.Form.Has(name) {
				*target = original
			}
		}
		preserve("employee_no", &data.Form.EmployeeNo, originalForm.EmployeeNo)
		preserve("name", &data.Form.Name, originalForm.Name)
		preserve("gender", &data.Form.Gender, originalForm.Gender)
		preserve("mobile", &data.Form.Mobile, originalForm.Mobile)
		preserve("id_card", &data.Form.IDCard, originalForm.IDCard)
		preserve("employment_status", &data.Form.EmploymentStatus, originalForm.EmploymentStatus)
		preserve("employment_type", &data.Form.EmploymentType, originalForm.EmploymentType)
		preserve("pay_basis", &data.Form.PayBasis, originalForm.PayBasis)
		preserve("joined_on", &data.Form.JoinedOn, originalForm.JoinedOn)
		preserve("regularized_on", &data.Form.RegularizedOn, originalForm.RegularizedOn)
		preserve("left_on", &data.Form.LeftOn, originalForm.LeftOn)
		preserve("entry_salary", &data.Form.EntrySalary, originalForm.EntrySalary)
		preserve("current_salary", &data.Form.CurrentSalary, originalForm.CurrentSalary)
		preserve("education", &data.Form.Education, originalForm.Education)
		preserve("hometown", &data.Form.Hometown, originalForm.Hometown)
		preserve("remark", &data.Form.Remark, originalForm.Remark)
		if !r.Form.Has("department_id") {
			data.Form.DepartmentID = originalForm.DepartmentID
		}
		if !r.Form.Has("position_id") {
			data.Form.PositionID = originalForm.PositionID
		}
		if !data.Temporary && !r.Form.Has("monthly_rest_days") {
			data.Form.MonthlyRestDays = originalForm.MonthlyRestDays
		}
	}
	if err := data.Form.Validate(); err != nil {
		data.Error = err.Error()
		renderForm()
		return
	}
	if _, err := h.Store.SaveEmployee(r.Context(), id, 0, data.Form); err != nil {
		var input mysql.MiniInputError
		var dbErr *driver.MySQLError
		switch {
		case errors.As(err, &input):
			data.Error = input.Message
		case errors.As(err, &dbErr) && dbErr.Number == 1062:
			data.Error = "员工编号已存在，请换一个编号"
		default:
			data.Error = "员工资料保存失败，请稍后重试"
		}
		renderForm()
		return
	}
	http.Redirect(w, r, "/admin/employees", http.StatusSeeOther)
}

// temporaryEmployeeNo 为临时工生成内部编号，用户无需手工维护编号。
func temporaryEmployeeNo() string {
	return generatedEmployeeNo("TMP")
}

// generatedEmployeeNo 由系统生成唯一的内部编号，录入员工时无需手填。
func generatedEmployeeNo(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102-150405.000000000")
}
func (h *Handler) LeavesPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.AttendanceRecords(r.Context())
	if err != nil {
		fail(w, err, "请假与公休记录读取失败")
		return
	}
	month := r.URL.Query().Get("month")
	if _, parseErr := time.Parse("2006-01", month); parseErr != nil {
		month = time.Now().Format("2006-01")
	}
	statistics, err := h.Store.AttendanceStatisticsAt(r.Context(), r.URL.Query().Get("period"), month)
	if err != nil {
		fail(w, err, "员工出勤统计读取失败")
		return
	}
	category, recordType, status, search := r.URL.Query().Get("category"), r.URL.Query().Get("record_type"), r.URL.Query().Get("status"), r.URL.Query().Get("q")
	items = filterAttendance(items, month, category, recordType, status, search)
	items, pager := pageAttendance(items, pageNumber(r))
	h.render(w, "leaves.html", RecordsPageData{ActiveMenu: "leaves", Exceptions: items, Statistics: statistics, AttendancePager: pager, AttendanceMonth: month, AttendanceCategory: category, AttendanceType: recordType, AttendanceStatus: status, AttendanceSearch: search, AttendanceTypes: []string{"公休", "请假", "特殊请假", "节假日"}, AttendanceStatuses: []string{"已通过", "已记录", "已确认", "已驳回"}})
}

// LeavesExport 导出全员逐日出勤表，并在文件中附带四类记录标记说明。
func (h *Handler) LeavesExport(tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := adminActor(w, r, tokens); !ok {
			return
		}
		if r.Method != http.MethodGet {
			httpx.MethodNotAllowed(w, http.MethodGet)
			return
		}
		month := mysql.NormalizeMonth(r.URL.Query().Get("month"))
		rows, days, err := h.Store.AttendanceCalendar(r.Context(), month)
		if err != nil {
			fail(w, err, "出勤表导出失败")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape("全员出勤_"+month+".csv"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("\xEF\xBB\xBF"))
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"标记说明", "√ 正常出勤", "○ 公休", "✕ 请假", "◎ 特殊请假补助", "☆ 法定节假日"})
		header := []string{"序号", "工号", "姓名"}
		for i := 1; i <= days; i++ {
			header = append(header, strconv.Itoa(i)+"日")
		}
		_ = writer.Write(header)
		for i, row := range rows {
			record := []string{strconv.Itoa(i + 1), row.EmployeeNo, row.Name}
			record = append(record, row.Marks...)
			_ = writer.Write(record)
		}
		writer.Flush()
	}
}
func (h *Handler) ChangesPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.EmploymentChanges(r.Context())
	if err != nil {
		fail(w, err, "人事异动读取失败")
		return
	}
	summary, err := h.Store.ChangeSummary(r.Context())
	if err != nil {
		fail(w, err, "人事异动统计读取失败")
		return
	}
	month, changeType, search := r.URL.Query().Get("month"), r.URL.Query().Get("change_type"), r.URL.Query().Get("q")
	items = filterChanges(items, month, changeType, search)
	items, pager := pageChanges(items, pageNumber(r))
	h.render(w, "changes.html", RecordsPageData{ActiveMenu: "changes", Changes: items, ChangeSummary: summary, ChangesPager: pager, ChangeMonth: month, ChangeType: changeType, ChangeSearch: search, ChangeTypes: []string{"入职", "转正", "调岗", "晋升", "降职", "离职"}})
}
func (h *Handler) SalaryPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.SalaryAdjustments(r.Context())
	if err != nil {
		fail(w, err, "调薪记录读取失败")
		return
	}
	summary, err := h.Store.SalarySummary(r.Context())
	if err != nil {
		fail(w, err, "调薪统计读取失败")
		return
	}
	month, search := r.URL.Query().Get("month"), r.URL.Query().Get("q")
	items = filterSalary(items, month, search)
	for i := range items {
		items[i].Seq = i + 1
	}
	h.render(w, "salary_adjustments.html", RecordsPageData{ActiveMenu: "salary", Adjustments: items, SalarySummary: summary, SalaryMonth: month, SalarySearch: search})
}
func (h *Handler) AuditPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.AuditLogs(r.Context())
	if err != nil {
		fail(w, err, "操作日志读取失败")
		return
	}
	h.render(w, "audit_logs.html", RecordsPageData{ActiveMenu: "audit", AuditLogs: items})
}

func (h *Handler) EmployeesAPI(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.Employees(r.Context())
	if err != nil {
		fail(w, err, "员工数据读取失败")
		return
	}
	reminders, err := h.Store.HealthCertificateReminders(r.Context())
	if err != nil {
		fail(w, err, "健康证到期提醒读取失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"employees": items, "health_certificate_reminders": reminders})
}
func (h *Handler) AttendanceAPI(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.AttendanceRecords(r.Context())
	if err != nil {
		fail(w, err, "请假与公休记录读取失败")
		return
	}
	statistics, err := h.Store.AttendanceStatistics(r.Context(), r.URL.Query().Get("period"))
	if err != nil {
		fail(w, err, "员工出勤统计读取失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"records": items, "statistics": statistics})
}
func (h *Handler) ChangesAPI(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.EmploymentChanges(r.Context())
	if err != nil {
		fail(w, err, "人事异动读取失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"changes": items})
}
func (h *Handler) SalaryAPI(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.SalaryAdjustments(r.Context())
	if err != nil {
		fail(w, err, "调薪记录读取失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"adjustments": items})
}
func (h *Handler) AuditAPI(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.AuditLogs(r.Context())
	if err != nil {
		fail(w, err, "操作日志读取失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"logs": items})
}
