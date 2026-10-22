package response

type Employee struct {
	Seq                            int                  `json:"seq,omitempty"`
	ID                             uint64               `json:"id"`
	DepartmentID                   uint64               `json:"department_id"`
	PositionID                     uint64               `json:"position_id"`
	EmployeeNo                     string               `json:"employee_no"`
	Name                           string               `json:"name"`
	Gender                         string               `json:"gender"`
	IDCard                         string               `json:"id_card"`
	Mobile                         string               `json:"mobile"`
	Department                     string               `json:"department"`
	Position                       string               `json:"position"`
	Status                         string               `json:"status"`
	StatusClass                    string               `json:"status_class"`
	JoinedOn                       string               `json:"joined_on"`
	RegularizedOn                  string               `json:"regularized_on"`
	LeftOn                         string               `json:"left_on"`
	EmploymentType                 string               `json:"employment_type"`
	PayBasis                       string               `json:"pay_basis"`
	MonthlyRestDays                int                  `json:"monthly_rest_days"`
	EntrySalary                    string               `json:"entry_salary"`
	Salary                         string               `json:"salary"`
	SalaryValue                    string               `json:"salary_value"`
	Education                      string               `json:"education"`
	Hometown                       string               `json:"hometown"`
	HealthCertificateID            uint64               `json:"health_certificate_id"`
	HealthCertificateURL           string               `json:"health_certificate_url"`
	HealthCertificateIssuedOn      string               `json:"health_certificate_issued_on"`
	HealthCertificateExpiresOn     string               `json:"health_certificate_expires_on"`
	HealthCertificateStatus        string               `json:"health_certificate_status"`
	HealthCertificateStatusClass   string               `json:"health_certificate_status_class"`
	HealthCertificateDaysRemaining int                  `json:"health_certificate_days_remaining"`
	Attachments                    []EmployeeAttachment `json:"attachments"`
}
type EmployeeAttachment struct {
	ID             uint64 `json:"id"`
	EmployeeID     uint64 `json:"employee_id"`
	AttachmentType string `json:"attachment_type"`
	Title          string `json:"title"`
	FileURL        string `json:"file_url"`
	OriginalName   string `json:"original_name"`
	MIMEType       string `json:"mime_type"`
	FileSize       int64  `json:"file_size"`
}

type WorkLog struct {
	ID            uint64 `json:"id"`
	EmployeeID    uint64 `json:"employee_id"`
	EmployeeNo    string `json:"employee_no"`
	Name          string `json:"name"`
	WorkDate      string `json:"work_date"`
	Hours         string `json:"hours"`
	HoursValue    string `json:"hours_value"`
	PayBasis      string `json:"pay_basis"`
	PayBasisLabel string `json:"pay_basis_label"`
	PayRate       string `json:"pay_rate"`
	WorkLabel     string `json:"work_label"`
	Amount        string `json:"amount"`
	AmountValue   string `json:"amount_value"`
	Remark        string `json:"remark"`
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
	Seq          int    `json:"seq,omitempty"`
	ID           uint64 `json:"id"`
	EmployeeNo   string `json:"employee_no"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Type         string `json:"type"`
	Date         string `json:"date"`
	Time         string `json:"time"`
	Duration     string `json:"duration"`
	Reason       string `json:"reason"`
	Status       string `json:"status"`
	StatusClass  string `json:"status_class"`
	Source       string `json:"source"`
	SalaryEffect string `json:"salary_effect"`
	Subsidy      string `json:"subsidy"`
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
	Seq         int    `json:"seq,omitempty"`
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
	Seq         int    `json:"seq,omitempty"`
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
	Seq                      int    `json:"seq,omitempty"`
	ID                       uint64 `json:"id"`
	BatchID                  uint64 `json:"batch_id"`
	Month                    string `json:"month"`
	EmployeeNo               string `json:"employee_no"`
	Name                     string `json:"name"`
	EmploymentType           string `json:"employment_type"`
	JoinedOn                 string `json:"joined_on"`
	RequiredAttendanceDays   int    `json:"required_attendance_days"`
	LeaveDays                string `json:"leave_days"`
	PublicRestDays           string `json:"public_rest_days"`
	PayBasis                 string `json:"pay_basis"`
	WorkHours                string `json:"work_hours"`
	EntrySalary              string `json:"entry_salary"`
	BaseSalary               string `json:"base_salary"`
	Bonus                    string `json:"bonus"`
	AttendanceDeduction      string `json:"attendance_deduction"`
	OtherDeduction           string `json:"other_deduction"`
	Deduction                string `json:"deduction"`
	SocialSecurity           string `json:"social_security"`
	Tax                      string `json:"tax"`
	NetSalary                string `json:"net_salary"`
	ManualNetSalary          string `json:"manual_net_salary"`
	Status                   string `json:"status"`
	StatusClass              string `json:"status_class"`
	BaseSalaryValue          string `json:"base_salary_value"`
	WorkHoursValue           string `json:"work_hours_value"`
	BonusValue               string `json:"bonus_value"`
	AttendanceDeductionValue string `json:"attendance_deduction_value"`
	OtherDeductionValue      string `json:"other_deduction_value"`
	SocialSecurityValue      string `json:"social_security_value"`
	TaxValue                 string `json:"tax_value"`
	ManualNetSalaryValue     string `json:"manual_net_salary_value"`
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
	Seq           int    `json:"seq,omitempty"`
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
	Seq           int    `json:"seq,omitempty"`
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

// MenuItem 是菜品菜单记录，保留来源工作表和行号便于核对原始文件。
type MenuItem struct {
	Seq           int    `json:"seq,omitempty"`
	ID            uint64 `json:"id"`
	SourceSheet   string `json:"source_sheet"`
	SourceRow     int    `json:"source_row"`
	SourceNo      string `json:"source_no"`
	Restaurant    string `json:"restaurant"`
	WindowName    string `json:"window_name"`
	Name          string `json:"name"`
	Specification string `json:"specification"`
	Price         string `json:"price"`
	PriceText     string `json:"price_text"`
	SalesPeriod   string `json:"sales_period"`
	PriceRange    string `json:"price_range"`
	PriceCategory string `json:"price_category"`
	ItemCategory  string `json:"item_category"`
	Remark        string `json:"remark"`
}

type MenuPage struct {
	Items           []MenuItem      `json:"items"`
	Page            int             `json:"page"`
	PageSize        int             `json:"page_size"`
	Total           int             `json:"total"`
	TotalPages      int             `json:"total_pages"`
	Sort            string          `json:"sort"`
	PriceStats      []MenuPriceStat `json:"price_stats"`
	PriceCategories []string        `json:"price_categories"`
	ItemCategories  []string        `json:"item_categories"`
}

type MenuPriceStat struct {
	Price     string `json:"price"`
	PriceText string `json:"price_text"`
	Count     int    `json:"count"`
}

type Stall struct {
	Seq                     int    `json:"seq,omitempty"`
	ID                      uint64 `json:"id"`
	StallNo                 string `json:"stall_no"`
	Name                    string `json:"name"`
	ResponsibleName         string `json:"responsible_name"`
	ResponsibleMobile       string `json:"responsible_mobile"`
	MonthlyRent             string `json:"monthly_rent"`
	DefaultGuaranteedAmount string `json:"default_guaranteed_amount"`
	AllocationMode          string `json:"allocation_mode"`
	ProfitJiaRate           string `json:"profit_jia_rate"`
	ProfitYiRate            string `json:"profit_yi_rate"`
	GuaranteedRate          string `json:"guaranteed_rate"`
	BaseAmountRate          string `json:"base_amount_rate"`
	BaseAmountFactor        string `json:"base_amount_factor"`
	TotalAmountRate         string `json:"total_amount_rate"`
	Remark                  string `json:"remark"`
	Month                   string `json:"month"`
	SchoolCardAmount        string `json:"school_card_amount"`
	HuilaimiAmount          string `json:"huilaimi_amount"`
	CostAmount              string `json:"cost_amount"`
	GuaranteedAmount        string `json:"guaranteed_amount"`
	NetTurnover             string `json:"net_turnover"`
	TotalTurnover           string `json:"total_turnover"`
	ConradCommission        string `json:"conrad_commission"`
	JiaCommission           string `json:"jia_commission"`
	YiCommission            string `json:"yi_commission"`
	OwnCommission           string `json:"own_commission"`
	ManualOwnCommission     string `json:"manual_own_commission"`
	ManualOwnCommissionSet  bool   `json:"manual_own_commission_set"`
	CanManualOwnCommission  bool   `json:"can_manual_own_commission"`
	RentCommission          string `json:"rent_commission"`
	GuaranteedCommission    string `json:"guaranteed_commission"`
	BaseCommission          string `json:"base_commission"`
	TotalRateCommission     string `json:"total_rate_commission"`
	TotalCommission         string `json:"total_commission"`
	Remaining               string `json:"remaining"`
	TurnoverID              uint64 `json:"turnover_id,omitempty"`
}

type StallSummary struct {
	StallCount        int    `json:"stall_count"`
	TotalCardTurnover string `json:"total_card_turnover"`
	TotalHuilaimi     string `json:"total_huilaimi"`
	TotalTurnover     string `json:"total_turnover"`
	TotalCost         string `json:"total_cost"`
	ConradCommission  string `json:"conrad_commission"`
	JiaCommission     string `json:"jia_commission"`
	YiCommission      string `json:"yi_commission"`
	OwnCommission     string `json:"own_commission"`
	TotalCommission   string `json:"total_commission"`
	Remaining         string `json:"remaining"`
}

type StallSettings struct {
	Month       string `json:"month"`
	CanteenName string `json:"canteen_name"`
	ConradRate  string `json:"conrad_rate"`
	Remark      string `json:"remark"`
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
	Period         string `json:"period"`
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
	Seq         int    `json:"seq,omitempty"`
	ID          uint64 `json:"id"`
	Time        string `json:"time"`
	Operator    string `json:"operator"`
	Module      string `json:"module"`
	Action      string `json:"action"`
	Target      string `json:"target"`
	Detail      string `json:"detail"`
	IP          string `json:"ip"`
	Result      string `json:"result"`
	ResultClass string `json:"result_class"`
}
