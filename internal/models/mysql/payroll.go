package mysql

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/apperror"
)

func NormalizeMonth(value string) string {
	if _, err := time.Parse("2006-01", value); err == nil {
		return value
	}
	return time.Now().Format("2006-01")
}

func (s *Store) Payroll(ctx context.Context, month string) ([]response.PayrollRecord, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	month = NormalizeMonth(month)
	rows, err := s.DB.QueryContext(ctx, `SELECT i.id,b.id,DATE_FORMAT(b.payroll_month,'%Y-%m'),e.employee_no,e.name,i.base_salary,i.bonus_amount,i.attendance_deduction,i.other_deduction,i.social_security,i.tax_amount,i.net_salary,i.status FROM payroll_items i JOIN payroll_batches b ON b.id=i.payroll_batch_id JOIN employees e ON e.id=i.employee_id WHERE b.payroll_month=? ORDER BY e.employee_no`, month+"-01")
	if err != nil {
		return nil, apperror.Wrap(err, "查询月度工资失败")
	}
	defer rows.Close()
	result := make([]response.PayrollRecord, 0)
	for rows.Next() {
		var v response.PayrollRecord
		var base, bonus, attendance, other, social, tax, net float64
		var state string
		if err := rows.Scan(&v.ID, &v.BatchID, &v.Month, &v.EmployeeNo, &v.Name, &base, &bonus, &attendance, &other, &social, &tax, &net, &state); err != nil {
			return nil, apperror.Wrap(err, "读取月度工资失败")
		}
		v.BaseSalary, v.Bonus, v.AttendanceDeduction, v.OtherDeduction, v.Deduction = Money(base), Money(bonus), Money(attendance), Money(other), Money(attendance+other)
		v.SocialSecurity, v.Tax, v.NetSalary = Money(social), Money(tax), Money(net)
		v.BaseSalaryValue, v.BonusValue, v.AttendanceDeductionValue, v.OtherDeductionValue, v.SocialSecurityValue, v.TaxValue = Decimal(base), Decimal(bonus), Decimal(attendance), Decimal(other), Decimal(social), Decimal(tax)
		v.Status, v.StatusClass = PayrollStatus(state)
		result = append(result, v)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历月度工资失败")
	}
	return result, nil
}

func (s *Store) PayrollSummary(ctx context.Context, month string) (response.PayrollSummary, error) {
	if err := s.ready(); err != nil {
		return response.PayrollSummary{}, err
	}
	month = NormalizeMonth(month)
	var v response.PayrollSummary
	var gross, deductions, net float64
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(i.base_salary+i.bonus_amount),0),COALESCE(SUM(i.attendance_deduction+i.other_deduction+i.social_security+i.tax_amount),0),COALESCE(SUM(i.net_salary),0),COUNT(i.id),COALESCE(SUM(i.status='paid'),0) FROM payroll_batches b LEFT JOIN payroll_items i ON i.payroll_batch_id=b.id WHERE b.payroll_month=?`, month+"-01").Scan(&gross, &deductions, &net, &v.EmployeeCount, &v.PaidCount)
	if err != nil {
		return response.PayrollSummary{}, apperror.Wrap(err, "统计月度工资失败")
	}
	v.Month, v.GrossAmount, v.DeductionAmount, v.NetAmount = month, Money(gross), Money(deductions), Money(net)
	err = s.DB.QueryRowContext(ctx, `SELECT id,status FROM payroll_batches WHERE payroll_month=?`, month+"-01").Scan(&v.BatchID, &v.BatchStatus)
	if errors.Is(err, sql.ErrNoRows) {
		v.BatchStatus = "not_generated"
	} else if err != nil {
		return response.PayrollSummary{}, apperror.Wrap(err, "读取工资批次状态失败")
	}
	return v, nil
}

func (s *Store) ConfirmPayroll(ctx context.Context, batchID uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始确认工资事务失败")
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM payroll_batches WHERE id=? FOR UPDATE`, batchID).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return apperror.Wrap(err, "读取工资批次失败")
	}
	if status == "paid" {
		return apperror.New("工资已经发放，不能重复确认")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE payroll_batches SET status='confirmed',confirmed_at=NOW() WHERE id=?`, batchID); err != nil {
		return apperror.Wrap(err, "确认工资批次失败")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE payroll_items SET status='confirmed' WHERE payroll_batch_id=? AND status<>'paid'`, batchID); err != nil {
		return apperror.Wrap(err, "确认工资明细失败")
	}
	return tx.Commit()
}

func validatePayroll(v request.PayrollUpdate) error {
	for _, n := range []float64{v.BaseSalary, v.Bonus, v.AttendanceDeduction, v.OtherDeduction, v.SocialSecurity, v.Tax} {
		if n < 0 || math.IsNaN(n) || math.IsInf(n, 0) {
			return apperror.New("工资金额必须是大于等于0的有效数字")
		}
	}
	return nil
}

func (s *Store) UpdatePayroll(ctx context.Context, id uint64, v request.PayrollUpdate) (float64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if err := validatePayroll(v); err != nil {
		return 0, err
	}
	net := v.BaseSalary + v.Bonus - v.AttendanceDeduction - v.OtherDeduction - v.SocialSecurity - v.Tax
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存工资事务失败")
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE payroll_items SET base_salary=?,bonus_amount=?,attendance_deduction=?,other_deduction=?,social_security=?,tax_amount=?,net_salary=? WHERE id=? AND status<>'paid'`, v.BaseSalary, v.Bonus, v.AttendanceDeduction, v.OtherDeduction, v.SocialSecurity, v.Tax, net, id)
	if err != nil {
		return 0, apperror.Wrap(err, "保存工资明细失败")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, apperror.Wrap(err, "读取工资保存结果失败")
	}
	if affected == 0 {
		return 0, ErrNotFound
	}
	if err := refreshBatch(ctx, tx, id); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交工资保存事务失败")
	}
	return net, nil
}

func refreshBatch(ctx context.Context, tx *sql.Tx, itemID uint64) error {
	_, err := tx.ExecContext(ctx, `UPDATE payroll_batches b SET employee_count=(SELECT COUNT(*) FROM payroll_items i WHERE i.payroll_batch_id=b.id),gross_amount=(SELECT COALESCE(SUM(base_salary+bonus_amount),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id),deduction_amount=(SELECT COALESCE(SUM(attendance_deduction+other_deduction+social_security+tax_amount),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id),net_amount=(SELECT COALESCE(SUM(net_salary),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id) WHERE b.id=(SELECT payroll_batch_id FROM payroll_items WHERE id=?)`, itemID)
	if err != nil {
		return apperror.Wrap(err, "同步工资批次汇总失败")
	}
	return nil
}

func (s *Store) GeneratePayroll(ctx context.Context, month string) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	month = NormalizeMonth(month)
	start, err := time.Parse("2006-01-02", month+"-01")
	if err != nil {
		return 0, apperror.Wrap(err, "工资月份格式不正确")
	}
	end := start.AddDate(0, 1, 0)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始生成工资事务失败")
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO payroll_batches(payroll_month) VALUES(?) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)`, start)
	if err != nil {
		return 0, apperror.Wrap(err, "创建工资批次失败")
	}
	batchID, err := result.LastInsertId()
	if err != nil {
		return 0, apperror.Wrap(err, "读取工资批次ID失败")
	}
	_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO payroll_items(payroll_batch_id,employee_id,base_salary,net_salary) SELECT ?,id,current_salary,current_salary FROM employees WHERE employment_status IN ('active','probation') AND active=1 AND (joined_on IS NULL OR joined_on<?) AND (left_on IS NULL OR left_on>=?)`, batchID, end, start)
	if err != nil {
		return 0, apperror.Wrap(err, "生成员工工资明细失败")
	}
	_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO payroll_adjustments(payroll_item_id,attendance_record_id,adjustment_type,amount,description) SELECT i.id,a.id,'deduction',0,CONCAT('待人工核算：',a.record_type,' ',IF(a.category='leave',CONCAT(a.duration_days,'天'),CONCAT(a.duration_minutes,'分钟'))) FROM payroll_items i JOIN attendance_records a ON a.employee_id=i.employee_id AND a.occurred_on>=? AND a.occurred_on<? AND a.status IN ('approved','recorded','confirmed') AND a.active=1 WHERE i.payroll_batch_id=?`, start, end, batchID)
	if err != nil {
		return 0, apperror.Wrap(err, "关联请假与异常记录失败")
	}
	var firstItem uint64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MIN(id),0) FROM payroll_items WHERE payroll_batch_id=?`, batchID).Scan(&firstItem); err != nil {
		return 0, apperror.Wrap(err, "读取工资明细失败")
	}
	if firstItem > 0 {
		if err := refreshBatch(ctx, tx, firstItem); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交生成工资事务失败")
	}
	return uint64(batchID), nil
}

func (s *Store) PayPayroll(ctx context.Context, batchID, accountID uint64, occurredOn string) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	date, err := time.Parse("2006-01-02", occurredOn)
	if err != nil {
		return 0, apperror.New("发放日期格式必须是 YYYY-MM-DD")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始工资发放事务失败")
	}
	defer tx.Rollback()
	var month, status string
	var amount float64
	var linked sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT DATE_FORMAT(payroll_month,'%Y-%m'),status,net_amount,ledger_entry_id FROM payroll_batches WHERE id=? FOR UPDATE`, batchID).Scan(&month, &status, &amount, &linked)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, apperror.Wrap(err, "读取工资批次失败")
	}
	if linked.Valid || status == "paid" {
		return uint64(linked.Int64), nil
	}
	if status != "confirmed" {
		return 0, apperror.New("请先确认本月工资，再执行发放")
	}
	if amount <= 0 {
		return 0, apperror.New("工资实发合计必须大于0")
	}
	var categoryID sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT id FROM ledger_categories WHERE direction='expense' AND name='工资薪酬' AND enabled=1 LIMIT 1`).Scan(&categoryID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, apperror.Wrap(err, "读取工资台账分类失败")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries(occurred_on,account_id,category_id,direction,amount,summary,counterparty) VALUES(?,?,?,'expense',?,?,?)`, date, accountID, categoryID, amount, month+"员工工资", "员工")
	if err != nil {
		return 0, apperror.Wrap(err, "写入工资台账支出失败")
	}
	entryID, err := result.LastInsertId()
	if err != nil {
		return 0, apperror.Wrap(err, "读取工资台账ID失败")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE payroll_batches SET status='paid',paid_at=NOW(),ledger_entry_id=? WHERE id=?`, entryID, batchID); err != nil {
		return 0, apperror.Wrap(err, "更新工资批次状态失败")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE payroll_items SET status='paid' WHERE payroll_batch_id=?`, batchID); err != nil {
		return 0, apperror.Wrap(err, "更新工资明细状态失败")
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交工资发放事务失败")
	}
	return uint64(entryID), nil
}
