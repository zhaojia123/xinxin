package request

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// WorkLogInput 是临时工每日工时记录，按小时、按天、按半天均以实际小时统一登记。
type WorkLogInput struct {
	EmployeeID uint64   `json:"employee_id"`
	WorkDate   string   `json:"work_date"`
	WorkDates  []string `json:"work_dates"`
	Hours      string   `json:"hours"`
	Remark     string   `json:"remark"`
}

// Dates 返回去重排序后的工作日期；编辑单条记录时继续兼容 work_date。
func (v *WorkLogInput) Dates() ([]string, error) {
	values := append([]string{}, v.WorkDates...)
	if len(values) == 0 && strings.TrimSpace(v.WorkDate) != "" {
		values = append(values, v.WorkDate)
	}
	seen := make(map[string]struct{}, len(values))
	dates := make([]string, 0, len(values))
	for _, value := range values {
		date := strings.TrimSpace(value)
		if !validDate(date, false) {
			return nil, fmt.Errorf("工作日期格式不正确")
		}
		if _, exists := seen[date]; !exists {
			seen[date] = struct{}{}
			dates = append(dates, date)
		}
	}
	if len(dates) == 0 {
		return nil, fmt.Errorf("请至少选择一个工作日期")
	}
	sort.Strings(dates)
	return dates, nil
}

func (v *WorkLogInput) Validate() error {
	v.WorkDate, v.Hours, v.Remark = strings.TrimSpace(v.WorkDate), strings.TrimSpace(v.Hours), strings.TrimSpace(v.Remark)
	if v.EmployeeID == 0 {
		return fmt.Errorf("请选择临时工")
	}
	dates, err := v.Dates()
	if err != nil {
		return err
	}
	v.WorkDates, v.WorkDate = dates, dates[0]
	hours, err := strconv.ParseFloat(v.Hours, 64)
	if err != nil || hours <= 0 || hours > 24 {
		return fmt.Errorf("当天工时必须大于0且不超过24小时")
	}
	if !textLength(v.Remark, 255) {
		return fmt.Errorf("备注不能超过255字")
	}
	return nil
}
