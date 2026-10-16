package request

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type DepartmentInput struct {
	DepartmentNo      string `json:"department_no"`
	Name              string `json:"name"`
	ParentID          uint64 `json:"parent_id"`
	ManagerEmployeeID uint64 `json:"manager_employee_id"`
	SortOrder         int    `json:"sort_order"`
	Enabled           bool   `json:"enabled"`
	Remark            string `json:"remark"`
}

func (v *DepartmentInput) Validate() error {
	v.DepartmentNo, v.Name = strings.TrimSpace(v.DepartmentNo), strings.TrimSpace(v.Name)
	if v.DepartmentNo == "" || !textLength(v.DepartmentNo, 32) || v.Name == "" || !textLength(v.Name, 64) {
		return fmt.Errorf("部门编号和名称必填，最多分别32字和64字")
	}
	if !textLength(v.Remark, 255) {
		return fmt.Errorf("备注不能超过255字")
	}
	return nil
}

type PositionInput struct {
	PositionNo   string `json:"position_no"`
	Name         string `json:"name"`
	DepartmentID uint64 `json:"department_id"`
	LevelName    string `json:"level_name"`
	MinSalary    string `json:"min_salary"`
	MaxSalary    string `json:"max_salary"`
	SortOrder    int    `json:"sort_order"`
	Enabled      bool   `json:"enabled"`
	Remark       string `json:"remark"`
}

func (v *PositionInput) Validate() error {
	v.PositionNo, v.Name = strings.TrimSpace(v.PositionNo), strings.TrimSpace(v.Name)
	if v.PositionNo == "" || !textLength(v.PositionNo, 32) || v.Name == "" || !textLength(v.Name, 64) {
		return fmt.Errorf("岗位编号和名称必填，最多分别32字和64字")
	}
	if v.DepartmentID == 0 {
		return fmt.Errorf("请选择所属部门")
	}
	if !textLength(v.LevelName, 32) || !textLength(v.Remark, 255) {
		return fmt.Errorf("职级或备注内容过长")
	}
	for _, amount := range []string{v.MinSalary, v.MaxSalary} {
		if amount != "" {
			if err := ValidateMoney(amount, false, 10); err != nil {
				return err
			}
		}
	}
	if v.MinSalary != "" && v.MaxSalary != "" {
		min, _ := strconv.ParseFloat(v.MinSalary, 64)
		max, _ := strconv.ParseFloat(v.MaxSalary, 64)
		if min > max {
			return fmt.Errorf("薪资下限不能大于上限")
		}
	}
	return nil
}

type AttendanceInput struct {
	EmployeeID uint64 `json:"employee_id"`
	Category   string `json:"category"`
	RecordType string `json:"record_type"`
	OccurredOn string `json:"occurred_on"`
	StartAt    string `json:"start_at"`
	EndAt      string `json:"end_at"`
	// 兼容旧版客户端的时间字段，新的后台使用 start_at/end_at。
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	DurationMinutes int    `json:"duration_minutes"`
	DurationDays    string `json:"duration_days"`
	SalaryEffect    string `json:"salary_effect"`
	SubsidyAmount   string `json:"subsidy_amount"`
	Reason          string `json:"reason"`
}

func validClock(v string) bool {
	if v == "" {
		return true
	}
	_, err := time.Parse("15:04", v)
	return err == nil
}
func validDateTime(v string) bool {
	if v == "" {
		return true
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if _, err := time.Parse(layout, v); err == nil {
			return true
		}
	}
	return false
}
func parseDateTime(v string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, v); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
func (v *AttendanceInput) Validate() error {
	v.RecordType, v.Reason = strings.TrimSpace(v.RecordType), strings.TrimSpace(v.Reason)
	v.SalaryEffect = strings.TrimSpace(v.SalaryEffect)
	if v.Category == "public_rest" {
		v.Category = "leave"
		if v.RecordType == "" {
			v.RecordType = "公休"
		}
	}
	if v.Category == "holiday" {
		v.Category = "leave"
		v.RecordType = "节假日"
		v.SalaryEffect = "none"
		v.SubsidyAmount = "0"
	}
	// 正常公休不扣工资；其余请假记录默认按扣款处理。
	if v.RecordType == "公休" || v.RecordType == "节假日" {
		if v.Category != "leave" {
			return fmt.Errorf("公休或节假日必须选择请假记录类别")
		}
		v.SalaryEffect = "none"
		v.SubsidyAmount = "0"
	}
	// 特殊请假属于请假记录；兼容前端误选“历史异常”的旧请求。
	if v.RecordType == "特殊请假" && v.Category == "exception" {
		v.Category = "leave"
	}
	if v.SalaryEffect == "" {
		if v.Category == "leave" {
			v.SalaryEffect = "deduct"
		} else {
			v.SalaryEffect = "none"
		}
	}
	if v.EmployeeID == 0 {
		return fmt.Errorf("请选择员工")
	}
	if v.Category != "leave" && v.Category != "exception" {
		return fmt.Errorf("请选择请假或异常")
	}
	if v.RecordType == "" || !textLength(v.RecordType, 32) {
		return fmt.Errorf("记录类型必填且不能超过32字")
	}
	if v.RecordType == "特殊请假" && v.Category != "leave" {
		return fmt.Errorf("特殊请假必须选择请假记录类别")
	}
	if !validDate(v.OccurredOn, false) || !validClock(v.StartTime) || !validClock(v.EndTime) || !validDateTime(v.StartAt) || !validDateTime(v.EndAt) {
		return fmt.Errorf("日期或时间格式不正确")
	}
	if v.StartAt != "" && v.EndAt != "" {
		start, _ := parseDateTime(v.StartAt)
		end, _ := parseDateTime(v.EndAt)
		if end.Before(start) {
			return fmt.Errorf("结束时间不能早于开始时间")
		}
	}
	if v.Category == "leave" {
		if err := ValidateMoney(v.DurationDays, true, 4); err != nil {
			return fmt.Errorf("请假天数必须大于0，最多2位小数")
		}
		if days, _ := strconv.ParseFloat(v.DurationDays, 64); math.Abs(days*2-math.Round(days*2)) > 1e-9 {
			return fmt.Errorf("请假或公休天数最小单位为0.5天")
		}
	}
	if v.SalaryEffect != "none" && v.SalaryEffect != "deduct" && v.SalaryEffect != "subsidy" && v.SalaryEffect != "deduct_and_subsidy" {
		return fmt.Errorf("工资影响方式不正确")
	}
	if v.Category == "exception" && v.DurationMinutes <= 0 && (v.DurationDays == "" || v.SalaryEffect != "deduct_and_subsidy") {
		return fmt.Errorf("异常记录请填写分钟数或天数")
	}
	if v.Category == "exception" && v.DurationDays != "" {
		if err := ValidateMoney(v.DurationDays, true, 4); err != nil {
			return fmt.Errorf("异常天数必须大于0，最多2位小数")
		}
		if days, _ := strconv.ParseFloat(v.DurationDays, 64); math.Abs(days*2-math.Round(days*2)) > 1e-9 {
			return fmt.Errorf("历史异常天数最小单位为0.5天")
		}
	}
	if v.SalaryEffect == "deduct" && strings.TrimSpace(v.SubsidyAmount) != "" && strings.TrimSpace(v.SubsidyAmount) != "0" && strings.TrimSpace(v.SubsidyAmount) != "0.00" {
		v.SalaryEffect = "deduct_and_subsidy"
	}
	if v.SalaryEffect == "subsidy" || v.SalaryEffect == "deduct_and_subsidy" {
		if v.SubsidyAmount == "" {
			return fmt.Errorf("补助工资影响方式必须填写补助金额")
		}
		if err := ValidateMoney(v.SubsidyAmount, false, 10); err != nil {
			return fmt.Errorf("补助金额格式不正确")
		}
	} else {
		v.SubsidyAmount = "0"
	}
	if !textLength(v.Reason, 500) {
		return fmt.Errorf("原因不能超过500字")
	}
	return nil
}

type ChangeInput struct {
	EmployeeID        uint64 `json:"employee_id"`
	ChangeType        string `json:"change_type"`
	AfterDepartmentID uint64 `json:"after_department_id"`
	AfterPositionID   uint64 `json:"after_position_id"`
	AfterStatus       string `json:"after_status"`
	EffectiveOn       string `json:"effective_on"`
	Reason            string `json:"reason"`
}

func (v *ChangeInput) Validate() error {
	if v.EmployeeID == 0 {
		return fmt.Errorf("请选择员工")
	}
	if !map[string]bool{"hire": true, "regularize": true, "transfer": true, "promotion": true, "demotion": true, "leave": true}[v.ChangeType] {
		return fmt.Errorf("异动类型不正确")
	}
	// 特定异动的员工状态由服务端统一确定，避免前端传入矛盾数据。
	if v.ChangeType == "leave" {
		v.AfterStatus = "left"
	}
	if v.ChangeType == "regularize" {
		v.AfterStatus = "active"
	}
	if !map[string]bool{"probation": true, "active": true, "left": true}[v.AfterStatus] {
		return fmt.Errorf("异动后状态不正确")
	}
	if !validDate(v.EffectiveOn, false) {
		return fmt.Errorf("生效日期不正确")
	}
	if v.EffectiveOn > time.Now().Format("2006-01-02") {
		return fmt.Errorf("人事异动会立即更新员工档案，生效日期不能晚于今天")
	}
	if !textLength(v.Reason, 500) {
		return fmt.Errorf("原因不能超过500字")
	}
	return nil
}

type SalaryAdjustmentInput struct {
	EmployeeID  uint64 `json:"employee_id"`
	AfterSalary string `json:"after_salary"`
	EffectiveOn string `json:"effective_on"`
	Reason      string `json:"reason"`
}

func (v *SalaryAdjustmentInput) Validate() error {
	if v.EmployeeID == 0 {
		return fmt.Errorf("请选择员工")
	}
	if err := ValidateMoney(v.AfterSalary, false, 10); err != nil {
		return err
	}
	if !validDate(v.EffectiveOn, false) {
		return fmt.Errorf("生效日期不正确")
	}
	if !textLength(v.Reason, 500) {
		return fmt.Errorf("原因不能超过500字")
	}
	return nil
}
