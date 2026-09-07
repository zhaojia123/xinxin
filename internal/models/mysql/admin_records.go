package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"friends-records/api/request"
	"friends-records/internal/apperror"
)

type AdminOption struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	DepartmentID uint64 `json:"department_id"`
	PositionID   uint64 `json:"position_id"`
	Status       string `json:"status"`
	Salary       string `json:"salary"`
}

func (s *Store) AdminOptions(ctx context.Context) (map[string]any, error) {
	base, err := s.MiniOptions(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,CONCAT(name,' · ',employee_no),COALESCE(department_id,0),COALESCE(position_id,0),employment_status,CAST(current_salary AS CHAR) FROM employees ORDER BY employment_status='left',employee_no`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询员工选项失败")
	}
	defer rows.Close()
	employees := []AdminOption{}
	for rows.Next() {
		var v AdminOption
		if err := rows.Scan(&v.ID, &v.Name, &v.DepartmentID, &v.PositionID, &v.Status, &v.Salary); err != nil {
			return nil, err
		}
		employees = append(employees, v)
	}
	return map[string]any{"accounts": base["accounts"], "categories": base["categories"], "departments": base["departments"], "positions": base["positions"], "employees": employees}, rows.Err()
}

type DepartmentRecord struct {
	ID uint64 `json:"id"`
	request.DepartmentInput
}

func (s *Store) DepartmentRecord(ctx context.Context, id uint64) (DepartmentRecord, error) {
	if err := s.ready(); err != nil {
		return DepartmentRecord{}, err
	}
	var v DepartmentRecord
	err := s.DB.QueryRowContext(ctx, `SELECT id,department_no,name,COALESCE(parent_id,0),COALESCE(manager_employee_id,0),sort_order,enabled,remark FROM departments WHERE id=?`, id).Scan(&v.ID, &v.DepartmentNo, &v.Name, &v.ParentID, &v.ManagerEmployeeID, &v.SortOrder, &v.Enabled, &v.Remark)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func (s *Store) SaveDepartment(ctx context.Context, id uint64, v request.DepartmentInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if id != 0 {
		var found uint64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM departments WHERE id=? FOR UPDATE`, id).Scan(&found); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrNotFound
			}
			return 0, err
		}
	}
	if v.ParentID == id && id != 0 {
		return 0, MiniInputError{"部门不能把自己设为上级"}
	}
	if v.ParentID != 0 {
		if err := requireReference(ctx, tx, `SELECT id FROM departments WHERE id=? FOR SHARE`, "上级部门不存在", v.ParentID); err != nil {
			return 0, err
		}
	}
	if v.ManagerEmployeeID != 0 {
		if err := requireReference(ctx, tx, `SELECT id FROM employees WHERE id=? FOR SHARE`, "负责人员工不存在", v.ManagerEmployeeID); err != nil {
			return 0, err
		}
	}
	if id == 0 {
		result, e := tx.ExecContext(ctx, `INSERT INTO departments(department_no,name,parent_id,manager_employee_id,sort_order,enabled,remark) VALUES(?,?,?,?,?,?,?)`, v.DepartmentNo, v.Name, optionalID(v.ParentID), optionalID(v.ManagerEmployeeID), v.SortOrder, v.Enabled, v.Remark)
		if e != nil {
			return 0, e
		}
		inserted, e := result.LastInsertId()
		if e != nil {
			return 0, e
		}
		id = uint64(inserted)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE departments SET department_no=?,name=?,parent_id=?,manager_employee_id=?,sort_order=?,enabled=?,remark=? WHERE id=?`, v.DepartmentNo, v.Name, optionalID(v.ParentID), optionalID(v.ManagerEmployeeID), v.SortOrder, v.Enabled, v.Remark, id)
		if err != nil {
			return 0, err
		}
	}
	if err := miniAudit(ctx, tx, 0, "organization", id, v.Name); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

type PositionRecord struct {
	ID uint64 `json:"id"`
	request.PositionInput
}

func (s *Store) PositionRecord(ctx context.Context, id uint64) (PositionRecord, error) {
	if err := s.ready(); err != nil {
		return PositionRecord{}, err
	}
	var v PositionRecord
	err := s.DB.QueryRowContext(ctx, `SELECT id,position_no,name,department_id,level_name,COALESCE(CAST(min_salary AS CHAR),''),COALESCE(CAST(max_salary AS CHAR),''),sort_order,enabled,remark FROM positions WHERE id=?`, id).Scan(&v.ID, &v.PositionNo, &v.Name, &v.DepartmentID, &v.LevelName, &v.MinSalary, &v.MaxSalary, &v.SortOrder, &v.Enabled, &v.Remark)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func optionalText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func (s *Store) SavePosition(ctx context.Context, id uint64, v request.PositionInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if id != 0 {
		var found uint64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM positions WHERE id=? FOR UPDATE`, id).Scan(&found); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrNotFound
			}
			return 0, err
		}
	}
	if err := requireReference(ctx, tx, `SELECT id FROM departments WHERE id=? AND enabled=1 FOR SHARE`, "所属部门不存在或已停用", v.DepartmentID); err != nil {
		return 0, err
	}
	if id == 0 {
		result, e := tx.ExecContext(ctx, `INSERT INTO positions(position_no,name,department_id,level_name,min_salary,max_salary,sort_order,enabled,remark) VALUES(?,?,?,?,?,?,?,?,?)`, v.PositionNo, v.Name, v.DepartmentID, v.LevelName, optionalText(v.MinSalary), optionalText(v.MaxSalary), v.SortOrder, v.Enabled, v.Remark)
		if e != nil {
			return 0, e
		}
		inserted, e := result.LastInsertId()
		if e != nil {
			return 0, e
		}
		id = uint64(inserted)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE positions SET position_no=?,name=?,department_id=?,level_name=?,min_salary=?,max_salary=?,sort_order=?,enabled=?,remark=? WHERE id=?`, v.PositionNo, v.Name, v.DepartmentID, v.LevelName, optionalText(v.MinSalary), optionalText(v.MaxSalary), v.SortOrder, v.Enabled, v.Remark, id)
		if err != nil {
			return 0, err
		}
	}
	if err := miniAudit(ctx, tx, 0, "organization", id, v.Name); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

type AttendanceRecordInput struct {
	ID uint64 `json:"id"`
	request.AttendanceInput
	Status string `json:"status"`
}

func (s *Store) AttendanceRecordInput(ctx context.Context, id uint64) (AttendanceRecordInput, error) {
	if err := s.ready(); err != nil {
		return AttendanceRecordInput{}, err
	}
	var v AttendanceRecordInput
	err := s.DB.QueryRowContext(ctx, `SELECT id,employee_id,category,record_type,DATE_FORMAT(occurred_on,'%Y-%m-%d'),COALESCE(DATE_FORMAT(start_time,'%H:%i'),''),COALESCE(DATE_FORMAT(end_time,'%H:%i'),''),duration_minutes,CAST(duration_days AS CHAR),reason,status FROM attendance_records WHERE id=?`, id).Scan(&v.ID, &v.EmployeeID, &v.Category, &v.RecordType, &v.OccurredOn, &v.StartTime, &v.EndTime, &v.DurationMinutes, &v.DurationDays, &v.Reason, &v.Status)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func (s *Store) SaveAttendance(ctx context.Context, id uint64, v request.AttendanceInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if v.Category == "leave" {
		v.DurationMinutes = 0
	} else {
		v.DurationDays = "0"
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	status := "pending"
	if v.Category == "exception" {
		status = "recorded"
	}
	if id != 0 {
		if err := tx.QueryRowContext(ctx, `SELECT status FROM attendance_records WHERE id=? FOR UPDATE`, id).Scan(&status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrNotFound
			}
			return 0, err
		}
	}
	if err := requireReference(ctx, tx, `SELECT id FROM employees WHERE id=? FOR SHARE`, "员工不存在", v.EmployeeID); err != nil {
		return 0, err
	}
	if id == 0 {
		result, e := tx.ExecContext(ctx, `INSERT INTO attendance_records(employee_id,category,record_type,occurred_on,start_time,end_time,duration_minutes,duration_days,reason,source,status) VALUES(?,?,?,?,?,?,?,?,?,'manual',?)`, v.EmployeeID, v.Category, v.RecordType, v.OccurredOn, optionalText(v.StartTime), optionalText(v.EndTime), v.DurationMinutes, v.DurationDays, v.Reason, status)
		if e != nil {
			return 0, e
		}
		inserted, e := result.LastInsertId()
		if e != nil {
			return 0, e
		}
		id = uint64(inserted)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE attendance_records SET employee_id=?,category=?,record_type=?,occurred_on=?,start_time=?,end_time=?,duration_minutes=?,duration_days=?,reason=? WHERE id=?`, v.EmployeeID, v.Category, v.RecordType, v.OccurredOn, optionalText(v.StartTime), optionalText(v.EndTime), v.DurationMinutes, v.DurationDays, v.Reason, id)
		if err != nil {
			return 0, err
		}
	}
	if err := miniAudit(ctx, tx, 0, "attendance", id, v.RecordType); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}
func (s *Store) SetAttendanceStatus(ctx context.Context, id uint64, status string) error {
	if err := s.ready(); err != nil {
		return err
	}
	if status != "approved" && status != "rejected" {
		return MiniInputError{"审批状态不正确"}
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE attendance_records SET status=?,reviewed_at=NOW() WHERE id=? AND status='pending'`, status, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return MiniInputError{"只有待审批记录可以通过或驳回"}
	}
	return nil
}

type ChangeRecordInput struct {
	ID uint64 `json:"id"`
	request.ChangeInput
}

func (s *Store) ChangeRecordInput(ctx context.Context, id uint64) (ChangeRecordInput, error) {
	if err := s.ready(); err != nil {
		return ChangeRecordInput{}, err
	}
	var v ChangeRecordInput
	err := s.DB.QueryRowContext(ctx, `SELECT id,employee_id,change_type,COALESCE(after_department_id,0),COALESCE(after_position_id,0),COALESCE(after_status,''),DATE_FORMAT(effective_on,'%Y-%m-%d'),reason FROM employment_changes WHERE id=?`, id).Scan(&v.ID, &v.EmployeeID, &v.ChangeType, &v.AfterDepartmentID, &v.AfterPositionID, &v.AfterStatus, &v.EffectiveOn, &v.Reason)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func (s *Store) CreateEmploymentChange(ctx context.Context, v request.ChangeInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var beforeDepartment, beforePosition uint64
	var beforeStatus string
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(department_id,0),COALESCE(position_id,0),employment_status FROM employees WHERE id=? FOR UPDATE`, v.EmployeeID).Scan(&beforeDepartment, &beforePosition, &beforeStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, MiniInputError{"员工不存在"}
		}
		return 0, err
	}
	if v.AfterDepartmentID != 0 {
		if err := requireReference(ctx, tx, `SELECT id FROM departments WHERE id=? AND enabled=1 FOR SHARE`, "异动后部门不存在或已停用", v.AfterDepartmentID); err != nil {
			return 0, err
		}
	}
	if v.AfterPositionID != 0 {
		if err := requireReference(ctx, tx, `SELECT id FROM positions WHERE id=? AND department_id=? AND enabled=1 FOR SHARE`, "异动后岗位与部门不匹配", v.AfterPositionID, v.AfterDepartmentID); err != nil {
			return 0, err
		}
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO employment_changes(employee_id,change_type,before_department_id,after_department_id,before_position_id,after_position_id,before_status,after_status,effective_on,reason) VALUES(?,?,?,?,?,?,?,?,?,?)`, v.EmployeeID, v.ChangeType, optionalID(beforeDepartment), optionalID(v.AfterDepartmentID), optionalID(beforePosition), optionalID(v.AfterPositionID), beforeStatus, v.AfterStatus, v.EffectiveOn, v.Reason)
	if err != nil {
		return 0, err
	}
	inserted, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	leftOn := any(nil)
	if v.AfterStatus == "left" {
		leftOn = v.EffectiveOn
	}
	_, err = tx.ExecContext(ctx, `UPDATE employees SET department_id=?,position_id=?,employment_status=?,left_on=?,regularized_on=IF(?='regularize',?,regularized_on),joined_on=IF(?='hire',?,joined_on) WHERE id=?`, optionalID(v.AfterDepartmentID), optionalID(v.AfterPositionID), v.AfterStatus, leftOn, v.ChangeType, v.EffectiveOn, v.ChangeType, v.EffectiveOn, v.EmployeeID)
	if err != nil {
		return 0, err
	}
	if err := miniAudit(ctx, tx, 0, "employment_changes", uint64(inserted), v.ChangeType); err != nil {
		return 0, err
	}
	return uint64(inserted), tx.Commit()
}

type SalaryRecordInput struct {
	ID uint64 `json:"id"`
	request.SalaryAdjustmentInput
	BeforeSalary string `json:"before_salary"`
}

func (s *Store) SalaryRecordInput(ctx context.Context, id uint64) (SalaryRecordInput, error) {
	if err := s.ready(); err != nil {
		return SalaryRecordInput{}, err
	}
	var v SalaryRecordInput
	err := s.DB.QueryRowContext(ctx, `SELECT id,employee_id,CAST(before_salary AS CHAR),CAST(after_salary AS CHAR),DATE_FORMAT(effective_on,'%Y-%m-%d'),reason FROM salary_adjustments WHERE id=?`, id).Scan(&v.ID, &v.EmployeeID, &v.BeforeSalary, &v.AfterSalary, &v.EffectiveOn, &v.Reason)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func (s *Store) CreateSalaryAdjustment(ctx context.Context, v request.SalaryAdjustmentInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var before string
	if err := tx.QueryRowContext(ctx, `SELECT CAST(current_salary AS CHAR) FROM employees WHERE id=? FOR UPDATE`, v.EmployeeID).Scan(&before); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, MiniInputError{"员工不存在"}
		}
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO salary_adjustments(employee_id,before_salary,after_salary,effective_on,reason) VALUES(?,?,?,?,?)`, v.EmployeeID, before, v.AfterSalary, v.EffectiveOn, v.Reason)
	if err != nil {
		return 0, err
	}
	inserted, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	date, _ := time.ParseInLocation("2006-01-02", v.EffectiveOn, time.Local)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if !date.After(today) {
		if _, err := tx.ExecContext(ctx, `UPDATE employees SET current_salary=? WHERE id=?`, v.AfterSalary, v.EmployeeID); err != nil {
			return 0, err
		}
	}
	if err := miniAudit(ctx, tx, 0, "salary", uint64(inserted), v.AfterSalary); err != nil {
		return 0, err
	}
	return uint64(inserted), tx.Commit()
}

func (s *Store) LeaveEmployee(ctx context.Context, id uint64, date string) error {
	if err := s.ready(); err != nil {
		return err
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return MiniInputError{"离职日期不正确"}
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var department, position uint64
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(department_id,0),COALESCE(position_id,0),employment_status FROM employees WHERE id=? FOR UPDATE`, id).Scan(&department, &position, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if status == "left" {
		return MiniInputError{"该员工已经离职"}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE employees SET employment_status='left',left_on=? WHERE id=?`, date, id); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO employment_changes(employee_id,change_type,before_department_id,after_department_id,before_position_id,after_position_id,before_status,after_status,effective_on,reason) VALUES(?,'leave',?,?,?,?,?,'left',?,'后台办理离职')`, id, optionalID(department), optionalID(department), optionalID(position), optionalID(position), status, date)
	if err != nil {
		return err
	}
	changeID, _ := result.LastInsertId()
	if err := miniAudit(ctx, tx, 0, "employment_changes", uint64(changeID), "员工离职"); err != nil {
		return err
	}
	return tx.Commit()
}
