package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"friends-records/api/request"
	"friends-records/api/response"
)

// MiniInputError 表示可以直接展示给小程序用户的业务错误。
type MiniInputError struct{ Message string }

func (e MiniInputError) Error() string { return e.Message }

func (s *Store) MiniEnabled(ctx context.Context, id uint64) (bool, error) {
	if err := s.ready(); err != nil {
		return false, err
	}
	var enabled bool
	err := s.DB.QueryRowContext(ctx, `SELECT enabled FROM mini_users WHERE id=?`, id).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return enabled, err
}

// MiniPermission 查询小程序用户是否拥有指定模块权限。
func (s *Store) MiniPermission(ctx context.Context, id uint64, module string) (bool, error) {
	return s.MiniPermissionAction(ctx, id, module, "view")
}

// MiniPermissionAction 查询用户对模块的具体操作权限。
func (s *Store) MiniPermissionAction(ctx context.Context, id uint64, module, action string) (bool, error) {
	if err := s.ready(); err != nil {
		return false, err
	}
	column := map[string]string{"view": "can_view", "create": "can_create", "edit": "can_edit", "delete": "can_delete"}[action]
	if column == "" {
		return false, nil
	}
	var allowed, moduleEnabled bool
	err := s.DB.QueryRowContext(ctx, `SELECT p.enabled,p.`+column+` FROM mini_user_permissions p JOIN mini_users u ON u.id=p.mini_user_id WHERE p.mini_user_id=? AND p.module_key=? AND u.enabled=1 LIMIT 1`, id, module).Scan(&moduleEnabled, &allowed)
	if err == nil {
		return moduleEnabled && allowed, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	// 兼容未执行权限表迁移的旧数据库，迁移完成后新模块直接使用权限表。
	if action != "view" {
		return false, nil
	}
	switch module {
	case "ledger":
		err = s.DB.QueryRowContext(ctx, `SELECT can_ledger FROM mini_users WHERE id=? AND enabled=1`, id).Scan(&allowed)
	case "purchases":
		err = s.DB.QueryRowContext(ctx, `SELECT can_purchases FROM mini_users WHERE id=? AND enabled=1`, id).Scan(&allowed)
	case "employees":
		err = s.DB.QueryRowContext(ctx, `SELECT can_employees FROM mini_users WHERE id=? AND enabled=1`, id).Scan(&allowed)
	default:
		return false, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return allowed, err
}

// MiniPermissions 返回用户当前启用的模块标识，供小程序动态生成导航。
func (s *Store) MiniPermissions(ctx context.Context, id uint64) ([]string, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT module_key,enabled,can_view FROM mini_user_permissions WHERE mini_user_id=? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	permissions := []string{}
	hasRows := false
	for rows.Next() {
		var key string
		var enabled, canView bool
		if err := rows.Scan(&key, &enabled, &canView); err != nil {
			return nil, err
		}
		hasRows = true
		if enabled && canView {
			permissions = append(permissions, key)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if hasRows {
		return permissions, nil
	}
	// 兼容旧表数据，未生成权限记录时从三个旧字段读取。
	var ledger, purchases, employees bool
	if err := s.DB.QueryRowContext(ctx, `SELECT can_ledger,can_purchases,can_employees FROM mini_users WHERE id=?`, id).Scan(&ledger, &purchases, &employees); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []string{}, nil
		}
		return nil, err
	}
	if ledger {
		permissions = append(permissions, "ledger")
	}
	if purchases {
		permissions = append(permissions, "purchases")
	}
	if employees {
		permissions = append(permissions, "employees")
	}
	return permissions, nil
}

// MiniActionPermissions 返回小程序页面隐藏新增、编辑、删除按钮所需的操作权限。
func (s *Store) MiniActionPermissions(ctx context.Context, id uint64) (map[string]MiniActionAccess, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT module_key,enabled,can_view,can_create,can_edit,can_delete FROM mini_user_permissions WHERE mini_user_id=? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]MiniActionAccess{}
	hasRows := false
	for rows.Next() {
		var key string
		var enabled bool
		var access MiniActionAccess
		if err := rows.Scan(&key, &enabled, &access.View, &access.Create, &access.Edit, &access.Delete); err != nil {
			return nil, err
		}
		hasRows = true
		if !enabled {
			access = MiniActionAccess{}
		}
		result[key] = access
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if hasRows {
		return result, nil
	}
	var ledger, purchases, employees bool
	if err := s.DB.QueryRowContext(ctx, `SELECT can_ledger,can_purchases,can_employees FROM mini_users WHERE id=?`, id).Scan(&ledger, &purchases, &employees); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, nil
		}
		return nil, err
	}
	for key, allowed := range map[string]bool{"ledger": ledger, "purchases": purchases, "employees": employees} {
		result[key] = MiniActionAccess{View: allowed, Create: allowed, Edit: allowed, Delete: allowed}
	}
	return result, nil
}

// MiniAnyPermission 用于选项接口，确保至少拥有一个业务模块权限。
func (s *Store) MiniAnyPermission(ctx context.Context, id uint64) (bool, error) {
	if err := s.ready(); err != nil {
		return false, err
	}
	var total, active int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(enabled=1 AND can_view=1),0) FROM mini_user_permissions WHERE mini_user_id=?`, id).Scan(&total, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if total > 0 {
		return active > 0, nil
	}
	var ledger, purchases, employees bool
	err = s.DB.QueryRowContext(ctx, `SELECT can_ledger,can_purchases,can_employees FROM mini_users WHERE id=? AND enabled=1`, id).Scan(&ledger, &purchases, &employees)
	return ledger || purchases || employees, err
}

type MiniOption struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Direction    string `json:"direction"`
	DepartmentID uint64 `json:"department_id"`
}

func (s *Store) MiniOptions(ctx context.Context) (map[string][]MiniOption, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	result := map[string][]MiniOption{}
	queries := map[string]string{
		"accounts":    `SELECT id,name,'',0 FROM ledger_accounts WHERE enabled=1 ORDER BY id`,
		"categories":  `SELECT id,name,direction,0 FROM ledger_categories WHERE enabled=1 ORDER BY sort_order,id`,
		"departments": `SELECT id,name,'',0 FROM departments WHERE enabled=1 AND active=1 ORDER BY sort_order,id`,
		"positions":   `SELECT p.id,p.name,'',p.department_id FROM positions p JOIN departments d ON d.id=p.department_id WHERE p.enabled=1 AND d.enabled=1 AND p.active=1 AND d.active=1 ORDER BY p.sort_order,p.id`,
	}
	for key, query := range queries {
		rows, err := s.DB.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		items := []MiniOption{}
		for rows.Next() {
			var item MiniOption
			if err := rows.Scan(&item.ID, &item.Name, &item.Direction, &item.DepartmentID); err != nil {
				rows.Close()
				return nil, err
			}
			items = append(items, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		result[key] = items
	}
	return result, nil
}

type MiniLedger struct {
	ID uint64 `json:"id"`
	request.LedgerInput
	Account        string             `json:"account"`
	Category       string             `json:"category"`
	Department     string             `json:"department"`
	PayrollBatchID uint64             `json:"payroll_batch_id"`
	Attachments    []LedgerAttachment `json:"attachments"`
}

const miniLedgerSelect = `SELECT l.id,DATE_FORMAT(l.occurred_on,'%Y-%m-%d'),l.account_id,COALESCE(l.category_id,0),COALESCE(l.department_id,0),l.direction,CAST(l.amount AS CHAR),l.summary,l.counterparty,l.voucher_no,l.remark,a.name,COALESCE(c.name,''),COALESCE(d.name,''),COALESCE(pb.id,COALESCE((SELECT pi.payroll_batch_id FROM payroll_items pi WHERE pi.ledger_entry_id=l.id LIMIT 1),0)) FROM ledger_entries l JOIN ledger_accounts a ON a.id=l.account_id LEFT JOIN ledger_categories c ON c.id=l.category_id LEFT JOIN departments d ON d.id=l.department_id LEFT JOIN payroll_batches pb ON pb.ledger_entry_id=l.id`

func scanMiniLedger(row scanner) (MiniLedger, error) {
	var v MiniLedger
	err := row.Scan(&v.ID, &v.OccurredOn, &v.AccountID, &v.CategoryID, &v.DepartmentID, &v.Direction, &v.Amount, &v.Summary, &v.Counterparty, &v.VoucherNo, &v.Remark, &v.Account, &v.Category, &v.Department, &v.PayrollBatchID)
	return v, err
}

func (s *Store) MiniLedger(ctx context.Context, id uint64) (MiniLedger, error) {
	if err := s.ready(); err != nil {
		return MiniLedger{}, err
	}
	v, err := scanMiniLedger(s.DB.QueryRowContext(ctx, miniLedgerSelect+` WHERE l.id=? AND l.active=1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	if err == nil {
		v.Attachments, err = s.LedgerAttachments(ctx, id)
	}
	return v, err
}

func (s *Store) MiniLedgerList(ctx context.Context, start, end, direction, search string, offset int) ([]MiniLedger, bool, error) {
	if err := s.ready(); err != nil {
		return nil, false, err
	}
	rows, err := s.DB.QueryContext(ctx, miniLedgerSelect+` WHERE l.active=1 AND l.occurred_on>=? AND l.occurred_on<DATE_ADD(?,INTERVAL 1 DAY) AND (?='' OR l.direction=?) AND (?='' OR LOCATE(?,l.summary)>0 OR LOCATE(?,l.counterparty)>0) ORDER BY l.occurred_on DESC,l.id DESC LIMIT 31 OFFSET ?`, start, end, direction, direction, search, search, search, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := []MiniLedger{}
	for rows.Next() {
		v, err := scanMiniLedger(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(items) > 30
	if more {
		items = items[:30]
	}
	return items, more, nil
}

func optionalID(id uint64) any {
	if id == 0 {
		return nil
	}
	return id
}
func optionalDate(date string) any {
	if date == "" {
		return nil
	}
	return date
}

func requireReference(ctx context.Context, tx *sql.Tx, query string, message string, args ...any) error {
	var id uint64
	err := tx.QueryRowContext(ctx, query, args...).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return MiniInputError{message}
	}
	return err
}

func miniAudit(ctx context.Context, tx *sql.Tx, actor uint64, module string, id uint64, title string) error {
	action, detail := "小程序保存", fmt.Sprintf("小程序用户ID：%d", actor)
	if actor == 0 {
		action, detail = "后台保存", "后台员工表单"
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_logs(module,action,target_type,target_id,target_text,detail) VALUES(?,?,?,?,?,?)`, module, action, module, id, title, detail)
	return err
}

func (s *Store) SaveMiniLedger(ctx context.Context, id, actor uint64, v request.LedgerInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if id != 0 {
		var locked uint64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM ledger_entries WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&locked); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrNotFound
			}
			return 0, err
		}
		var batch uint64
		err := tx.QueryRowContext(ctx, `SELECT id FROM payroll_batches WHERE ledger_entry_id=? LIMIT 1 FOR UPDATE`, id).Scan(&batch)
		if err == nil {
			return 0, MiniInputError{"工资发放产生的流水请在工资模块处理，不能直接修改"}
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		var payrollItem uint64
		err = tx.QueryRowContext(ctx, `SELECT id FROM payroll_items WHERE ledger_entry_id=? LIMIT 1 FOR UPDATE`, id).Scan(&payrollItem)
		if err == nil {
			return 0, MiniInputError{"工资发放产生的流水请在工资模块处理，不能直接修改"}
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
	}
	if err := requireReference(ctx, tx, `SELECT id FROM ledger_accounts WHERE id=? AND enabled=1 FOR SHARE`, "账户不存在或已停用", v.AccountID); err != nil {
		return 0, err
	}
	if v.CategoryID != 0 {
		if err := requireReference(ctx, tx, `SELECT id FROM ledger_categories WHERE id=? AND direction=? AND enabled=1 FOR SHARE`, "分类不存在、已停用或收支方向不匹配", v.CategoryID, v.Direction); err != nil {
			return 0, err
		}
	}
	if v.DepartmentID != 0 {
		if err := requireReference(ctx, tx, `SELECT id FROM departments WHERE id=? AND enabled=1 AND active=1 FOR SHARE`, "部门不存在或已停用", v.DepartmentID); err != nil {
			return 0, err
		}
	}
	args := []any{v.OccurredOn, v.AccountID, optionalID(v.CategoryID), optionalID(v.DepartmentID), v.Direction, v.Amount, v.Summary, v.Counterparty, v.VoucherNo, v.Remark}
	if id == 0 {
		result, e := tx.ExecContext(ctx, `INSERT INTO ledger_entries(occurred_on,account_id,category_id,department_id,direction,amount,summary,counterparty,voucher_no,remark) VALUES(?,?,?,?,?,?,?,?,?,?)`, args...)
		if e != nil {
			return 0, e
		}
		inserted, e := result.LastInsertId()
		if e != nil {
			return 0, e
		}
		id = uint64(inserted)
	} else {
		args = append(args, id)
		if _, err := tx.ExecContext(ctx, `UPDATE ledger_entries SET occurred_on=?,account_id=?,category_id=?,department_id=?,direction=?,amount=?,summary=?,counterparty=?,voucher_no=?,remark=? WHERE id=? AND active=1`, args...); err != nil {
			return 0, err
		}
	}
	if err := miniAudit(ctx, tx, actor, "ledger", id, v.Summary); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (s *Store) DeleteLedger(ctx context.Context, id, actor uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var summary string
	if err := tx.QueryRowContext(ctx, `SELECT summary FROM ledger_entries WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&summary); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	var payrollCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payroll_batches WHERE ledger_entry_id=?`, id).Scan(&payrollCount); err != nil {
		return err
	}
	var payrollItemCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payroll_items WHERE ledger_entry_id=?`, id).Scan(&payrollItemCount); err != nil {
		return err
	}
	if payrollCount > 0 || payrollItemCount > 0 {
		return MiniInputError{"工资发放产生的流水不能删除"}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE ledger_entries SET active=0,deleted_at=NOW(),deleted_by=? WHERE id=? AND active=1`, actor, id); err != nil {
		return err
	}
	if err := miniAudit(ctx, tx, actor, "ledger", id, "删除流水："+summary); err != nil {
		return err
	}
	return tx.Commit()
}

type MiniEmployee struct {
	ID uint64 `json:"id"`
	request.EmployeeInput
	Department                     string                        `json:"department"`
	Position                       string                        `json:"position"`
	HealthCertificateID            uint64                        `json:"health_certificate_id"`
	HealthCertificateURL           string                        `json:"health_certificate_url"`
	HealthCertificateIssuedOn      string                        `json:"health_certificate_issued_on"`
	HealthCertificateExpiresOn     string                        `json:"health_certificate_expires_on"`
	HealthCertificateStatus        string                        `json:"health_certificate_status"`
	HealthCertificateStatusClass   string                        `json:"health_certificate_status_class"`
	HealthCertificateDaysRemaining int                           `json:"health_certificate_days_remaining"`
	Attachments                    []response.EmployeeAttachment `json:"attachments"`
}

func (s *Store) MiniEmployee(ctx context.Context, id uint64) (MiniEmployee, error) {
	if err := s.ready(); err != nil {
		return MiniEmployee{}, err
	}
	var v MiniEmployee
	var days int
	err := s.DB.QueryRowContext(ctx, `SELECT e.id,e.employee_no,e.name,e.gender,e.id_card,e.mobile,COALESCE(e.department_id,0),COALESCE(e.position_id,0),e.employment_status,e.employment_type,COALESCE(e.pay_basis,'monthly'),COALESCE(DATE_FORMAT(e.joined_on,'%Y-%m-%d'),''),COALESCE(DATE_FORMAT(e.regularized_on,'%Y-%m-%d'),''),COALESCE(DATE_FORMAT(e.left_on,'%Y-%m-%d'),''),COALESCE(CAST(e.entry_salary AS CHAR),''),COALESCE(CAST(e.current_salary AS CHAR),''),e.education,e.hometown,e.remark,COALESCE(d.name,''),COALESCE(p.name,''),COALESCE(hc.id,0),COALESCE(hc.file_url,''),COALESCE(DATE_FORMAT(hc.issued_on,'%Y-%m-%d'),''),COALESCE(DATE_FORMAT(hc.expires_on,'%Y-%m-%d'),''),COALESCE(DATEDIFF(hc.expires_on,CURDATE()),0) FROM employees e LEFT JOIN departments d ON d.id=e.department_id LEFT JOIN positions p ON p.id=e.position_id LEFT JOIN employee_health_certificates hc ON hc.id=(SELECT current_hc.id FROM employee_health_certificates current_hc WHERE current_hc.employee_id=e.id AND current_hc.is_current=1 ORDER BY current_hc.expires_on DESC,current_hc.id DESC LIMIT 1) WHERE e.id=? AND e.active=1`, id).Scan(&v.ID, &v.EmployeeNo, &v.Name, &v.Gender, &v.IDCard, &v.Mobile, &v.DepartmentID, &v.PositionID, &v.EmploymentStatus, &v.EmploymentType, &v.PayBasis, &v.JoinedOn, &v.RegularizedOn, &v.LeftOn, &v.EntrySalary, &v.CurrentSalary, &v.Education, &v.Hometown, &v.Remark, &v.Department, &v.Position, &v.HealthCertificateID, &v.HealthCertificateURL, &v.HealthCertificateIssuedOn, &v.HealthCertificateExpiresOn, &days)
	v.HealthCertificateDaysRemaining = days
	v.HealthCertificateStatus, v.HealthCertificateStatusClass = healthCertificateStatus(v.HealthCertificateID, days)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	if err == nil {
		v.Attachments, err = s.EmployeeAttachments(ctx, id)
	}
	return v, err
}

func (s *Store) MiniEmployeeList(ctx context.Context, search, status string, offset int) ([]response.Employee, bool, error) {
	if err := s.ready(); err != nil {
		return nil, false, err
	}
	rows, err := s.DB.QueryContext(ctx, employeeSelect+` WHERE e.active=1 AND (?='' OR LOCATE(?,e.name)>0 OR LOCATE(?,e.employee_no)>0 OR LOCATE(?,e.mobile)>0) AND (?='' OR e.employment_status=?) ORDER BY e.employment_status='left',e.employee_no,e.id LIMIT 31 OFFSET ?`, search, search, search, search, status, status, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := []response.Employee{}
	for rows.Next() {
		v, err := scanEmployee(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(items) > 30
	if more {
		items = items[:30]
	}
	return items, more, nil
}

func (s *Store) SaveEmployee(ctx context.Context, id, actor uint64, v request.EmployeeInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if id != 0 {
		var locked uint64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM employees WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&locked); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrNotFound
			}
			return 0, err
		}
	}
	if v.DepartmentID != 0 {
		if err := requireReference(ctx, tx, `SELECT id FROM departments WHERE id=? AND enabled=1 AND active=1 FOR SHARE`, "部门不存在或已停用", v.DepartmentID); err != nil {
			return 0, err
		}
	}
	if v.PositionID != 0 {
		if err := requireReference(ctx, tx, `SELECT id FROM positions WHERE id=? AND department_id=? AND enabled=1 AND active=1 FOR SHARE`, "岗位与所选部门不匹配", v.PositionID, v.DepartmentID); err != nil {
			return 0, err
		}
	}
	entrySalary := v.EntrySalary
	if entrySalary == "" {
		entrySalary = v.CurrentSalary
	}
	args := []any{v.EmployeeNo, v.Name, v.Gender, v.IDCard, v.Mobile, optionalID(v.DepartmentID), optionalID(v.PositionID), v.EmploymentStatus, v.EmploymentType, optionalDate(v.JoinedOn), optionalDate(v.RegularizedOn), optionalDate(v.LeftOn), optionalMoney(entrySalary), optionalMoney(v.CurrentSalary), v.PayBasis, v.Education, v.Hometown, v.Remark}
	if id == 0 {
		result, e := tx.ExecContext(ctx, `INSERT INTO employees(employee_no,name,gender,id_card,mobile,department_id,position_id,employment_status,employment_type,joined_on,regularized_on,left_on,entry_salary,current_salary,pay_basis,education,hometown,remark) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, args...)
		if e != nil {
			return 0, e
		}
		inserted, e := result.LastInsertId()
		if e != nil {
			return 0, e
		}
		id = uint64(inserted)
	} else {
		args = append(args, id)
		if _, err := tx.ExecContext(ctx, `UPDATE employees SET employee_no=?,name=?,gender=?,id_card=?,mobile=?,department_id=?,position_id=?,employment_status=?,employment_type=?,joined_on=?,regularized_on=?,left_on=?,entry_salary=?,current_salary=?,pay_basis=?,education=?,hometown=?,remark=? WHERE id=? AND active=1`, args...); err != nil {
			return 0, err
		}
	}
	if err := miniAudit(ctx, tx, actor, "employees", id, v.Name); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (s *Store) DeleteEmployee(ctx context.Context, id, actor uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var name string
	if err := tx.QueryRowContext(ctx, `SELECT name FROM employees WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE mini_users SET enabled=0 WHERE employee_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE employees SET active=0,deleted_at=NOW(),deleted_by=? WHERE id=? AND active=1`, actor, id); err != nil {
		return err
	}
	if err := miniAudit(ctx, tx, actor, "employees", id, "删除员工："+name); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateMiniOption 支持空数据库首次建账，不自动插入业务数据。
func (s *Store) CreateMiniOption(ctx context.Context, kind, name, direction string, departmentID, actor uint64) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return 0, err
	}
	no := "M" + hex.EncodeToString(b)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var result sql.Result
	switch kind {
	case "accounts":
		result, err = tx.ExecContext(ctx, `INSERT INTO ledger_accounts(account_no,name,account_type) VALUES(?,?,'other')`, no, name)
	case "categories":
		result, err = tx.ExecContext(ctx, `INSERT INTO ledger_categories(name,direction) VALUES(?,?)`, name, direction)
	case "departments":
		result, err = tx.ExecContext(ctx, `INSERT INTO departments(department_no,name) VALUES(?,?)`, no, name)
	case "positions":
		if err := requireReference(ctx, tx, `SELECT id FROM departments WHERE id=? AND enabled=1 AND active=1 FOR SHARE`, "请先选择有效部门", departmentID); err != nil {
			return 0, err
		}
		result, err = tx.ExecContext(ctx, `INSERT INTO positions(position_no,name,department_id) VALUES(?,?,?)`, no, name, departmentID)
	default:
		return 0, MiniInputError{"不支持的选项类型"}
	}
	if err != nil {
		return 0, err
	}
	inserted, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := miniAudit(ctx, tx, actor, kind, uint64(inserted), name); err != nil {
		return 0, err
	}
	return uint64(inserted), tx.Commit()
}
