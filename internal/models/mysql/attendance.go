package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"friends-records/api/response"
	"friends-records/internal/apperror"
)

type AttendanceCalendarRow struct {
	EmployeeNo string
	Name       string
	Marks      []string
}

// AttendanceCalendar 返回指定月份的全员日历；无记录日期默认标记为正常出勤。
func (s *Store) AttendanceCalendar(ctx context.Context, month string) ([]AttendanceCalendarRow, int, error) {
	if err := s.ready(); err != nil {
		return nil, 0, err
	}
	start, err := time.Parse("2006-01-02", month+"-01")
	if err != nil {
		return nil, 0, apperror.Wrap(err, "出勤月份格式不正确")
	}
	days := start.AddDate(0, 1, -1).Day()
	rows, err := s.DB.QueryContext(ctx, `SELECT e.employee_no,e.name,DAY(a.occurred_on),a.category,a.record_type,a.salary_effect,COALESCE(a.subsidy_amount,0) FROM employees e LEFT JOIN attendance_records a ON a.employee_id=e.id AND a.occurred_on>=? AND a.occurred_on<? AND a.active=1 AND a.status IN ('pending','approved','recorded','confirmed') WHERE e.active=1 ORDER BY e.employee_no,a.occurred_on,a.id`, start, start.AddDate(0, 1, 0))
	if err != nil {
		return nil, 0, apperror.Wrap(err, "查询全员出勤日历失败")
	}
	defer rows.Close()
	type markSet struct {
		no, name string
		marks    []string
	}
	ordered := make([]*markSet, 0)
	byNo := map[string]*markSet{}
	markPriority := map[string]int{"√": 0, "○": 1, "☆": 2, "✕": 3, "◎": 4}
	for rows.Next() {
		var no, name string
		var day sql.NullInt64
		var category, recordType, salaryEffect sql.NullString
		var subsidy sql.NullFloat64
		if err := rows.Scan(&no, &name, &day, &category, &recordType, &salaryEffect, &subsidy); err != nil {
			return nil, 0, err
		}
		item := byNo[no]
		if item == nil {
			item = &markSet{no: no, name: name, marks: make([]string, days)}
			for i := range item.marks {
				item.marks[i] = "√"
			}
			byNo[no] = item
			ordered = append(ordered, item)
		}
		if !day.Valid || day.Int64 < 1 || int(day.Int64) > days {
			continue
		}
		idx := int(day.Int64) - 1
		mark := ""
		switch {
		case category.String == "leave" && recordType.String == "特殊请假" && (salaryEffect.String == "subsidy" || salaryEffect.String == "deduct_and_subsidy" || subsidy.Float64 > 0):
			mark = "◎"
		case category.String == "leave" && recordType.String != "公休" && recordType.String != "节假日":
			mark = "✕"
		case recordType.String == "节假日":
			mark = "☆"
		case recordType.String == "公休":
			mark = "○"
		}
		if mark != "" {
			if markPriority[mark] > markPriority[item.marks[idx]] {
				item.marks[idx] = mark
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	result := make([]AttendanceCalendarRow, 0, len(ordered))
	for _, item := range ordered {
		result = append(result, AttendanceCalendarRow{EmployeeNo: item.no, Name: item.name, Marks: item.marks})
	}
	return result, days, nil
}

func attendanceStatus(value string) (string, string) {
	switch value {
	case "approved":
		return "已通过", "active"
	case "rejected":
		return "已驳回", "inactive"
	case "recorded":
		return "已记录", "warning"
	case "confirmed":
		return "已确认", "warning"
	case "cancelled":
		return "已取消", "inactive"
	case "pending":
		// 兼容历史数据：旧记录曾保存为待审批，现在按已通过处理。
		return "已通过", "active"
	default:
		return "已通过", "active"
	}
}
func sourceLabel(value string) string {
	switch value {
	case "employee":
		return "员工申请"
	case "mini_program":
		return "小程序"
	case "import":
		return "导入"
	default:
		return "手工登记"
	}
}

func (s *Store) AttendanceRecords(ctx context.Context) ([]response.AttendanceRecord, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT a.id,e.employee_no,e.name,a.category,a.record_type,DATE_FORMAT(a.occurred_on,'%Y-%m-%d'),CASE WHEN a.start_at IS NOT NULL THEN DATE_FORMAT(a.start_at,'%Y-%m-%d %H:%i') ELSE COALESCE(DATE_FORMAT(a.start_time,'%H:%i'),'') END,a.duration_minutes,a.duration_days,a.salary_effect,COALESCE(a.subsidy_amount,0),a.reason,a.status,a.source FROM attendance_records a JOIN employees e ON e.id=a.employee_id WHERE a.active=1 AND a.category<>'exception' ORDER BY a.occurred_on DESC,a.id DESC LIMIT 500`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询请假与公休记录失败")
	}
	defer rows.Close()
	result := make([]response.AttendanceRecord, 0)
	for rows.Next() {
		var item response.AttendanceRecord
		var category, status, source, effect string
		var minutes int
		var days, subsidy float64
		if err := rows.Scan(&item.ID, &item.EmployeeNo, &item.Name, &category, &item.Type, &item.Date, &item.Time, &minutes, &days, &effect, &subsidy, &item.Reason, &status, &source); err != nil {
			return nil, apperror.Wrap(err, "读取请假与公休记录失败")
		}
		if category == "leave" {
			if item.Type == "公休" {
				item.Category = "公休"
			} else if item.Type == "节假日" {
				item.Category = "节假日"
			} else if item.Type == "特殊请假" {
				item.Category = "特殊请假"
			} else {
				item.Category = "请假"
			}
			item.Duration = fmt.Sprintf("%g天", days)
		} else {
			item.Category = "异常"
			if days > 0 {
				item.Duration = fmt.Sprintf("%g天", days)
			} else {
				item.Duration = fmt.Sprintf("%d分钟", minutes)
			}
		}
		item.SalaryEffect = effect
		if subsidy > 0 {
			item.Subsidy = Money(subsidy)
		}
		item.Status, item.StatusClass = attendanceStatus(status)
		item.Source = sourceLabel(source)
		item.Seq = len(result) + 1
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历请假与公休记录失败")
	}
	return result, nil
}

type metricRow struct {
	no, name                                                     string
	leave                                                        float64
	leaveCount, lateMinutes, lateCount, earlyMinutes, earlyCount int
}

func periodRange(period string, now time.Time) (string, string, time.Time, time.Time) {
	y, m, _ := now.Date()
	loc := now.Location()
	switch period {
	case "quarter":
		startMonth := time.Month(((int(m)-1)/3)*3 + 1)
		start := time.Date(y, startMonth, 1, 0, 0, 0, 0, loc)
		return "quarter", fmt.Sprintf("%d年第%d季度", y, (int(m)-1)/3+1), start, start.AddDate(0, 3, 0)
	case "half":
		startMonth, label := time.January, fmt.Sprintf("%d年上半年", y)
		if m >= time.July {
			startMonth, label = time.July, fmt.Sprintf("%d年下半年", y)
		}
		start := time.Date(y, startMonth, 1, 0, 0, 0, 0, loc)
		return "half", label, start, start.AddDate(0, 6, 0)
	case "year":
		start := time.Date(y, time.January, 1, 0, 0, 0, 0, loc)
		return "year", fmt.Sprintf("%d年度", y), start, start.AddDate(1, 0, 0)
	default:
		start := time.Date(y, m, 1, 0, 0, 0, 0, loc)
		return "month", fmt.Sprintf("%d年%d月", y, m), start, start.AddDate(0, 1, 0)
	}
}

func (s *Store) AttendanceStatistics(ctx context.Context, period string) (response.AttendanceStatistics, error) {
	return s.AttendanceStatisticsAt(ctx, period, "")
}

// AttendanceStatisticsAt 按页面选择的月份计算统计，避免列表月份和顶部统计月份不一致。
func (s *Store) AttendanceStatisticsAt(ctx context.Context, period, month string) (response.AttendanceStatistics, error) {
	if err := s.ready(); err != nil {
		return response.AttendanceStatistics{}, err
	}
	base := time.Now()
	if parsed, err := time.Parse("2006-01", strings.TrimSpace(month)); err == nil {
		base = parsed
	}
	period, label, start, end := periodRange(period, base)
	rows, err := s.DB.QueryContext(ctx, `SELECT e.employee_no,e.name,COALESCE(SUM(CASE WHEN a.category='leave' AND a.record_type NOT IN ('公休','节假日') AND a.status IN ('pending','approved') THEN a.duration_days ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.category='leave' AND a.record_type NOT IN ('公休','节假日') AND a.status IN ('pending','approved') THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.record_type='迟到' AND a.status IN ('pending','recorded','confirmed','approved') THEN a.duration_minutes ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.record_type='迟到' AND a.status IN ('pending','recorded','confirmed','approved') THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.record_type='早退' AND a.status IN ('pending','recorded','confirmed','approved') THEN a.duration_minutes ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.record_type='早退' AND a.status IN ('pending','recorded','confirmed','approved') THEN 1 ELSE 0 END),0) FROM employees e LEFT JOIN attendance_records a ON a.employee_id=e.id AND a.occurred_on>=? AND a.occurred_on<? AND a.active=1 WHERE e.employment_status IN ('active','probation') AND e.active=1 GROUP BY e.id,e.employee_no,e.name ORDER BY e.employee_no`, start, end)
	if err != nil {
		return response.AttendanceStatistics{}, apperror.Wrap(err, "查询员工出勤统计失败")
	}
	defer rows.Close()
	all := make([]metricRow, 0)
	for rows.Next() {
		var v metricRow
		if err := rows.Scan(&v.no, &v.name, &v.leave, &v.leaveCount, &v.lateMinutes, &v.lateCount, &v.earlyMinutes, &v.earlyCount); err != nil {
			return response.AttendanceStatistics{}, apperror.Wrap(err, "读取员工出勤统计失败")
		}
		all = append(all, v)
	}
	if err := rows.Err(); err != nil {
		return response.AttendanceStatistics{}, apperror.Wrap(err, "遍历员工出勤统计失败")
	}
	result := response.AttendanceStatistics{Period: period, Label: label, Leave: metrics(all, "leave"), Late: metrics(all, "late"), Early: metrics(all, "early"), Best: bestEmployees(all, period)}
	// 当前出勤登记不再走审批流程，历史待审批记录也按已通过处理。
	result.PendingCount = 0
	return result, nil
}

func metrics(all []metricRow, kind string) []response.EmployeeMetric {
	type vr struct {
		row   metricRow
		value float64
		count int
	}
	values := make([]vr, 0)
	for _, row := range all {
		value, count := row.leave, row.leaveCount
		if kind == "late" {
			value, count = float64(row.lateMinutes), row.lateCount
		}
		if kind == "early" {
			value, count = float64(row.earlyMinutes), row.earlyCount
		}
		if value > 0 {
			values = append(values, vr{row, value, count})
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].value > values[j].value })
	max := 0.0
	if len(values) > 0 {
		max = values[0].value
	}
	result := make([]response.EmployeeMetric, 0, len(values))
	for _, v := range values {
		percent := 10
		if max > 0 {
			percent = int(math.Round(v.value / max * 100))
		}
		if percent < 10 {
			percent = 10
		}
		value := fmt.Sprintf("%g天", v.value)
		detail := fmt.Sprintf("请假 %d次", v.count)
		if kind != "leave" {
			value = fmt.Sprintf("%g分钟", v.value)
			detail = fmt.Sprintf("%d次", v.count)
		}
		result = append(result, response.EmployeeMetric{EmployeeNo: v.row.no, Name: v.row.name, Value: value, Detail: detail, Percent: percent})
	}
	return result
}

func bestEmployees(all []metricRow, period string) []response.BestCategory {
	prefix := map[string]string{"month": "本月", "quarter": "本季度", "half": "本半年", "year": "本年度"}[period]
	build := func(kind string) []response.BestEmployee {
		rows := append([]metricRow(nil), all...)
		sort.SliceStable(rows, func(i, j int) bool {
			if kind == "leave" {
				return rows[i].leave < rows[j].leave
			}
			if kind == "late" {
				return rows[i].lateMinutes < rows[j].lateMinutes
			}
			return rows[i].earlyMinutes < rows[j].earlyMinutes
		})
		if len(rows) > 3 {
			rows = rows[:3]
		}
		result := make([]response.BestEmployee, 0, len(rows))
		for i, row := range rows {
			value := fmt.Sprintf("%d分钟", row.earlyMinutes)
			if kind == "leave" {
				value = fmt.Sprintf("%g天", row.leave)
			}
			if kind == "late" {
				value = fmt.Sprintf("%d分钟", row.lateMinutes)
			}
			result = append(result, response.BestEmployee{Rank: fmt.Sprint(i + 1), Name: row.name, Value: value})
		}
		return result
	}
	return []response.BestCategory{{Title: "请假最少", Icon: "假", Description: prefix + "累计请假", Employees: build("leave")}, {Title: "迟到最少", Icon: "迟", Description: prefix + "累计迟到", Employees: build("late")}, {Title: "早退最少", Icon: "早", Description: prefix + "累计早退", Employees: build("early")}}
}
