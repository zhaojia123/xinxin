package request

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// 金额使用十进制字符串传输，避免浮点运算改变分位。
var moneyPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,11})(\.[0-9]{1,2})?$`)

func ValidateMoney(value string, positive bool, integerDigits int) error {
	if !moneyPattern.MatchString(value) || len(strings.Split(value, ".")[0]) > integerDigits {
		return fmt.Errorf("金额格式不正确，最多填写%d位整数和2位小数", integerDigits)
	}
	if positive && strings.Trim(value, "0.") == "" {
		return fmt.Errorf("金额必须大于0")
	}
	return nil
}

func validDate(value string, optional bool) bool {
	if value == "" {
		return optional
	}
	d, err := time.Parse("2006-01-02", value)
	return err == nil && d.Year() >= 1000 && d.Year() <= 9999
}

func textLength(value string, max int) bool { return utf8.RuneCountInString(value) <= max }

type OptionInput struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Direction    string `json:"direction"`
	DepartmentID uint64 `json:"department_id"`
}

func (v *OptionInput) Validate() error {
	v.Name = strings.TrimSpace(v.Name)
	if !map[string]bool{"accounts": true, "categories": true, "departments": true, "positions": true}[v.Kind] {
		return fmt.Errorf("不支持的选项类型")
	}
	if v.Name == "" || !textLength(v.Name, 64) {
		return fmt.Errorf("名称必填且不能超过64字")
	}
	if v.Kind == "categories" && v.Direction != "income" && v.Direction != "expense" {
		return fmt.Errorf("请选择分类的收支方向")
	}
	if v.Kind == "positions" && v.DepartmentID == 0 {
		return fmt.Errorf("请先选择岗位所属部门")
	}
	return nil
}

type LedgerInput struct {
	OccurredOn   string `json:"occurred_on"`
	AccountID    uint64 `json:"account_id"`
	CategoryID   uint64 `json:"category_id"`
	DepartmentID uint64 `json:"department_id"`
	Direction    string `json:"direction"`
	Amount       string `json:"amount"`
	Summary      string `json:"summary"`
	Counterparty string `json:"counterparty"`
	VoucherNo    string `json:"voucher_no"`
	Remark       string `json:"remark"`
}

func (v *LedgerInput) Validate() error {
	v.Summary = strings.TrimSpace(v.Summary)
	if !validDate(v.OccurredOn, false) {
		return fmt.Errorf("请选择正确的记账日期")
	}
	if v.AccountID == 0 {
		return fmt.Errorf("请选择资金账户")
	}
	if v.Direction != "income" && v.Direction != "expense" {
		return fmt.Errorf("请选择收入或支出")
	}
	if err := ValidateMoney(v.Amount, true, 12); err != nil {
		return err
	}
	if v.Summary == "" || !textLength(v.Summary, 255) {
		return fmt.Errorf("摘要必填，最多255字")
	}
	if !textLength(v.Counterparty, 128) || !textLength(v.VoucherNo, 64) || !textLength(v.Remark, 500) {
		return fmt.Errorf("对方名称、凭证号或备注过长")
	}
	return nil
}

type EmployeeInput struct {
	EmployeeNo       string `json:"employee_no"`
	Name             string `json:"name"`
	Gender           string `json:"gender"`
	IDCard           string `json:"id_card"`
	Mobile           string `json:"mobile"`
	DepartmentID     uint64 `json:"department_id"`
	PositionID       uint64 `json:"position_id"`
	EmploymentStatus string `json:"employment_status"`
	EmploymentType   string `json:"employment_type"`
	JoinedOn         string `json:"joined_on"`
	RegularizedOn    string `json:"regularized_on"`
	LeftOn           string `json:"left_on"`
	CurrentSalary    string `json:"current_salary"`
	Education        string `json:"education"`
	Hometown         string `json:"hometown"`
	Remark           string `json:"remark"`
}

func (v *EmployeeInput) Validate() error {
	v.Name, v.EmployeeNo = strings.TrimSpace(v.Name), strings.TrimSpace(v.EmployeeNo)
	if v.Name == "" || !textLength(v.Name, 64) || v.EmployeeNo == "" || !textLength(v.EmployeeNo, 32) {
		return fmt.Errorf("姓名和员工编号必填，最多分别64字和32字")
	}
	if v.Gender != "unknown" && v.Gender != "male" && v.Gender != "female" {
		return fmt.Errorf("性别不正确")
	}
	if v.EmploymentStatus != "active" && v.EmploymentStatus != "probation" && v.EmploymentStatus != "left" {
		return fmt.Errorf("在职状态不正确")
	}
	if v.EmploymentType != "full_time" && v.EmploymentType != "part_time" && v.EmploymentType != "intern" {
		return fmt.Errorf("用工类型不正确")
	}
	for _, date := range []string{v.JoinedOn, v.RegularizedOn, v.LeftOn} {
		if !validDate(date, true) {
			return fmt.Errorf("日期格式不正确")
		}
	}
	if v.JoinedOn != "" && ((v.RegularizedOn != "" && v.RegularizedOn < v.JoinedOn) || (v.LeftOn != "" && v.LeftOn < v.JoinedOn)) {
		return fmt.Errorf("转正或离职日期不能早于入职日期")
	}
	if v.EmploymentStatus == "left" && v.LeftOn == "" {
		return fmt.Errorf("离职员工请填写离职日期")
	}
	if v.EmploymentStatus != "left" && v.LeftOn != "" {
		return fmt.Errorf("非离职员工请清空离职日期")
	}
	if strings.Contains(v.Mobile, "*") || strings.Contains(v.IDCard, "*") {
		return fmt.Errorf("不能将脱敏内容保存为手机号或身份证号")
	}
	if !textLength(v.Mobile, 32) || !textLength(v.IDCard, 32) || !textLength(v.Education, 32) || !textLength(v.Hometown, 128) || !textLength(v.Remark, 500) {
		return fmt.Errorf("员工资料内容过长")
	}
	return ValidateMoney(v.CurrentSalary, false, 10)
}
