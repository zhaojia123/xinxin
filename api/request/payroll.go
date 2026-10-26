package request

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type PayrollUpdate struct {
	WorkHours           float64 `json:"work_hours"`
	BaseSalary          float64 `json:"base_salary"`
	Bonus               float64 `json:"bonus"`
	AttendanceDeduction float64 `json:"attendance_deduction"`
	OtherDeduction      float64 `json:"other_deduction"`
	SocialSecurity      float64 `json:"social_security"`
	Tax                 float64 `json:"tax"`
	// ManualNetSalary 非空时跳过算法，直接作为本次实发工资。
	ManualNetSalary *float64 `json:"manual_net_salary"`
}

func (v PayrollUpdate) ValidManualNetSalary() bool {
	return v.ManualNetSalary == nil || (*v.ManualNetSalary >= 0 && !math.IsNaN(*v.ManualNetSalary) && !math.IsInf(*v.ManualNetSalary, 0))
}

type PayrollPay struct {
	BatchID        uint64            `json:"batch_id"`
	ItemID         uint64            `json:"item_id"`
	ItemIDs        []uint64          `json:"item_ids"`
	PaymentRemark  string            `json:"payment_remark"`
	PaymentRemarks map[uint64]string `json:"payment_remarks"`
	AccountID      uint64            `json:"account_id"`
	OccurredOn     string            `json:"occurred_on"`
}

// PayrollSettlementInput 按员工类型确认并汇总发放当月工资。
type PayrollSettlementInput struct {
	BatchID     uint64 `json:"batch_id"`
	PayrollType string `json:"payroll_type"`
	OccurredOn  string `json:"occurred_on"`
}

func (v *PayrollSettlementInput) Validate() error {
	if v.BatchID == 0 {
		return fmt.Errorf("工资批次ID不正确")
	}
	if v.PayrollType != "full_time" && v.PayrollType != "part_time" {
		return fmt.Errorf("工资类型不正确")
	}
	if _, err := time.Parse("2006-01-02", v.OccurredOn); err != nil {
		return fmt.Errorf("发放日期格式必须是 YYYY-MM-DD")
	}
	return nil
}

// SpecialPayrollInput 用于录入尚未建立员工档案的临时结算工资。
type SpecialPayrollInput struct {
	Month  string `json:"month"`
	Name   string `json:"name"`
	Amount string `json:"amount"`
	Remark string `json:"remark"`
}

func (v *SpecialPayrollInput) Validate() error {
	v.Month = strings.TrimSpace(v.Month)
	v.Name = strings.TrimSpace(v.Name)
	v.Remark = strings.TrimSpace(v.Remark)
	if _, err := time.Parse("2006-01", v.Month); err != nil {
		return fmt.Errorf("工资月份格式必须是 YYYY-MM")
	}
	if v.Name == "" || len([]rune(v.Name)) > 64 {
		return fmt.Errorf("特殊员工姓名必填且不能超过64字")
	}
	if err := ValidateMoney(v.Amount, true, 12); err != nil {
		return err
	}
	if len([]rune(v.Remark)) > 500 {
		return fmt.Errorf("备注不能超过500字")
	}
	return nil
}

type SpecialPayrollSettlementInput struct {
	Month      string `json:"month"`
	OccurredOn string `json:"occurred_on"`
}

func (v *SpecialPayrollSettlementInput) Validate() error {
	v.Month = strings.TrimSpace(v.Month)
	if _, err := time.Parse("2006-01", v.Month); err != nil {
		return fmt.Errorf("工资月份格式必须是 YYYY-MM")
	}
	if _, err := time.Parse("2006-01-02", v.OccurredOn); err != nil {
		return fmt.Errorf("发放日期格式必须是 YYYY-MM-DD")
	}
	return nil
}

// TemporaryPayrollInput 用于临时工直接发放，不创建员工档案。
type TemporaryPayrollInput struct {
	Name       string  `json:"name"`
	Amount     float64 `json:"amount"`
	AccountID  uint64  `json:"account_id"`
	OccurredOn string  `json:"occurred_on"`
	Remark     string  `json:"remark"`
}

func (v *TemporaryPayrollInput) Validate() error {
	v.Name = strings.TrimSpace(v.Name)
	v.Remark = strings.TrimSpace(v.Remark)
	if v.Name == "" || len([]rune(v.Name)) > 64 {
		return fmt.Errorf("临时工姓名必填且不能超过64字")
	}
	if v.Amount <= 0 || math.IsNaN(v.Amount) || math.IsInf(v.Amount, 0) {
		return fmt.Errorf("临时工工资必须大于0")
	}
	if v.AccountID == 0 {
		return fmt.Errorf("请选择付款账户")
	}
	if _, err := time.Parse("2006-01-02", v.OccurredOn); err != nil {
		return fmt.Errorf("发放日期格式必须是 YYYY-MM-DD")
	}
	if len([]rune(v.Remark)) > 255 {
		return fmt.Errorf("备注不能超过255字")
	}
	return nil
}
