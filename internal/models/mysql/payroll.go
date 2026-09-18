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
	rows, err := s.DB.QueryContext(ctx, `SELECT i.id,b.id,DATE_FORMAT(b.payroll_month,'%Y-%m'),e.employee_no,e.name,e.pay_basis,COALESCE(e.entry_salary,e.current_salary),i.base_salary,i.bonus_amount,i.attendance_deduction,i.other_deduction,i.social_security,i.tax_amount,i.net_salary,i.status FROM payroll_items i JOIN payroll_batches b ON b.id=i.payroll_batch_id JOIN employees e ON e.id=i.employee_id WHERE b.payroll_month=? ORDER BY e.employee_no`, month+"-01")
	if err != nil {
		return nil, apperror.Wrap(err, "查询月度工资失败")
	}
	defer rows.Close()
	result := make([]response.PayrollRecord, 0)
	for rows.Next() {
		var v response.PayrollRecord
		var entrySalary sql.NullFloat64
		var base, bonus, attendance, other, social, tax, net float64
		var state string
		if err := rows.Scan(&v.ID, &v.BatchID, &v.Month, &v.EmployeeNo, &v.Name, &v.PayBasis, &entrySalary, &base, &bonus, &attendance, &other, &social, &tax, &net, &state); err != nil {
			return nil, apperror.Wrap(err, "读取月度工资失败")
		}
		if entrySalary.Valid {
			v.EntrySalary = Money(entrySalary.Float64)
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
	return s.generatePayroll(ctx, month, 0)
}

// GeneratePayrollForEmployee 只为指定员工生成当月工资明细。
func (s *Store) GeneratePayrollForEmployee(ctx context.Context, month string, employeeID uint64) (uint64, error) {
	if employeeID == 0 {
		return 0, apperror.New("员工ID不正确")
	}
	return s.generatePayroll(ctx, month, employeeID)
}

func (s *Store) generatePayroll(ctx context.Context, month string, employeeID uint64) (uint64, error) {
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
	var batchStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM payroll_batches WHERE id=? FOR UPDATE`, batchID).Scan(&batchStatus); err != nil {
		return 0, apperror.Wrap(err, "读取工资批次状态失败")
	}
	if batchStatus == "paid" {
		return 0, apperror.New("本月工资已经发放，不能再次生成")
	}
	if batchStatus == "cancelled" {
		return 0, apperror.New("本月工资批次已取消，不能生成")
	}
	// 月薪员工按21.75个计薪日折算；入离职跨越部分月份时，先按实际在岗日数折算。
	monthlyDaysExpr := `DATEDIFF(LEAST(COALESCE(DATE_SUB(left_on,INTERVAL 1 DAY),DATE_SUB(?,INTERVAL 1 DAY)),DATE_SUB(?,INTERVAL 1 DAY)),GREATEST(COALESCE(joined_on,?),?))+1`
	// 满整月时直接取月薪，避免 5500/21.75 的中间舍入导致显示 5499.92；不满整月才按日薪舍入。
	monthlyBaseExpr := `CASE WHEN pay_basis='monthly' THEN CASE WHEN (` + monthlyDaysExpr + `)>=21.75 THEN ROUND(COALESCE(current_salary,0),2) ELSE ROUND(ROUND(COALESCE(current_salary,0)/21.75,2)*(` + monthlyDaysExpr + `),2) END ELSE 0 END`
	itemArgs := []any{batchID, end, end, start, start, end, end, start, start, start, end, start}
	itemFilter := ""
	if employeeID > 0 {
		itemFilter = " AND id=?"
		itemArgs = append(itemArgs, employeeID)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO payroll_items(payroll_batch_id,employee_id,base_salary) SELECT ?,id,`+monthlyBaseExpr+` FROM employees WHERE active=1 AND (employment_status IN ('active','probation') OR (employment_status='left' AND left_on>=?)) AND (joined_on IS NULL OR joined_on<?) AND (left_on IS NULL OR left_on>=?)`+itemFilter+` ON DUPLICATE KEY UPDATE base_salary=VALUES(base_salary)`, itemArgs...)
	if err != nil {
		return 0, apperror.Wrap(err, "生成员工工资明细失败")
	}
	baseArgs := []any{end, end, start, start, end, end, start, start, batchID}
	baseFilter := ""
	if employeeID > 0 {
		baseFilter = " AND i.employee_id=?"
		baseArgs = append(baseArgs, employeeID)
	}
	_, err = tx.ExecContext(ctx, `UPDATE payroll_items i JOIN employees e ON e.id=i.employee_id SET i.base_salary=`+monthlyBaseExpr+` WHERE i.payroll_batch_id=? AND i.status<>'paid'`+baseFilter, baseArgs...)
	if err != nil {
		return 0, apperror.Wrap(err, "同步员工基本工资失败")
	}
	adjustDeleteArgs := []any{batchID}
	adjustDeleteFilter := ""
	if employeeID > 0 {
		adjustDeleteFilter = " AND i.employee_id=?"
		adjustDeleteArgs = append(adjustDeleteArgs, employeeID)
	}
	_, err = tx.ExecContext(ctx, `DELETE a FROM payroll_adjustments a JOIN payroll_items i ON i.id=a.payroll_item_id WHERE i.payroll_batch_id=? AND i.status<>'paid'`+adjustDeleteFilter, adjustDeleteArgs...)
	if err != nil {
		return 0, apperror.Wrap(err, "清理原工资关联明细失败")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO payroll_adjustments(payroll_item_id,attendance_record_id,adjustment_type,amount,description) SELECT i.id,a.id,'deduction',CASE WHEN e.pay_basis='monthly' AND (a.salary_effect IN ('deduct','deduct_and_subsidy') OR a.record_type='特殊休息') THEN ROUND((CASE WHEN a.duration_days>0 THEN a.duration_days ELSE a.duration_minutes/480 END)*ROUND(COALESCE(e.current_salary,0)/21.75,2),2) ELSE 0 END,CONCAT('考勤扣款：',a.record_type,' ',IF(a.duration_days>0,CONCAT(a.duration_days,'天'),CONCAT(a.duration_minutes,'分钟'))) FROM payroll_items i JOIN employees e ON e.id=i.employee_id JOIN attendance_records a ON a.employee_id=i.employee_id AND a.occurred_on>=? AND a.occurred_on<? AND (e.joined_on IS NULL OR a.occurred_on>=e.joined_on) AND (e.left_on IS NULL OR a.occurred_on<e.left_on) AND a.status IN ('approved','recorded','confirmed') AND a.active=1 WHERE i.payroll_batch_id=? AND (?=0 OR i.employee_id=?) ON DUPLICATE KEY UPDATE amount=VALUES(amount),description=VALUES(description)`, start, end, batchID, employeeID, employeeID)
	if err != nil {
		return 0, apperror.Wrap(err, "关联请假与异常记录失败")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO payroll_adjustments(payroll_item_id,attendance_record_id,adjustment_type,amount,description) SELECT i.id,a.id,'bonus',CASE WHEN a.record_type='特殊休息' AND COALESCE(a.subsidy_amount,0)=0 THEN 50 ELSE COALESCE(a.subsidy_amount,0) END,CONCAT('特殊休息补助：',a.record_type) FROM payroll_items i JOIN employees e ON e.id=i.employee_id JOIN attendance_records a ON a.employee_id=i.employee_id AND a.occurred_on>=? AND a.occurred_on<? AND (e.joined_on IS NULL OR a.occurred_on>=e.joined_on) AND (e.left_on IS NULL OR a.occurred_on<e.left_on) AND (a.salary_effect IN ('subsidy','deduct_and_subsidy') OR a.record_type='特殊休息') AND a.status IN ('approved','recorded','confirmed') AND a.active=1 WHERE i.payroll_batch_id=? AND (?=0 OR i.employee_id=?) ON DUPLICATE KEY UPDATE amount=VALUES(amount),description=VALUES(description)`, start, end, batchID, employeeID, employeeID)
	if err != nil {
		return 0, apperror.Wrap(err, "关联异常补助失败")
	}
	_, err = tx.ExecContext(ctx, `UPDATE payroll_items i SET i.bonus_amount=COALESCE((SELECT SUM(a.amount) FROM payroll_adjustments a WHERE a.payroll_item_id=i.id AND a.adjustment_type='bonus'),0),i.attendance_deduction=COALESCE((SELECT SUM(a.amount) FROM payroll_adjustments a WHERE a.payroll_item_id=i.id AND a.adjustment_type='deduction'),0),i.net_salary=i.base_salary+i.bonus_amount-i.attendance_deduction-i.other_deduction-i.social_security-i.tax_amount WHERE i.payroll_batch_id=? AND i.status<>'paid' AND (?=0 OR i.employee_id=?)`, batchID, employeeID, employeeID)
	if err != nil {
		return 0, apperror.Wrap(err, "计算考勤工资影响失败")
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
	var linked sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT DATE_FORMAT(payroll_month,'%Y-%m'),status,ledger_entry_id FROM payroll_batches WHERE id=? FOR UPDATE`, batchID).Scan(&month, &status, &linked)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, apperror.Wrap(err, "读取工资批次失败")
	}
	if status == "paid" {
		return uint64(linked.Int64), nil
	}
	if status != "confirmed" {
		return 0, apperror.New("请先确认本月工资，再执行发放")
	}
	var amount float64
	var unpaidCount, paidCount int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN status<>'paid' THEN net_salary ELSE 0 END),0),COALESCE(SUM(status<>'paid'),0),COALESCE(SUM(status='paid'),0) FROM payroll_items WHERE payroll_batch_id=?`, batchID).Scan(&amount, &unpaidCount, &paidCount); err != nil {
		return 0, apperror.Wrap(err, "统计待发工资失败")
	}
	if unpaidCount == 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE payroll_batches SET status='paid',paid_at=COALESCE(paid_at,NOW()) WHERE id=?`, batchID); err != nil {
			return 0, apperror.Wrap(err, "同步工资批次状态失败")
		}
		if err := tx.Commit(); err != nil {
			return 0, apperror.Wrap(err, "提交工资批次状态失败")
		}
		return 0, nil
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
	if _, err = tx.ExecContext(ctx, `UPDATE payroll_items SET status='paid',ledger_entry_id=?,paid_at=NOW() WHERE payroll_batch_id=? AND status<>'paid'`, entryID, batchID); err != nil {
		return 0, apperror.Wrap(err, "更新工资明细状态失败")
	}
	var remaining int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payroll_items WHERE payroll_batch_id=? AND status<>'paid'`, batchID).Scan(&remaining); err != nil {
		return 0, apperror.Wrap(err, "读取剩余工资明细失败")
	}
	if remaining == 0 {
		if paidCount == 0 {
			_, err = tx.ExecContext(ctx, `UPDATE payroll_batches SET status='paid',paid_at=NOW(),ledger_entry_id=? WHERE id=?`, entryID, batchID)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE payroll_batches SET status='paid',paid_at=NOW() WHERE id=?`, batchID)
		}
		if err != nil {
			return 0, apperror.Wrap(err, "更新工资批次状态失败")
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交工资发放事务失败")
	}
	return uint64(entryID), nil
}

// PayPayrollItem 只发放一条已确认的工资明细，已发放明细重复调用不会再次入账。
func (s *Store) PayPayrollItem(ctx context.Context, itemID, accountID uint64, occurredOn string) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	date, err := time.Parse("2006-01-02", occurredOn)
	if err != nil {
		return 0, apperror.New("发放日期格式必须是 YYYY-MM-DD")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始单独发放工资事务失败")
	}
	defer tx.Rollback()
	var batchID uint64
	var itemStatus, batchStatus, month, employeeName string
	var amount float64
	err = tx.QueryRowContext(ctx, `SELECT i.payroll_batch_id,i.status,i.net_salary,b.status,DATE_FORMAT(b.payroll_month,'%Y-%m'),e.name FROM payroll_items i JOIN payroll_batches b ON b.id=i.payroll_batch_id JOIN employees e ON e.id=i.employee_id WHERE i.id=? FOR UPDATE`, itemID).Scan(&batchID, &itemStatus, &amount, &batchStatus, &month, &employeeName)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, apperror.Wrap(err, "读取工资明细失败")
	}
	if itemStatus == "paid" {
		return 0, nil
	}
	if batchStatus != "confirmed" {
		return 0, apperror.New("请先确认本月工资，再单独发放")
	}
	if amount <= 0 {
		return 0, apperror.New("该员工实发工资必须大于0")
	}
	var categoryID sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT id FROM ledger_categories WHERE direction='expense' AND name='工资薪酬' AND enabled=1 LIMIT 1`).Scan(&categoryID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, apperror.Wrap(err, "读取工资台账分类失败")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries(occurred_on,account_id,category_id,direction,amount,summary,counterparty) VALUES(?,?,?,'expense',?,?,?)`, date, accountID, categoryID, amount, month+"员工工资-"+employeeName, employeeName)
	if err != nil {
		return 0, apperror.Wrap(err, "写入员工工资台账支出失败")
	}
	entryID, err := result.LastInsertId()
	if err != nil {
		return 0, apperror.Wrap(err, "读取员工工资台账ID失败")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE payroll_items SET status='paid',ledger_entry_id=?,paid_at=NOW() WHERE id=? AND status<>'paid'`, entryID, itemID); err != nil {
		return 0, apperror.Wrap(err, "更新员工工资发放状态失败")
	}
	var remaining int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payroll_items WHERE payroll_batch_id=? AND status<>'paid'`, batchID).Scan(&remaining); err != nil {
		return 0, apperror.Wrap(err, "读取剩余工资明细失败")
	}
	if remaining == 0 {
		if _, err = tx.ExecContext(ctx, `UPDATE payroll_batches SET status='paid',paid_at=NOW() WHERE id=? AND status='confirmed'`, batchID); err != nil {
			return 0, apperror.Wrap(err, "更新工资批次状态失败")
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交单独发放工资事务失败")
	}
	return uint64(entryID), nil
}
