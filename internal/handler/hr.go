package handler

import (
	"errors"
	"net/http"
	"strconv"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
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
	Employee                    response.Employee
	Form                        request.EmployeeInput
	Error                       string
	Events                      []response.EmployeeEvent
	Departments                 []response.Department
	Positions                   []response.Position
}
type RecordsPageData struct {
	ActiveMenu      string
	Exceptions      []response.AttendanceRecord
	Statistics      response.AttendanceStatistics
	Changes         []response.EmploymentChange
	ChangeSummary   response.ChangeSummary
	Adjustments     []response.SalaryAdjustment
	SalarySummary   response.SalarySummary
	AuditLogs       []response.AuditLog
	AttendancePager Pager
	ChangesPager    Pager
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
	data := EmployeePageData{ActiveMenu: "employees", Mode: "new", PageTitle: "新增员工", Departments: departments, Positions: positions}
	data.Form.Gender, data.Form.EmploymentStatus, data.Form.EmploymentType, data.Form.PayBasis, data.Form.CurrentSalary = "unknown", "probation", "full_time", "monthly", ""
	var id uint64
	if r.URL.Path == "/admin/employees/edit" {
		id, _ = strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
		data.ID = id
		item, err := h.Store.MiniEmployee(r.Context(), id)
		if err != nil {
			fail(w, err, "员工数据读取失败")
			return
		}
		data.Mode, data.PageTitle, data.Form = "edit", "编辑员工", item.EmployeeInput
	}
	if r.Method == http.MethodGet {
		h.render(w, "employee_form.html", data)
		return
	}
	if err := r.ParseForm(); err != nil {
		data.Error = "提交内容格式不正确"
		h.render(w, "employee_form.html", data)
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
	if err := data.Form.Validate(); err != nil {
		data.Error = err.Error()
		h.render(w, "employee_form.html", data)
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
		h.render(w, "employee_form.html", data)
		return
	}
	http.Redirect(w, r, "/admin/employees", http.StatusSeeOther)
}
func (h *Handler) LeavesPage(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	items, err := h.Store.AttendanceRecords(r.Context())
	if err != nil {
		fail(w, err, "请假与异常记录读取失败")
		return
	}
	statistics, err := h.Store.AttendanceStatistics(r.Context(), r.URL.Query().Get("period"))
	if err != nil {
		fail(w, err, "员工出勤统计读取失败")
		return
	}
	items, pager := pageAttendance(items, pageNumber(r))
	h.render(w, "leaves.html", RecordsPageData{ActiveMenu: "leaves", Exceptions: items, Statistics: statistics, AttendancePager: pager})
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
	items, pager := pageChanges(items, pageNumber(r))
	h.render(w, "changes.html", RecordsPageData{ActiveMenu: "changes", Changes: items, ChangeSummary: summary, ChangesPager: pager})
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
	h.render(w, "salary_adjustments.html", RecordsPageData{ActiveMenu: "salary", Adjustments: items, SalarySummary: summary})
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
		fail(w, err, "请假与异常记录读取失败")
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
