package router

import (
	"net/http"

	"friends-records/internal/handler"
	modelmysql "friends-records/internal/models/mysql"
	"friends-records/internal/service"
	"friends-records/internal/token"
)

// registerAdminRoutes 只注册后台 HTML 页面和 /api/admin 接口。
func registerAdminRoutes(mux *http.ServeMux, h *handler.Handler, store *modelmysql.Store, tokens *token.Manager) {
	admin := service.Admin{Store: store}
	payroll := service.Payroll{Store: store}

	// GET /admin：跳转后台登录页。
	mux.HandleFunc("/admin", h.AdminRoot)
	// GET /admin/login：显示登录页；POST /admin/login：提交后台登录表单。
	mux.HandleFunc("/admin/login", h.AdminLoginPage(admin, tokens))
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
	mux.HandleFunc("/admin/leaves/export", h.LeavesExport(tokens))
	// GET /admin/changes：人事异动页面。
	mux.HandleFunc("/admin/changes", h.ChangesPage)
	// GET /admin/salary-adjustments：调薪记录页面。
	mux.HandleFunc("/admin/salary-adjustments", h.SalaryPage)
	// GET /admin/payroll：月度工资编辑页面。
	mux.HandleFunc("/admin/payroll", h.PayrollPage)
	// GET /admin/work-logs：小时工每日工时记录。
	mux.HandleFunc("/admin/work-logs", h.WorkLogsPage)
	mux.HandleFunc("/admin/work-logs/export", h.WorkLogsExport(tokens))
	mux.HandleFunc("/admin/payroll/export", h.PayrollExport(tokens))
	// GET /admin/ledger：台账流水和收支统计页面。
	mux.HandleFunc("/admin/ledger", h.LedgerPage)
	// GET /admin/stalls：档口基础资料和抽成规则。
	mux.HandleFunc("/admin/stalls", h.StallsPage)
	// GET /admin/stalls/monthly：指定月份的营业额和抽成计算。
	mux.HandleFunc("/admin/stalls/monthly", h.StallsMonthlyPage)
	// GET /admin/stall-costs：三个自营档口的月度成本。
	mux.HandleFunc("/admin/stall-costs", h.StallCostsPage)
	// GET /admin/menu：菜单菜品列表和维护页面。
	mux.HandleFunc("/admin/menu", h.MenuPage)
	// GET /admin/purchases：每日供货商采购清单。
	mux.HandleFunc("/admin/purchases", h.PurchasesPage)
	mux.HandleFunc("/admin/purchases/export", h.PurchaseExport(tokens))
	// GET /admin/audit-logs：后台操作日志页面。
	mux.HandleFunc("/admin/audit-logs", h.AuditPage)
	// GET /admin/mini-users：小程序用户与模块权限管理页面。
	mux.HandleFunc("/admin/mini-users", h.MiniUsersPage)

	// GET/POST/PUT/DELETE /api/admin/employees：查询、新增、修改和安全删除员工档案。
	mux.HandleFunc("/api/admin/employees", h.AdminEmployeesAPI(tokens))
	mux.HandleFunc("/api/admin/employees/leave", h.AdminEmployeeLeaveAPI)
	mux.HandleFunc("/api/admin/options", h.AdminOptionsAPI)
	mux.HandleFunc("/api/admin/departments", h.AdminDepartmentsAPI(tokens))
	mux.HandleFunc("/api/admin/positions", h.AdminPositionsAPI(tokens))
	// GET /api/admin/organization：查询部门和岗位。
	mux.HandleFunc("/api/admin/organization", h.OrganizationAPI)
	// GET/POST/PUT/DELETE /api/admin/attendance：查询、新增、修改和删除请假异常。
	mux.HandleFunc("/api/admin/attendance", h.AdminAttendanceAPI(tokens))
	mux.HandleFunc("/api/admin/attendance/status", h.AdminAttendanceStatusAPI)
	// GET/POST /api/admin/employment-changes：查询和新增人事异动。
	mux.HandleFunc("/api/admin/employment-changes", h.AdminChangesAPI)
	// GET/POST /api/admin/salary-adjustments：查询调薪历史和新增调薪。
	mux.HandleFunc("/api/admin/salary-adjustments", h.AdminSalaryAPI)
	// GET/PUT/DELETE /api/admin/payroll：查询、编辑和删除未发放工资明细。
	mux.HandleFunc("/api/admin/payroll", h.PayrollAPI(payroll, tokens))
	// GET/POST/PUT/DELETE /api/admin/work-logs：小时工每日工时增删改查。
	mux.HandleFunc("/api/admin/work-logs", h.WorkLogsAPI(tokens))
	// POST /api/admin/payroll/generate：生成指定月份的员工工资快照，可选 employee_id 单独生成一人。
	mux.HandleFunc("/api/admin/payroll/generate", h.PayrollGenerateAPI(payroll))
	// POST /api/admin/payroll/pay：保留旧地址并提示改由档口月度核算统一入账。
	mux.HandleFunc("/api/admin/payroll/pay", h.PayrollPayAPI(payroll))
	mux.HandleFunc("/api/admin/payroll/temporary-pay", h.TemporaryPayrollAPI(payroll))
	mux.HandleFunc("/api/admin/payroll/confirm", h.PayrollConfirmAPI(payroll))
	mux.HandleFunc("/api/admin/payroll/settlement", h.PayrollSettlementAPI(payroll, tokens))
	mux.HandleFunc("/api/admin/payroll/settlement/revoke", h.PayrollSettlementRevokeAPI(payroll, tokens))
	mux.HandleFunc("/api/admin/special-payroll", h.SpecialPayrollAPI(payroll, tokens))
	mux.HandleFunc("/api/admin/special-payroll/settlement", h.SpecialPayrollSettlementAPI(payroll, tokens))
	// GET/POST/PUT/DELETE /api/admin/ledger：查询、新增、修改和删除普通台账记录。
	mux.HandleFunc("/api/admin/ledger", h.AdminLedgerAPI(tokens))
	// GET/POST/PUT/DELETE /api/admin/stalls：档口基础资料。
	mux.HandleFunc("/api/admin/stalls", h.StallsAPI(tokens))
	mux.HandleFunc("/api/admin/stall-settings", h.StallSettingsAPI(tokens))
	mux.HandleFunc("/api/admin/stall-monthly-rules", h.StallMonthlyRuleAPI(tokens))
	// POST/PUT/DELETE /api/admin/stall-turnovers：档口月度营业额。
	mux.HandleFunc("/api/admin/stall-turnovers", h.StallTurnoverAPI(tokens))
	mux.HandleFunc("/api/admin/stall-costs", h.StallCostsAPI(tokens))
	mux.HandleFunc("/api/admin/stall-costs/ledger", h.StallCostLedgerAPI(tokens))
	mux.HandleFunc("/api/admin/stall-monthly-expenses", h.StallMonthlyExpenseAPI(tokens))
	mux.HandleFunc("/api/admin/stall-monthly-expenses/ledger", h.StallMonthlyExpenseLedgerAPI(tokens))
	mux.HandleFunc("/api/admin/stall-monthly-expenses/ledger/revoke", h.StallMonthlyExpenseLedgerRevokeAPI(tokens))
	// GET/POST/PUT/DELETE /api/admin/menu-items：菜单查询和增删改。
	mux.HandleFunc("/api/admin/menu-items", h.MenuAPI(tokens))
	// GET/POST/PUT/DELETE /api/admin/purchases：查询、新增、修改和假删除采购明细。
	mux.HandleFunc("/api/admin/purchases", h.AdminPurchasesAPI(tokens))
	// GET /api/admin/audit-logs：查询操作日志。
	mux.HandleFunc("/api/admin/audit-logs", h.AuditAPI)
	// GET/PUT /api/admin/mini-users：查询和更新小程序用户权限。
	mux.HandleFunc("/api/admin/mini-users", h.AdminMiniUsersAPI(tokens))
}
