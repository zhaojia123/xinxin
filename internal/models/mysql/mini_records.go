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
		"departments": `SELECT id,name,'',0 FROM departments WHERE enabled=1 ORDER BY sort_order,id`,
		"positions":   `SELECT p.id,p.name,'',p.department_id FROM positions p JOIN departments d ON d.id=p.department_id WHERE p.enabled=1 AND d.enabled=1 ORDER BY p.sort_order,p.id`,
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
	Account        string `json:"account"`
	Category       string `json:"category"`
	Department     string `json:"department"`
	PayrollBatchID uint64 `json:"payroll_batch_id"`
}

const miniLedgerSelect = `SELECT l.id,DATE_FORMAT(l.occurred_on,'%Y-%m-%d'),l.account_id,COALESCE(l.category_id,0),COALESCE(l.department_id,0),l.direction,CAST(l.amount AS CHAR),l.summary,l.counterparty,l.voucher_no,l.remark,a.name,COALESCE(c.name,''),COALESCE(d.name,''),COALESCE(pb.id,0) FROM ledger_entries l JOIN ledger_accounts a ON a.id=l.account_id LEFT JOIN ledger_categories c ON c.id=l.category_id LEFT JOIN departments d ON d.id=l.department_id LEFT JOIN payroll_batches pb ON pb.ledger_entry_id=l.id`

func scanMiniLedger(row scanner) (MiniLedger, error) {
	var v MiniLedger
	err := row.Scan(&v.ID, &v.OccurredOn, &v.AccountID, &v.CategoryID, &v.DepartmentID, &v.Direction, &v.Amount, &v.Summary, &v.Counterparty, &v.VoucherNo, &v.Remark, &v.Account, &v.Category, &v.Department, &v.PayrollBatchID)
	return v, err
}

func (s *Store) MiniLedger(ctx context.Context, id uint64) (MiniLedger, error) {
	if err := s.ready(); err != nil {
		return MiniLedger{}, err
	}
	v, err := scanMiniLedger(s.DB.QueryRowContext(ctx, miniLedgerSelect+` WHERE l.id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}

func (s *Store) MiniLedgerList(ctx context.Context, month, direction, search string, offset int) ([]MiniLedger, bool, error) {
	if err := s.ready(); err != nil {
		return nil, false, err
	}
	rows, err := s.DB.QueryContext(ctx, miniLedgerSelect+` WHERE l.occurred_on>=? AND l.occurred_on<DATE_ADD(?,INTERVAL 1 MONTH) AND (?='' OR l.direction=?) AND (?='' OR LOCATE(?,l.summary)>0 OR LOCATE(?,l.counterparty)>0) ORDER BY l.occurred_on DESC,l.id DESC LIMIT 31 OFFSET ?`, month+"-01", month+"-01", direction, direction, search, search, search, offset)
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
		if err := tx.QueryRowContext(ctx, `SELECT id FROM ledger_entries WHERE id=? FOR UPDATE`, id).Scan(&locked); err != nil {
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
		if err := requireReference(ctx, tx, `SELECT id FROM departments WHERE id=? AND enabled=1 FOR SHARE`, "部门不存在或已停用", v.DepartmentID); err != nil {
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
		if _, err := tx.ExecContext(ctx, `UPDATE ledger_entries SET occurred_on=?,account_id=?,category_id=?,department_id=?,direction=?,amount=?,summary=?,counterparty=?,voucher_no=?,remark=? WHERE id=?`, args...); err != nil {
			return 0, err
		}
	}
	if err := miniAudit(ctx, tx, actor, "ledger", id, v.Summary); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

type MiniEmployee struct {
	ID uint64 `json:"id"`
	request.EmployeeInput
}

func (s *Store) MiniEmployee(ctx context.Context, id uint64) (MiniEmployee, error) {
	if err := s.ready(); err != nil {
		return MiniEmployee{}, err
	}
	var v MiniEmployee
	err := s.DB.QueryRowContext(ctx, `SELECT id,employee_no,name,gender,id_card,mobile,COALESCE(department_id,0),COALESCE(position_id,0),employment_status,employment_type,COALESCE(DATE_FORMAT(joined_on,'%Y-%m-%d'),''),COALESCE(DATE_FORMAT(regularized_on,'%Y-%m-%d'),''),COALESCE(DATE_FORMAT(left_on,'%Y-%m-%d'),''),CAST(current_salary AS CHAR),education,hometown,remark FROM employees WHERE id=?`, id).Scan(&v.ID, &v.EmployeeNo, &v.Name, &v.Gender, &v.IDCard, &v.Mobile, &v.DepartmentID, &v.PositionID, &v.EmploymentStatus, &v.EmploymentType, &v.JoinedOn, &v.RegularizedOn, &v.LeftOn, &v.CurrentSalary, &v.Education, &v.Hometown, &v.Remark)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}

func (s *Store) MiniEmployeeList(ctx context.Context, search, status string, offset int) ([]response.Employee, bool, error) {
	if err := s.ready(); err != nil {
		return nil, false, err
	}
	rows, err := s.DB.QueryContext(ctx, employeeSelect+` WHERE (?='' OR LOCATE(?,e.name)>0 OR LOCATE(?,e.employee_no)>0 OR LOCATE(?,e.mobile)>0) AND (?='' OR e.employment_status=?) ORDER BY e.employment_status='left',e.employee_no,e.id LIMIT 31 OFFSET ?`, search, search, search, search, status, status, offset)
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
		if err := tx.QueryRowContext(ctx, `SELECT id FROM employees WHERE id=? FOR UPDATE`, id).Scan(&locked); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrNotFound
			}
			return 0, err
		}
	}
	if v.DepartmentID != 0 {
		if err := requireReference(ctx, tx, `SELECT id FROM departments WHERE id=? AND enabled=1 FOR SHARE`, "部门不存在或已停用", v.DepartmentID); err != nil {
			return 0, err
		}
	}
	if v.PositionID != 0 {
		if err := requireReference(ctx, tx, `SELECT id FROM positions WHERE id=? AND department_id=? AND enabled=1 FOR SHARE`, "岗位与所选部门不匹配", v.PositionID, v.DepartmentID); err != nil {
			return 0, err
		}
	}
	args := []any{v.EmployeeNo, v.Name, v.Gender, v.IDCard, v.Mobile, optionalID(v.DepartmentID), optionalID(v.PositionID), v.EmploymentStatus, v.EmploymentType, optionalDate(v.JoinedOn), optionalDate(v.RegularizedOn), optionalDate(v.LeftOn), v.CurrentSalary, v.Education, v.Hometown, v.Remark}
	if id == 0 {
		result, e := tx.ExecContext(ctx, `INSERT INTO employees(employee_no,name,gender,id_card,mobile,department_id,position_id,employment_status,employment_type,joined_on,regularized_on,left_on,current_salary,education,hometown,remark) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, args...)
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
		if _, err := tx.ExecContext(ctx, `UPDATE employees SET employee_no=?,name=?,gender=?,id_card=?,mobile=?,department_id=?,position_id=?,employment_status=?,employment_type=?,joined_on=?,regularized_on=?,left_on=?,current_salary=?,education=?,hometown=?,remark=? WHERE id=?`, args...); err != nil {
			return 0, err
		}
	}
	if err := miniAudit(ctx, tx, actor, "employees", id, v.Name); err != nil {
		return 0, err
	}
	return id, tx.Commit()
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
		if err := requireReference(ctx, tx, `SELECT id FROM departments WHERE id=? AND enabled=1 FOR SHARE`, "请先选择有效部门", departmentID); err != nil {
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
