package mysql

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"friends-records/api/response"
	"friends-records/internal/apperror"
)

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
	default:
		return "待审批", "pending"
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
	rows, err := s.DB.QueryContext(ctx, `SELECT a.id,e.employee_no,e.name,a.category,a.record_type,DATE_FORMAT(a.occurred_on,'%Y-%m-%d'),COALESCE(DATE_FORMAT(a.start_time,'%H:%i'),''),a.duration_minutes,a.duration_days,a.reason,a.status,a.source FROM attendance_records a JOIN employees e ON e.id=a.employee_id WHERE a.active=1 ORDER BY a.occurred_on DESC,a.id DESC LIMIT 500`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询请假与异常记录失败")
	}
	defer rows.Close()
	result := make([]response.AttendanceRecord, 0)
	for rows.Next() {
		var item response.AttendanceRecord
		var category, status, source string
		var minutes int
		var days float64
		if err := rows.Scan(&item.ID, &item.EmployeeNo, &item.Name, &category, &item.Type, &item.Date, &item.Time, &minutes, &days, &item.Reason, &status, &source); err != nil {
			return nil, apperror.Wrap(err, "读取请假与异常记录失败")
		}
		if category == "leave" {
			item.Category, item.Duration = "请假", fmt.Sprintf("%g天", days)
		} else {
			item.Category, item.Duration = "异常", fmt.Sprintf("%d分钟", minutes)
		}
		item.Status, item.StatusClass = attendanceStatus(status)
		item.Source = sourceLabel(source)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历请假与异常记录失败")
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
	if err := s.ready(); err != nil {
		return response.AttendanceStatistics{}, err
	}
	period, label, start, end := periodRange(period, time.Now())
	rows, err := s.DB.QueryContext(ctx, `SELECT e.employee_no,e.name,COALESCE(SUM(CASE WHEN a.category='leave' AND a.status='approved' THEN a.duration_days ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.category='leave' AND a.status='approved' THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.record_type='迟到' AND a.status IN ('recorded','confirmed','approved') THEN a.duration_minutes ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.record_type='迟到' AND a.status IN ('recorded','confirmed','approved') THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.record_type='早退' AND a.status IN ('recorded','confirmed','approved') THEN a.duration_minutes ELSE 0 END),0),COALESCE(SUM(CASE WHEN a.record_type='早退' AND a.status IN ('recorded','confirmed','approved') THEN 1 ELSE 0 END),0) FROM employees e LEFT JOIN attendance_records a ON a.employee_id=e.id AND a.occurred_on>=? AND a.occurred_on<? AND a.active=1 WHERE e.employment_status IN ('active','probation') AND e.active=1 GROUP BY e.id,e.employee_no,e.name ORDER BY e.employee_no`, start, end)
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
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM attendance_records WHERE status='pending' AND active=1 AND occurred_on>=? AND occurred_on<?`, start, end).Scan(&result.PendingCount); err != nil {
		return response.AttendanceStatistics{}, apperror.Wrap(err, "统计待审批记录失败")
	}
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
