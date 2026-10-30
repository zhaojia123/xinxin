package request

import "testing"

func TestStallInputUsesFixedProfitShareRates(t *testing.T) {
	v := StallInput{
		StallNo:          "S01",
		Name:             "益禾堂",
		AllocationMode:   "profit_share",
		ProfitJiaRate:    "1",
		ProfitYiRate:     "99",
		MonthlyRent:      "0",
		GuaranteedRate:   "0",
		BaseAmountRate:   "0",
		BaseAmountFactor: "0",
		TotalAmountRate:  "0",
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("固定分成档口校验失败：%v", err)
	}
	if v.ProfitJiaRate != "0.90000000" || v.ProfitYiRate != "0.10000000" {
		t.Fatalf("甲乙分成必须固定为90%%和10%%，实际为甲=%s、乙=%s", v.ProfitJiaRate, v.ProfitYiRate)
	}
}

func TestStallMonthlyRuleUsesFixedProfitShareRates(t *testing.T) {
	v := StallMonthlyRuleInput{
		StallID:          1,
		Month:            "2026-09",
		AllocationMode:   "profit_share",
		ProfitJiaRate:    "50",
		ProfitYiRate:     "50",
		MonthlyRent:      "0",
		GuaranteedRate:   "0",
		BaseAmountRate:   "0",
		BaseAmountFactor: "0",
		TotalAmountRate:  "0",
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("固定分成当月规则校验失败：%v", err)
	}
	if v.ProfitJiaRate != "0.90000000" || v.ProfitYiRate != "0.10000000" {
		t.Fatalf("甲乙当月分成必须固定为90%%和10%%，实际为甲=%s、乙=%s", v.ProfitJiaRate, v.ProfitYiRate)
	}
}

func TestStallMonthlyRuleEmptyAmountsBecomeZero(t *testing.T) {
	v := StallMonthlyRuleInput{
		StallID:        1,
		Month:          "2026-09",
		AllocationMode: "custom",
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("空金额的月度规则应可保存：%v", err)
	}
	if v.MonthlyRent != "0" || v.FixedCommissionAmount != "0" || v.DefaultGuaranteedAmount != "0" {
		t.Fatalf("空金额应按0保存，实际月租=%q、固定抽成=%q、保底=%q", v.MonthlyRent, v.FixedCommissionAmount, v.DefaultGuaranteedAmount)
	}
}

func TestStallMonthlyExpenseInputValidation(t *testing.T) {
	v := StallMonthlyExpenseInput{Month: "2026-09", WaterAmount: "100", ElectricityAmount: "88.50"}
	if err := v.Validate(); err != nil {
		t.Fatalf("经营支出应允许工资留空并自动汇总：%v", err)
	}
	v.WaterAmount = "-1"
	if err := v.Validate(); err == nil {
		t.Fatal("水费不能为负数")
	}
}

func TestStallMonthlyExpenseEmptyWaterAndElectricityBecomeZero(t *testing.T) {
	v := StallMonthlyExpenseInput{Month: "2026-09"}
	if err := v.Validate(); err != nil {
		t.Fatalf("空水电应按0保存：%v", err)
	}
	if v.WaterAmount != "0" || v.ElectricityAmount != "0" {
		t.Fatalf("空水电应规范为0，实际水费=%q，电费=%q", v.WaterAmount, v.ElectricityAmount)
	}
}

func TestStallMonthlyExpenseLedgerUsesSelectedMonth(t *testing.T) {
	v := StallMonthlyExpenseLedgerInput{Month: "2026-09"}
	if err := v.Validate(); err != nil {
		t.Fatalf("未填写日期时应默认记入核算月份：%v", err)
	}
	if v.OccurredOn != "2026-09-01" {
		t.Fatalf("默认入账日期应为核算月首日，实际=%s", v.OccurredOn)
	}
	v.OccurredOn = "2026-10-01"
	if err := v.Validate(); err == nil {
		t.Fatal("入账日期不属于核算月份时应拒绝保存")
	}
}

func TestStallSettingsAllowsMonthlyConradRate(t *testing.T) {
	v := StallSettingsInput{Month: "2026-09", ConradRate: "3.5"}
	if err := v.Validate(); err != nil {
		t.Fatalf("康莱德抽点比例应允许保存：%v", err)
	}
	if v.ConradRate != "0.03500000" {
		t.Fatalf("康莱德比例应标准化为小数，实际：%s", v.ConradRate)
	}
}
