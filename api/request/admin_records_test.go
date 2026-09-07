package request

import "testing"

func TestLedgerInputValidate(t *testing.T) {
	v := LedgerInput{OccurredOn: "2026-09-07", AccountID: 1, Direction: "expense", Amount: "12.34", Summary: "日常采购"}
	if err := v.Validate(); err != nil { t.Fatalf("合法台账被拒绝：%v", err) }
	v.Amount = "0"
	if err := v.Validate(); err == nil { t.Fatal("零金额应被拒绝") }
}

func TestEmployeeInputRejectsMaskedDataAndInvalidDates(t *testing.T) {
	v := EmployeeInput{EmployeeNo: "E001", Name: "小欣", Gender: "female", Mobile: "138****0000", EmploymentStatus: "active", EmploymentType: "full_time", JoinedOn: "2026-09-07", CurrentSalary: "5000.00"}
	if err := v.Validate(); err == nil { t.Fatal("不应允许保存脱敏手机号") }
	v.Mobile, v.RegularizedOn = "13800000000", "2026-09-01"
	if err := v.Validate(); err == nil { t.Fatal("转正日期早于入职日期时应报错") }
}

func TestAttendanceInputRequiresMatchingDuration(t *testing.T) {
	v := AttendanceInput{EmployeeID: 1, Category: "leave", RecordType: "事假", OccurredOn: "2026-09-07", DurationDays: "1.00"}
	if err := v.Validate(); err != nil { t.Fatalf("合法请假记录被拒绝：%v", err) }
	v.Category, v.DurationMinutes = "exception", 0
	if err := v.Validate(); err == nil { t.Fatal("异常记录应填写分钟数") }
}
