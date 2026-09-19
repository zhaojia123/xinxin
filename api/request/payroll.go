package request

import "math"

type PayrollUpdate struct {
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
	BatchID    uint64 `json:"batch_id"`
	ItemID     uint64 `json:"item_id"`
	AccountID  uint64 `json:"account_id"`
	OccurredOn string `json:"occurred_on"`
}
