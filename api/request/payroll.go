package request

type PayrollUpdate struct {
	BaseSalary          float64 `json:"base_salary"`
	Bonus               float64 `json:"bonus"`
	AttendanceDeduction float64 `json:"attendance_deduction"`
	OtherDeduction      float64 `json:"other_deduction"`
	SocialSecurity      float64 `json:"social_security"`
	Tax                 float64 `json:"tax"`
}

type PayrollPay struct {
	BatchID    uint64 `json:"batch_id"`
	AccountID  uint64 `json:"account_id"`
	OccurredOn string `json:"occurred_on"`
}
