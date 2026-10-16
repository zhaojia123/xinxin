package request

import (
	"fmt"
	"strconv"
	"strings"
)

// WorkLogInput 是临时工每日工时记录，按小时、按天、按半天均以实际小时统一登记。
type WorkLogInput struct {
	EmployeeID uint64 `json:"employee_id"`
	WorkDate   string `json:"work_date"`
	Hours      string `json:"hours"`
	Remark     string `json:"remark"`
}

func (v *WorkLogInput) Validate() error {
	v.WorkDate, v.Hours, v.Remark = strings.TrimSpace(v.WorkDate), strings.TrimSpace(v.Hours), strings.TrimSpace(v.Remark)
	if v.EmployeeID == 0 {
		return fmt.Errorf("请选择临时工")
	}
	if !validDate(v.WorkDate, false) {
		return fmt.Errorf("工作日期格式不正确")
	}
	hours, err := strconv.ParseFloat(v.Hours, 64)
	if err != nil || hours <= 0 || hours > 24 {
		return fmt.Errorf("当天工时必须大于0且不超过24小时")
	}
	if !textLength(v.Remark, 255) {
		return fmt.Errorf("备注不能超过255字")
	}
	return nil
}
