package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/apperror"
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
	AccountType  string `json:"account_type,omitempty"`
	Remark       string `json:"remark,omitempty"`
	DisplayName  string `json:"display_name"`
}

func (s *Store) MiniOptions(ctx context.Context) (map[string][]MiniOption, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	result := map[string][]MiniOption{}
	queries := map[string]string{
		// 系统经营流水账户仅供自动入账使用，手工台账不显示，避免误选为对方资金账户。
		"accounts":    `SELECT id,name,'',0,account_type,remark FROM ledger_accounts WHERE enabled=1 AND account_no<>'SYS-OPERATING' ORDER BY id`,
		"categories":  `SELECT id,name,direction,0,'','' FROM ledger_categories WHERE enabled=1 ORDER BY sort_order,id`,
		"departments": `SELECT id,name,'',0,'','' FROM departments WHERE enabled=1 AND active=1 ORDER BY sort_order,id`,
		"positions":   `SELECT p.id,p.name,'',p.department_id,'','' FROM positions p JOIN departments d ON d.id=p.department_id WHERE p.enabled=1 AND d.enabled=1 AND p.active=1 AND d.active=1 ORDER BY p.sort_order,p.id`,
	}
	for key, query := range queries {
		rows, err := s.DB.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		items := []MiniOption{}
		for rows.Next() {
			var item MiniOption
			if err := rows.Scan(&item.ID, &item.Name, &item.Direction, &item.DepartmentID, &item.AccountType, &item.Remark); err != nil {
				rows.Close()
				return nil, err
			}
			item.DisplayName = item.Name
			if key == "accounts" {
				labels := map[string]string{"bank": "银行卡", "wechat": "微信", "alipay": "支付宝", "qq": "QQ", "cash": "现金", "other": "其他"}
				if label := labels[item.AccountType]; label != "" {
					item.DisplayName += " · " + label
				}
				if item.Remark != "" {
					item.DisplayName += " · " + item.Remark
				}
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
	Seq int    `json:"seq,omitempty"`
	ID  uint64 `json:"id"`
	request.LedgerInput
	Account        string              `json:"account"`
	Category       string              `json:"category"`
	Department     string              `json:"department"`
	PayrollBatchID uint64              `json:"payroll_batch_id"`
	PayrollType    string              `json:"payroll_type"`
	Attachments    []LedgerAttachment  `json:"attachments"`
	PayrollItems   []PayrollLedgerItem `json:"payroll_items"`
}

type PayrollLedgerItem struct {
	EmployeeName string `json:"employee_name"`
	Amount       string `json:"amount"`
}

const miniLedgerSelect = `SELECT l.id,DATE_FORMAT(l.occurred_on,'%Y-%m-%d'),l.account_id,COALESCE(l.category_id,0),COALESCE(l.department_id,0),l.direction,CAST(l.amount AS CHAR),l.summary,l.counterparty,l.voucher_no,l.remark,a.name,COALESCE(c.name,''),COALESCE(d.name,''),COALESCE(pb.id,COALESCE((SELECT pi.payroll_batch_id FROM payroll_items pi WHERE pi.ledger_entry_id=l.id LIMIT 1),0)),COALESCE((SELECT pse.payroll_type FROM payroll_settlement_entries pse WHERE pse.ledger_entry_id=l.id LIMIT 1),'') FROM ledger_entries l JOIN ledger_accounts a ON a.id=l.account_id LEFT JOIN ledger_categories c ON c.id=l.category_id LEFT JOIN departments d ON d.id=l.department_id LEFT JOIN payroll_batches pb ON pb.ledger_entry_id=l.id`

func scanMiniLedger(row scanner) (MiniLedger, error) {
	var v MiniLedger
	err := row.Scan(&v.ID, &v.OccurredOn, &v.AccountID, &v.CategoryID, &v.DepartmentID, &v.Direction, &v.Amount, &v.Summary, &v.Counterparty, &v.VoucherNo, &v.Remark, &v.Account, &v.Category, &v.Department, &v.PayrollBatchID, &v.PayrollType)
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
		if err == nil {
			v.PayrollItems, err = s.PayrollLedgerItems(ctx, id)
		}
	}
	return v, err
}

func (s *Store) PayrollLedgerItems(ctx context.Context, ledgerID uint64) ([]PayrollLedgerItem, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT employee_name,amount FROM (
		SELECT e.name AS employee_name,CAST(i.net_salary AS CHAR) AS amount,e.employee_no AS sort_no,i.id AS sort_id FROM payroll_items i JOIN employees e ON e.id=i.employee_id WHERE i.ledger_entry_id=?
		UNION ALL
		SELECT s.employee_name,CAST(s.amount AS CHAR),'',s.id FROM special_payroll_records s WHERE s.ledger_entry_id=?
	) payroll_details ORDER BY sort_no,sort_id`, ledgerID, ledgerID)
	if err != nil {
		return nil, apperror.Wrap(err, "读取工资台账明细失败")
	}
	defer rows.Close()
	items := make([]PayrollLedgerItem, 0)
	for rows.Next() {
		var item PayrollLedgerItem
		if err := rows.Scan(&item.EmployeeName, &item.Amount); err != nil {
			return nil, apperror.Wrap(err, "读取工资台账明细失败")
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历工资台账明细失败")
	}
	return items, nil
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
		v.Seq = len(items) + 1 + offset
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
		var settlement uint64
		err = tx.QueryRowContext(ctx, `SELECT id FROM payroll_settlement_entries WHERE ledger_entry_id=? LIMIT 1 FOR UPDATE`, id).Scan(&settlement)
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
	var payrollSettlementCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payroll_settlement_entries WHERE ledger_entry_id=?`, id).Scan(&payrollSettlementCount); err != nil {
		return err
	}
	if payrollCount > 0 || payrollItemCount > 0 || payrollSettlementCount > 0 {
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
	err := s.DB.QueryRowContext(ctx, `SELECT e.id,e.employee_no,e.name,e.gender,e.id_card,e.mobile,COALESCE(e.department_id,0),COALESCE(e.position_id,0),e.employment_status,e.employment_type,COALESCE(e.pay_basis,'monthly'),COALESCE(e.monthly_rest_days,0),COALESCE(DATE_FORMAT(e.joined_on,'%Y-%m-%d'),''),COALESCE(DATE_FORMAT(e.regularized_on,'%Y-%m-%d'),''),COALESCE(DATE_FORMAT(e.left_on,'%Y-%m-%d'),''),COALESCE(CAST(e.entry_salary AS CHAR),''),COALESCE(CAST(COALESCE(e.current_salary,e.entry_salary) AS CHAR),''),e.education,e.hometown,e.remark,COALESCE(d.name,''),COALESCE(p.name,''),COALESCE(hc.id,0),COALESCE(hc.file_url,''),COALESCE(DATE_FORMAT(hc.issued_on,'%Y-%m-%d'),''),COALESCE(DATE_FORMAT(hc.expires_on,'%Y-%m-%d'),''),COALESCE(DATEDIFF(hc.expires_on,CURDATE()),0) FROM employees e LEFT JOIN departments d ON d.id=e.department_id LEFT JOIN positions p ON p.id=e.position_id LEFT JOIN employee_health_certificates hc ON hc.id=(SELECT current_hc.id FROM employee_health_certificates current_hc WHERE current_hc.employee_id=e.id AND current_hc.is_current=1 ORDER BY current_hc.expires_on DESC,current_hc.id DESC LIMIT 1) WHERE e.id=? AND e.active=1`, id).Scan(&v.ID, &v.EmployeeNo, &v.Name, &v.Gender, &v.IDCard, &v.Mobile, &v.DepartmentID, &v.PositionID, &v.EmploymentStatus, &v.EmploymentType, &v.PayBasis, &v.MonthlyRestDays, &v.JoinedOn, &v.RegularizedOn, &v.LeftOn, &v.EntrySalary, &v.CurrentSalary, &v.Education, &v.Hometown, &v.Remark, &v.Department, &v.Position, &v.HealthCertificateID, &v.HealthCertificateURL, &v.HealthCertificateIssuedOn, &v.HealthCertificateExpiresOn, &days)
	v.HealthCertificateDaysRemaining = days
	v.Gender = request.NormalizeGender(v.Gender)
	v.HealthCertificateStatus, v.HealthCertificateStatusClass = healthCertificateStatus(v.HealthCertificateID, days)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	if err == nil {
		v.Attachments, err = s.EmployeeAttachments(ctx, id)
	}
	return v, err
}

func (s *Store) MiniEmployeeList(ctx context.Context, search, status, employmentType string, offset int) ([]response.Employee, bool, error) {
	if err := s.ready(); err != nil {
		return nil, false, err
	}
	rows, err := s.DB.QueryContext(ctx, employeeSelect+` WHERE e.active=1 AND (?='' OR LOCATE(?,e.name)>0 OR LOCATE(?,e.employee_no)>0 OR LOCATE(?,e.mobile)>0) AND (?='' OR e.employment_status=?) AND (?='' OR e.employment_type=?) ORDER BY e.employment_status='left',e.employee_no,e.id LIMIT 31 OFFSET ?`, search, search, search, search, status, status, employmentType, employmentType, offset)
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
		v.Seq = len(items) + 1 + offset
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
	// 新增员工时两项工资默认一致，避免只填写入职工资导致正式工工资无法计算。
	currentSalary := v.CurrentSalary
	if currentSalary == "" {
		currentSalary = entrySalary
	}
	var restDays any
	if v.MonthlyRestDays > 0 {
		restDays = v.MonthlyRestDays
	}
	args := []any{v.EmployeeNo, v.Name, v.Gender, v.IDCard, v.Mobile, optionalID(v.DepartmentID), optionalID(v.PositionID), v.EmploymentStatus, v.EmploymentType, optionalDate(v.JoinedOn), optionalDate(v.RegularizedOn), optionalDate(v.LeftOn), optionalMoney(entrySalary), optionalMoney(currentSalary), v.PayBasis, restDays, v.Education, v.Hometown, v.Remark}
	if id == 0 {
		result, e := tx.ExecContext(ctx, `INSERT INTO employees(employee_no,name,gender,id_card,mobile,department_id,position_id,employment_status,employment_type,joined_on,regularized_on,left_on,entry_salary,current_salary,pay_basis,monthly_rest_days,education,hometown,remark) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, args...)
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
		if _, err := tx.ExecContext(ctx, `UPDATE employees SET employee_no=?,name=?,gender=?,id_card=?,mobile=?,department_id=?,position_id=?,employment_status=?,employment_type=?,joined_on=?,regularized_on=?,left_on=?,entry_salary=?,current_salary=?,pay_basis=?,monthly_rest_days=?,education=?,hometown=?,remark=? WHERE id=? AND active=1`, args...); err != nil {
			return 0, err
		}
	}
	if err := miniAudit(ctx, tx, actor, "employees", id, v.Name); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// UpdateEmployeeField 复用完整员工校验，避免列表内联编辑绕过日期、工资和状态校验。
func (s *Store) UpdateEmployeeField(ctx context.Context, id, actor uint64, field, value string) error {
	item, err := s.MiniEmployee(ctx, id)
	if err != nil {
		return err
	}
	v := item.EmployeeInput
	value = strings.TrimSpace(value)
	switch field {
	case "mobile":
		v.Mobile = value
	case "joined_on":
		v.JoinedOn = value
	case "left_on":
		v.LeftOn = value
		if value != "" {
			v.EmploymentStatus = "left"
		} else if v.EmploymentStatus == "left" {
			v.EmploymentStatus = "active"
		}
	case "entry_salary":
		v.EntrySalary = value
	case "current_salary":
		v.CurrentSalary = value
	case "pay_basis":
		v.PayBasis = value
	case "employment_status":
		v.EmploymentStatus = value
	default:
		return MiniInputError{"该员工字段不支持直接修改"}
	}
	if err := v.Validate(); err != nil {
		return MiniInputError{err.Error()}
	}
	_, err = s.SaveEmployee(ctx, id, actor, v)
	return err
}

func (s *Store) DeleteEmployee(ctx context.Context, id, actor uint64) error {
	_, err := s.DeleteEmployees(ctx, []uint64{id}, actor)
	return err
}

// DeleteEmployees 软删除多个员工，保留工资和台账等历史关联数据。
func (s *Store) DeleteEmployees(ctx context.Context, ids []uint64, actor uint64) (int, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	seen := make(map[uint64]struct{}, len(ids))
	validIDs := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			validIDs = append(validIDs, id)
		}
	}
	if len(validIDs) == 0 {
		return 0, MiniInputError{"请选择要删除的员工"}
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, id := range validIDs {
		var name string
		if err := tx.QueryRowContext(ctx, `SELECT name FROM employees WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&name); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrNotFound
			}
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE mini_users SET enabled=0 WHERE employee_id=?`, id); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE employees SET active=0,deleted_at=NOW(),deleted_by=? WHERE id=? AND active=1`, actor, id); err != nil {
			return 0, err
		}
		if err := miniAudit(ctx, tx, actor, "employees", id, "删除员工："+name); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(validIDs), nil
}

// CreateMiniOption 支持空数据库首次建账，不自动插入业务数据。
func (s *Store) CreateMiniOption(ctx context.Context, kind, name, direction, accountType, remark string, departmentID, actor uint64) (uint64, error) {
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
		result, err = tx.ExecContext(ctx, `INSERT INTO ledger_accounts(account_no,name,account_type,remark) VALUES(?,?,?,?)`, no, name, accountType, remark)
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
