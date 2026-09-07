package mysql

import (
	"context"
	"fmt"

	"friends-records/api/response"
	"friends-records/internal/apperror"
)

func changeLabel(v string) string {
	return map[string]string{"hire": "入职", "regularize": "转正", "transfer": "调岗", "promotion": "晋升", "demotion": "降职", "leave": "离职"}[v]
}

func (s *Store) EmploymentChanges(ctx context.Context) ([]response.EmploymentChange, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT c.id,e.employee_no,e.name,c.change_type,CONCAT_WS(' / ',NULLIF(bd.name,''),NULLIF(bp.name,''),NULLIF(c.before_status,'')),CONCAT_WS(' / ',NULLIF(ad.name,''),NULLIF(ap.name,''),NULLIF(c.after_status,'')),DATE_FORMAT(c.effective_on,'%Y-%m-%d'),c.reason,COALESCE(u.display_name,'系统') FROM employment_changes c JOIN employees e ON e.id=c.employee_id LEFT JOIN departments bd ON bd.id=c.before_department_id LEFT JOIN positions bp ON bp.id=c.before_position_id LEFT JOIN departments ad ON ad.id=c.after_department_id LEFT JOIN positions ap ON ap.id=c.after_position_id LEFT JOIN admin_users u ON u.id=c.operator_id ORDER BY c.effective_on DESC,c.id DESC LIMIT 500`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询人事异动失败")
	}
	defer rows.Close()
	result := make([]response.EmploymentChange, 0)
	for rows.Next() {
		var v response.EmploymentChange
		var kind string
		if err := rows.Scan(&v.ID, &v.EmployeeNo, &v.Name, &kind, &v.Before, &v.After, &v.EffectiveOn, &v.Reason, &v.Operator); err != nil {
			return nil, apperror.Wrap(err, "读取人事异动失败")
		}
		v.Type = changeLabel(kind)
		result = append(result, v)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历人事异动失败")
	}
	return result, nil
}

func (s *Store) ChangeSummary(ctx context.Context) (response.ChangeSummary, error) {
	if err := s.ready(); err != nil {
		return response.ChangeSummary{}, err
	}
	var v response.ChangeSummary
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(change_type='hire' AND effective_on>=DATE_FORMAT(CURDATE(),'%Y-%m-01')),0),COALESCE(SUM(change_type='regularize' AND effective_on>=DATE_FORMAT(CURDATE(),'%Y-%m-01')),0),COALESCE(SUM(change_type='promotion' AND YEAR(effective_on)=YEAR(CURDATE())),0),COALESCE(SUM(change_type='leave' AND effective_on>=DATE_FORMAT(CURDATE(),'%Y-%m-01')),0) FROM employment_changes`).Scan(&v.JoinedThisMonth, &v.RegularizedThisMonth, &v.PromotedThisYear, &v.LeftThisMonth)
	if err != nil {
		return response.ChangeSummary{}, apperror.Wrap(err, "统计人事异动失败")
	}
	return v, nil
}

func (s *Store) SalaryAdjustments(ctx context.Context) ([]response.SalaryAdjustment, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT s.id,e.employee_no,e.name,s.before_salary,s.after_salary,DATE_FORMAT(s.effective_on,'%Y-%m-%d'),s.reason,COALESCE(u.display_name,'系统') FROM salary_adjustments s JOIN employees e ON e.id=s.employee_id LEFT JOIN admin_users u ON u.id=s.operator_id ORDER BY s.effective_on DESC,s.id DESC LIMIT 500`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询调薪记录失败")
	}
	defer rows.Close()
	result := make([]response.SalaryAdjustment, 0)
	for rows.Next() {
		var v response.SalaryAdjustment
		var before, after float64
		if err := rows.Scan(&v.ID, &v.EmployeeNo, &v.Name, &before, &after, &v.EffectiveOn, &v.Reason, &v.Operator); err != nil {
			return nil, apperror.Wrap(err, "读取调薪记录失败")
		}
		v.Before, v.After, v.Increase = Money(before), Money(after), fmt.Sprintf("%+.2f", after-before)
		result = append(result, v)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历调薪记录失败")
	}
	return result, nil
}

func (s *Store) SalarySummary(ctx context.Context) (response.SalarySummary, error) {
	if err := s.ready(); err != nil {
		return response.SalarySummary{}, err
	}
	var v response.SalarySummary
	var avg, percent, total float64
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(DISTINCT CASE WHEN YEAR(effective_on)=YEAR(CURDATE()) THEN employee_id END),COALESCE(AVG(CASE WHEN YEAR(effective_on)=YEAR(CURDATE()) THEN after_salary-before_salary END),0),COALESCE(AVG(CASE WHEN YEAR(effective_on)=YEAR(CURDATE()) AND before_salary>0 THEN (after_salary-before_salary)/before_salary*100 END),0),COALESCE((SELECT SUM(current_salary) FROM employees WHERE employment_status IN ('active','probation')),0),COALESCE(SUM(effective_on>CURDATE()),0) FROM salary_adjustments`).Scan(&v.AdjustedEmployees, &avg, &percent, &total, &v.PendingCount)
	if err != nil {
		return response.SalarySummary{}, apperror.Wrap(err, "统计调薪数据失败")
	}
	v.AverageAdjustment, v.AveragePercent, v.CurrentSalary = Money(avg), fmt.Sprintf("%.1f%%", percent), Money(total)
	return v, nil
}
