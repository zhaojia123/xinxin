package response

type Employee struct {
	ID                             uint64 `json:"id"`
	DepartmentID                   uint64 `json:"department_id"`
	PositionID                     uint64 `json:"position_id"`
	EmployeeNo                     string `json:"employee_no"`
	Name                           string `json:"name"`
	Gender                         string `json:"gender"`
	IDCard                         string `json:"id_card"`
	Mobile                         string `json:"mobile"`
	Department                     string `json:"department"`
	Position                       string `json:"position"`
	Status                         string `json:"status"`
	StatusClass                    string `json:"status_class"`
	JoinedOn                       string `json:"joined_on"`
	RegularizedOn                  string `json:"regularized_on"`
	EmploymentType                 string `json:"employment_type"`
	Salary                         string `json:"salary"`
	SalaryValue                    string `json:"salary_value"`
	Education                      string `json:"education"`
	Hometown                       string `json:"hometown"`
	HealthCertificateID            uint64 `json:"health_certificate_id"`
	HealthCertificateURL           string `json:"health_certificate_url"`
	HealthCertificateIssuedOn      string `json:"health_certificate_issued_on"`
	HealthCertificateExpiresOn     string `json:"health_certificate_expires_on"`
	HealthCertificateStatus        string `json:"health_certificate_status"`
	HealthCertificateStatusClass   string `json:"health_certificate_status_class"`
	HealthCertificateDaysRemaining int    `json:"health_certificate_days_remaining"`
}
type HealthCertificateReminder struct {
	EmployeeID    uint64 `json:"employee_id"`
	EmployeeNo    string `json:"employee_no"`
	Name          string `json:"name"`
	ExpiresOn     string `json:"expires_on"`
	DaysRemaining int    `json:"days_remaining"`
	Status        string `json:"status"`
	StatusClass   string `json:"status_class"`
}
type EmployeeSummary struct {
	Total         int `json:"total"`
	Active        int `json:"active"`
	Probation     int `json:"probation"`
	LeftThisMonth int `json:"left_this_month"`
}
type EmployeeEvent struct {
	Date   string `json:"date"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Color  string `json:"color"`
}

type AttendanceRecord struct {
	ID          uint64 `json:"id"`
	EmployeeNo  string `json:"employee_no"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Type        string `json:"type"`
	Date        string `json:"date"`
	Time        string `json:"time"`
	Duration    string `json:"duration"`
	Reason      string `json:"reason"`
	Status      string `json:"status"`
	StatusClass string `json:"status_class"`
	Source      string `json:"source"`
}
type EmployeeMetric struct {
	EmployeeNo string `json:"employee_no"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	Detail     string `json:"detail"`
	Percent    int    `json:"percent"`
}
type BestEmployee struct {
	Rank  string `json:"rank"`
	Name  string `json:"name"`
	Value string `json:"value"`
}
type BestCategory struct {
	Title       string         `json:"title"`
	Icon        string         `json:"icon"`
	Description string         `json:"description"`
	Employees   []BestEmployee `json:"employees"`
}
type AttendanceStatistics struct {
	Period       string           `json:"period"`
	Label        string           `json:"label"`
	Leave        []EmployeeMetric `json:"leave"`
	Late         []EmployeeMetric `json:"late"`
	Early        []EmployeeMetric `json:"early"`
	Best         []BestCategory   `json:"best"`
	PendingCount int              `json:"pending_count"`
}

type EmploymentChange struct {
	ID          uint64 `json:"id"`
	EmployeeNo  string `json:"employee_no"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Before      string `json:"before"`
	After       string `json:"after"`
	EffectiveOn string `json:"effective_on"`
	Reason      string `json:"reason"`
	Operator    string `json:"operator"`
}
type ChangeSummary struct {
	JoinedThisMonth      int `json:"joined_this_month"`
	RegularizedThisMonth int `json:"regularized_this_month"`
	PromotedThisYear     int `json:"promoted_this_year"`
	LeftThisMonth        int `json:"left_this_month"`
}

type SalaryAdjustment struct {
	ID          uint64 `json:"id"`
	EmployeeNo  string `json:"employee_no"`
	Name        string `json:"name"`
	Before      string `json:"before"`
	After       string `json:"after"`
	Increase    string `json:"increase"`
	EffectiveOn string `json:"effective_on"`
	Reason      string `json:"reason"`
	Operator    string `json:"operator"`
}
type SalarySummary struct {
	AdjustedEmployees int    `json:"adjusted_employees"`
	AverageAdjustment string `json:"average_adjustment"`
	AveragePercent    string `json:"average_percent"`
	CurrentSalary     string `json:"current_salary"`
	PendingCount      int    `json:"pending_count"`
}

type PayrollRecord struct {
	ID                       uint64 `json:"id"`
	BatchID                  uint64 `json:"batch_id"`
	Month                    string `json:"month"`
	EmployeeNo               string `json:"employee_no"`
	Name                     string `json:"name"`
	BaseSalary               string `json:"base_salary"`
	Bonus                    string `json:"bonus"`
	AttendanceDeduction      string `json:"attendance_deduction"`
	OtherDeduction           string `json:"other_deduction"`
	Deduction                string `json:"deduction"`
	SocialSecurity           string `json:"social_security"`
	Tax                      string `json:"tax"`
	NetSalary                string `json:"net_salary"`
	Status                   string `json:"status"`
	StatusClass              string `json:"status_class"`
	BaseSalaryValue          string `json:"base_salary_value"`
	BonusValue               string `json:"bonus_value"`
	AttendanceDeductionValue string `json:"attendance_deduction_value"`
	OtherDeductionValue      string `json:"other_deduction_value"`
	SocialSecurityValue      string `json:"social_security_value"`
	TaxValue                 string `json:"tax_value"`
}
type PayrollSummary struct {
	BatchID         uint64 `json:"batch_id"`
	BatchStatus     string `json:"batch_status"`
	Month           string `json:"month"`
	GrossAmount     string `json:"gross_amount"`
	DeductionAmount string `json:"deduction_amount"`
	NetAmount       string `json:"net_amount"`
	EmployeeCount   int    `json:"employee_count"`
	PaidCount       int    `json:"paid_count"`
}

type Department struct {
	ID            uint64 `json:"id"`
	DepartmentNo  string `json:"department_no"`
	Name          string `json:"name"`
	ParentName    string `json:"parent_name"`
	ManagerName   string `json:"manager_name"`
	Status        string `json:"status"`
	StatusClass   string `json:"status_class"`
	Remark        string `json:"remark"`
	PositionCount int    `json:"position_count"`
	EmployeeCount int    `json:"employee_count"`
}
type Position struct {
	ID            uint64 `json:"id"`
	DepartmentID  uint64 `json:"department_id"`
	PositionNo    string `json:"position_no"`
	Name          string `json:"name"`
	Department    string `json:"department"`
	LevelName     string `json:"level_name"`
	SalaryRange   string `json:"salary_range"`
	Status        string `json:"status"`
	StatusClass   string `json:"status_class"`
	Remark        string `json:"remark"`
	EmployeeCount int    `json:"employee_count"`
}
type OrganizationSummary struct {
	DepartmentCount     int `json:"department_count"`
	PositionCount       int `json:"position_count"`
	ActiveEmployeeCount int `json:"active_employee_count"`
}

type LedgerRecord struct {
	ID             uint64 `json:"id"`
	No             int    `json:"no"`
	Date           string `json:"date"`
	Department     string `json:"department"`
	Summary        string `json:"summary"`
	Counterparty   string `json:"counterparty"`
	Month          string `json:"month"`
	Account        string `json:"account"`
	Category       string `json:"category"`
	VoucherNo      string `json:"voucher_no"`
	Income         string `json:"income"`
	Expense        string `json:"expense"`
	Balance        string `json:"balance"`
	PayrollBatchID uint64 `json:"payroll_batch_id"`
}
type LedgerSummary struct {
	Month          string `json:"month"`
	MonthIncome    string `json:"month_income"`
	MonthExpense   string `json:"month_expense"`
	MonthNet       string `json:"month_net"`
	CurrentBalance string `json:"current_balance"`
	YearIncome     string `json:"year_income"`
	YearExpense    string `json:"year_expense"`
	YearNet        string `json:"year_net"`
	MaxIncome      string `json:"max_income"`
	MaxExpense     string `json:"max_expense"`
	EntryCount     int    `json:"entry_count"`
}
type TrendMonth struct {
	Month          string `json:"month"`
	Income         string `json:"income"`
	Expense        string `json:"expense"`
	IncomePercent  int    `json:"income_percent"`
	ExpensePercent int    `json:"expense_percent"`
}
type AuditLog struct {
	ID          uint64 `json:"id"`
	Time        string `json:"time"`
	Operator    string `json:"operator"`
	Module      string `json:"module"`
	Action      string `json:"action"`
	Target      string `json:"target"`
	IP          string `json:"ip"`
	Result      string `json:"result"`
	ResultClass string `json:"result_class"`
}
