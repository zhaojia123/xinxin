package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"friends-records/api/response"
	"friends-records/internal/apperror"
)

const employeeSelect = `SELECT e.id,e.employee_no,e.name,e.gender,e.id_card,e.mobile,COALESCE(e.department_id,0),COALESCE(d.name,''),COALESCE(e.position_id,0),COALESCE(p.name,''),e.employment_status,COALESCE(DATE_FORMAT(e.joined_on,'%Y-%m-%d'),''),COALESCE(DATE_FORMAT(e.regularized_on,'%Y-%m-%d'),''),e.employment_type,e.current_salary,e.education,e.hometown,COALESCE(hc.id,0),COALESCE(hc.file_url,''),COALESCE(DATE_FORMAT(hc.issued_on,'%Y-%m-%d'),''),COALESCE(DATE_FORMAT(hc.expires_on,'%Y-%m-%d'),''),COALESCE(DATEDIFF(hc.expires_on,CURDATE()),0) FROM employees e LEFT JOIN departments d ON d.id=e.department_id LEFT JOIN positions p ON p.id=e.position_id LEFT JOIN employee_health_certificates hc ON hc.id=(SELECT current_hc.id FROM employee_health_certificates current_hc WHERE current_hc.employee_id=e.id AND current_hc.is_current=1 ORDER BY current_hc.expires_on DESC,current_hc.id DESC LIMIT 1)`

type scanner interface{ Scan(...any) error }

func scanEmployee(row scanner) (response.Employee, error) {
	var item response.Employee
	var gender, status, employmentType, idCard, mobile string
	var salary float64
	err := row.Scan(&item.ID, &item.EmployeeNo, &item.Name, &gender, &idCard, &mobile, &item.DepartmentID, &item.Department, &item.PositionID, &item.Position, &status, &item.JoinedOn, &item.RegularizedOn, &employmentType, &salary, &item.Education, &item.Hometown, &item.HealthCertificateID, &item.HealthCertificateURL, &item.HealthCertificateIssuedOn, &item.HealthCertificateExpiresOn, &item.HealthCertificateDaysRemaining)
	if err != nil {
		return response.Employee{}, err
	}
	switch gender {
	case "male":
		item.Gender = "男"
	case "female":
		item.Gender = "女"
	default:
		item.Gender = "未知"
	}
	item.IDCard = Mask(idCard, 3, 4)
	item.Mobile = Mask(mobile, 3, 4)
	item.Status, item.StatusClass = Status(status)
	switch employmentType {
	case "part_time":
		item.EmploymentType = "兼职"
	case "intern":
		item.EmploymentType = "实习"
	default:
		item.EmploymentType = "正式员工"
	}
	item.Salary, item.SalaryValue = Money(salary), Decimal(salary)
	item.HealthCertificateStatus, item.HealthCertificateStatusClass = healthCertificateStatus(item.HealthCertificateID, item.HealthCertificateDaysRemaining)
	return item, nil
}

func healthCertificateStatus(id uint64, days int) (string, string) {
	if id == 0 {
		return "未上传", "pending"
	}
	if days < 0 {
		return "已过期", "inactive"
	}
	if days <= 20 {
		return "即将到期", "warning"
	}
	return "有效", "active"
}

func (s *Store) Employees(ctx context.Context) ([]response.Employee, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, employeeSelect+` WHERE e.active=1 ORDER BY e.employment_status='left',e.employee_no`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询员工列表失败")
	}
	defer rows.Close()
	result := make([]response.Employee, 0)
	for rows.Next() {
		item, err := scanEmployee(rows)
		if err != nil {
			return nil, apperror.Wrap(err, "读取员工列表失败")
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历员工列表失败")
	}
	return result, nil
}

func (s *Store) Employee(ctx context.Context, id uint64) (response.Employee, bool, error) {
	if err := s.ready(); err != nil {
		return response.Employee{}, false, err
	}
	item, err := scanEmployee(s.DB.QueryRowContext(ctx, employeeSelect+` WHERE e.id=? AND e.active=1 LIMIT 1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return response.Employee{}, false, nil
	}
	if err != nil {
		return response.Employee{}, false, apperror.Wrap(err, "查询员工详情失败")
	}
	return item, true, nil
}

func (s *Store) EmployeeSummary(ctx context.Context) (response.EmployeeSummary, error) {
	if err := s.ready(); err != nil {
		return response.EmployeeSummary{}, err
	}
	var v response.EmployeeSummary
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(employment_status='active'),0),COALESCE(SUM(employment_status='probation'),0),COALESCE(SUM(employment_status='left' AND left_on>=DATE_FORMAT(CURDATE(),'%Y-%m-01') AND left_on<DATE_ADD(DATE_FORMAT(CURDATE(),'%Y-%m-01'),INTERVAL 1 MONTH)),0) FROM employees WHERE active=1`).Scan(&v.Total, &v.Active, &v.Probation, &v.LeftThisMonth)
	if err != nil {
		return response.EmployeeSummary{}, apperror.Wrap(err, "统计员工数据失败")
	}
	return v, nil
}

func (s *Store) HealthCertificateReminders(ctx context.Context) ([]response.HealthCertificateReminder, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT e.id,e.employee_no,e.name,DATE_FORMAT(hc.expires_on,'%Y-%m-%d'),DATEDIFF(hc.expires_on,CURDATE()) FROM employee_health_certificates hc JOIN employees e ON e.id=hc.employee_id WHERE hc.is_current=1 AND e.active=1 AND e.employment_status IN ('active','probation') AND hc.expires_on>=CURDATE() AND hc.expires_on<DATE_ADD(CURDATE(),INTERVAL 20 DAY) ORDER BY hc.expires_on,e.employee_no`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询健康证到期提醒失败")
	}
	defer rows.Close()
	result := make([]response.HealthCertificateReminder, 0)
	for rows.Next() {
		var item response.HealthCertificateReminder
		if err := rows.Scan(&item.EmployeeID, &item.EmployeeNo, &item.Name, &item.ExpiresOn, &item.DaysRemaining); err != nil {
			return nil, apperror.Wrap(err, "读取健康证到期提醒失败")
		}
		if item.DaysRemaining < 0 {
			item.Status, item.StatusClass = "已过期", "inactive"
		} else {
			item.Status, item.StatusClass = "剩余 "+fmt.Sprint(item.DaysRemaining)+" 天", "warning"
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历健康证到期提醒失败")
	}
	return result, nil
}

func (s *Store) EmployeeEvents(ctx context.Context, id uint64) ([]response.EmployeeEvent, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT event_date,title,detail,color FROM (
		SELECT occurred_on event_date,CONCAT(record_type,' · ',CASE status WHEN 'approved' THEN '已通过' WHEN 'pending' THEN '待审批' WHEN 'recorded' THEN '已记录' WHEN 'confirmed' THEN '已确认' ELSE '已结束' END) title,reason detail,'blue' color FROM attendance_records WHERE employee_id=? AND active=1
		UNION ALL SELECT effective_on,CONCAT('人事异动 · ',CASE change_type WHEN 'hire' THEN '入职' WHEN 'regularize' THEN '转正' WHEN 'transfer' THEN '调岗' WHEN 'promotion' THEN '晋升' WHEN 'demotion' THEN '降职' WHEN 'leave' THEN '离职' END),reason,'green' FROM employment_changes WHERE employee_id=?
		UNION ALL SELECT effective_on,CONCAT('工资调整至 ',FORMAT(after_salary,2)),reason,'green' FROM salary_adjustments WHERE employee_id=?
		UNION ALL SELECT issued_on,CONCAT('健康证更新 · 到期日期 ',DATE_FORMAT(expires_on,'%Y-%m-%d')),original_name,'blue' FROM employee_health_certificates WHERE employee_id=?
	) events ORDER BY event_date DESC LIMIT 100`, id, id, id, id)
	if err != nil {
		return nil, apperror.Wrap(err, "查询员工动态失败")
	}
	defer rows.Close()
	result := make([]response.EmployeeEvent, 0)
	for rows.Next() {
		var event response.EmployeeEvent
		if err := rows.Scan(&event.Date, &event.Title, &event.Detail, &event.Color); err != nil {
			return nil, apperror.Wrap(err, "读取员工动态失败")
		}
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历员工动态失败")
	}
	return result, nil
}
