package request

import (
	"testing"
	"time"
)

func TestLedgerInputValidate(t *testing.T) {
	v := LedgerInput{OccurredOn: "2026-09-07", AccountID: 1, Direction: "expense", Amount: "12.34", Summary: "日常采购"}
	if err := v.Validate(); err != nil {
		t.Fatalf("合法台账被拒绝：%v", err)
	}
	v.Amount = "0"
	if err := v.Validate(); err == nil {
		t.Fatal("零金额应被拒绝")
	}
}

func TestEmployeeInputRejectsMaskedDataAndInvalidDates(t *testing.T) {
	v := EmployeeInput{EmployeeNo: "E001", Name: "小欣", Gender: "female", Mobile: "138****0000", EmploymentStatus: "active", EmploymentType: "full_time", JoinedOn: "2026-09-07", CurrentSalary: "5000.00"}
	if err := v.Validate(); err == nil {
		t.Fatal("不应允许保存脱敏手机号")
	}
	v.Mobile, v.RegularizedOn = "13800000000", "2026-09-01"
	if err := v.Validate(); err == nil {
		t.Fatal("转正日期早于入职日期时应报错")
	}
}

func TestAttendanceInputRequiresMatchingDuration(t *testing.T) {
	v := AttendanceInput{EmployeeID: 1, Category: "leave", RecordType: "事假", OccurredOn: "2026-09-07", DurationDays: "1.00"}
	if err := v.Validate(); err != nil {
		t.Fatalf("合法请假记录被拒绝：%v", err)
	}
	v.Category, v.DurationMinutes = "exception", 0
	if err := v.Validate(); err == nil {
		t.Fatal("异常记录应填写分钟数")
	}
}

func TestChangeInputNormalizesStatusAndRejectsFutureDate(t *testing.T) {
	v := ChangeInput{EmployeeID: 1, ChangeType: "leave", AfterStatus: "active", EffectiveOn: time.Now().Format("2006-01-02")}
	if err := v.Validate(); err != nil {
		t.Fatalf("合法离职异动被拒绝：%v", err)
	}
	if v.AfterStatus != "left" {
		t.Fatalf("离职异动状态应由服务端改为 left，实际为 %s", v.AfterStatus)
	}
	v.EffectiveOn = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	if err := v.Validate(); err == nil {
		t.Fatal("立即更新员工档案的异动不应允许未来日期")
	}
}

func TestOptionInputRequiresCategoryDirectionAndPositionDepartment(t *testing.T) {
	category := OptionInput{Kind: "categories", Name: "日常采购"}
	if err := category.Validate(); err == nil {
		t.Fatal("收支分类必须指定收入或支出方向")
	}
	position := OptionInput{Kind: "positions", Name: "店员"}
	if err := position.Validate(); err == nil {
		t.Fatal("岗位必须指定所属部门")
	}
}
