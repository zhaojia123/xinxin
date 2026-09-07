package router

import (
	"net/http"

	"friends-records/internal/handler"
	modelmysql "friends-records/internal/models/mysql"
	"friends-records/internal/service"
)

// registerAdminRoutes 只注册后台 HTML 页面和 /api/admin 接口。
func registerAdminRoutes(mux *http.ServeMux, h *handler.Handler, store *modelmysql.Store) {
	admin := service.Admin{Store: store}
	payroll := service.Payroll{Store: store}

	// GET /admin：跳转后台登录页。
	mux.HandleFunc("/admin", h.AdminRoot)
	// GET /admin/login：显示登录页；POST /admin/login：提交后台登录表单。
	mux.HandleFunc("/admin/login", h.AdminLoginPage(admin))
	// GET /admin/employees：员工档案列表和健康证到期提醒。
	mux.HandleFunc("/admin/employees", h.EmployeesPage)
	// GET /admin/employees/view：员工详情、健康证和员工动态。
	mux.HandleFunc("/admin/employees/view", h.EmployeeDetail)
	// GET/POST /admin/employees/new：新增员工页面和表单提交。
	mux.HandleFunc("/admin/employees/new", h.EmployeeForm)
	// GET/POST /admin/employees/edit：编辑员工页面和表单提交。
	mux.HandleFunc("/admin/employees/edit", h.EmployeeForm)
	// GET /admin/organization：部门和岗位管理页面。
	mux.HandleFunc("/admin/organization", h.OrganizationPage)
	// GET /admin/leaves：请假、迟到、早退记录与统计页面。
	mux.HandleFunc("/admin/leaves", h.LeavesPage)
	// GET /admin/changes：人事异动页面。
	mux.HandleFunc("/admin/changes", h.ChangesPage)
	// GET /admin/salary-adjustments：调薪记录页面。
	mux.HandleFunc("/admin/salary-adjustments", h.SalaryPage)
	// GET /admin/payroll：月度工资编辑页面。
	mux.HandleFunc("/admin/payroll", h.PayrollPage)
	// GET /admin/ledger：台账流水和收支统计页面。
	mux.HandleFunc("/admin/ledger", h.LedgerPage)
	// GET /admin/audit-logs：后台操作日志页面。
	mux.HandleFunc("/admin/audit-logs", h.AuditPage)

	// GET/POST/PUT /api/admin/employees：查询、新增和修改员工档案。
	mux.HandleFunc("/api/admin/employees", h.AdminEmployeesAPI)
	mux.HandleFunc("/api/admin/employees/leave", h.AdminEmployeeLeaveAPI)
	mux.HandleFunc("/api/admin/options", h.AdminOptionsAPI)
	mux.HandleFunc("/api/admin/departments", h.AdminDepartmentsAPI)
	mux.HandleFunc("/api/admin/positions", h.AdminPositionsAPI)
	// GET /api/admin/organization：查询部门和岗位。
	mux.HandleFunc("/api/admin/organization", h.OrganizationAPI)
	// GET/POST/PUT /api/admin/attendance：查询、新增和修改请假异常。
	mux.HandleFunc("/api/admin/attendance", h.AdminAttendanceAPI)
	mux.HandleFunc("/api/admin/attendance/status", h.AdminAttendanceStatusAPI)
	// GET/POST /api/admin/employment-changes：查询和新增人事异动。
	mux.HandleFunc("/api/admin/employment-changes", h.AdminChangesAPI)
	// GET/POST /api/admin/salary-adjustments：查询调薪历史和新增调薪。
	mux.HandleFunc("/api/admin/salary-adjustments", h.AdminSalaryAPI)
	// GET /api/admin/payroll：查询月度工资；PUT：编辑一条工资明细。
	mux.HandleFunc("/api/admin/payroll", h.PayrollAPI(payroll))
	// POST /api/admin/payroll/generate：生成指定月份的员工工资快照。
	mux.HandleFunc("/api/admin/payroll/generate", h.PayrollGenerateAPI(payroll))
	// POST /api/admin/payroll/pay：发放工资并自动生成关联台账支出。
	mux.HandleFunc("/api/admin/payroll/pay", h.PayrollPayAPI(payroll))
	mux.HandleFunc("/api/admin/payroll/confirm", h.PayrollConfirmAPI(payroll))
	// GET/POST/PUT /api/admin/ledger：查询、新增和修改台账记录。
	mux.HandleFunc("/api/admin/ledger", h.AdminLedgerAPI)
	// GET /api/admin/audit-logs：查询操作日志。
	mux.HandleFunc("/api/admin/audit-logs", h.AuditAPI)
}
