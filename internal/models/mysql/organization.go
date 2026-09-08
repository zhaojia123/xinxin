package mysql

import (
	"context"

	"friends-records/api/response"
	"friends-records/internal/apperror"
)

func (s *Store) Departments(ctx context.Context) ([]response.Department, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT d.id,d.department_no,d.name,COALESCE(parent.name,''),COALESCE(manager.name,''),d.enabled,d.remark,COUNT(DISTINCT p.id),COUNT(DISTINCT e.id) FROM departments d LEFT JOIN departments parent ON parent.id=d.parent_id LEFT JOIN employees manager ON manager.id=d.manager_employee_id LEFT JOIN positions p ON p.department_id=d.id AND p.active=1 LEFT JOIN employees e ON e.department_id=d.id AND e.employment_status<>'left' AND e.active=1 WHERE d.active=1 GROUP BY d.id,d.department_no,d.name,parent.name,manager.name,d.enabled,d.remark,d.sort_order ORDER BY d.sort_order,d.id`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询部门列表失败")
	}
	defer rows.Close()
	result := make([]response.Department, 0)
	for rows.Next() {
		var item response.Department
		var enabled bool
		if err := rows.Scan(&item.ID, &item.DepartmentNo, &item.Name, &item.ParentName, &item.ManagerName, &enabled, &item.Remark, &item.PositionCount, &item.EmployeeCount); err != nil {
			return nil, apperror.Wrap(err, "读取部门列表失败")
		}
		if enabled {
			item.Status, item.StatusClass = "启用", "active"
		} else {
			item.Status, item.StatusClass = "停用", "inactive"
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历部门列表失败")
	}
	return result, nil
}

func (s *Store) Positions(ctx context.Context) ([]response.Position, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT p.id,p.department_id,p.position_no,p.name,d.name,p.level_name,p.min_salary,p.max_salary,p.enabled,p.remark,COUNT(e.id) FROM positions p JOIN departments d ON d.id=p.department_id LEFT JOIN employees e ON e.position_id=p.id AND e.employment_status<>'left' AND e.active=1 WHERE p.active=1 AND d.active=1 GROUP BY p.id,p.department_id,p.position_no,p.name,d.name,p.level_name,p.min_salary,p.max_salary,p.enabled,p.remark,p.sort_order ORDER BY d.sort_order,p.sort_order,p.id`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询岗位列表失败")
	}
	defer rows.Close()
	result := make([]response.Position, 0)
	for rows.Next() {
		var item response.Position
		var min, max *float64
		var enabled bool
		if err := rows.Scan(&item.ID, &item.DepartmentID, &item.PositionNo, &item.Name, &item.Department, &item.LevelName, &min, &max, &enabled, &item.Remark, &item.EmployeeCount); err != nil {
			return nil, apperror.Wrap(err, "读取岗位列表失败")
		}
		if min != nil && max != nil {
			item.SalaryRange = Money(*min) + " - " + Money(*max)
		} else {
			item.SalaryRange = "—"
		}
		if enabled {
			item.Status, item.StatusClass = "启用", "active"
		} else {
			item.Status, item.StatusClass = "停用", "inactive"
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历岗位列表失败")
	}
	return result, nil
}

func (s *Store) OrganizationSummary(ctx context.Context) (response.OrganizationSummary, error) {
	if err := s.ready(); err != nil {
		return response.OrganizationSummary{}, err
	}
	var v response.OrganizationSummary
	err := s.DB.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM departments WHERE enabled=1 AND active=1),(SELECT COUNT(*) FROM positions WHERE enabled=1 AND active=1),(SELECT COUNT(*) FROM employees WHERE employment_status IN ('active','probation') AND active=1)`).Scan(&v.DepartmentCount, &v.PositionCount, &v.ActiveEmployeeCount)
	if err != nil {
		return response.OrganizationSummary{}, apperror.Wrap(err, "统计部门岗位失败")
	}
	return v, nil
}
