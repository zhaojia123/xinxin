package mysql

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strconv"
	"strings"
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
	defaultRestDays := s.MonthlyRestDays
	if defaultRestDays <= 0 {
		defaultRestDays = 3
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT i.id,b.id,DATE_FORMAT(b.payroll_month,'%Y-%m'),e.employee_no,e.name,e.employment_type,COALESCE(DATE_FORMAT(e.joined_on,'%Y-%m-%d'),''),GREATEST(30-COALESCE(e.monthly_rest_days,`+strconv.Itoa(defaultRestDays)+`),1),COALESCE((SELECT SUM(COALESCE(a.duration_days,0)) FROM attendance_records a WHERE a.employee_id=e.id AND a.category='leave' AND a.record_type NOT IN ('公休','节假日') AND a.active=1 AND a.status IN ('pending','approved','recorded','confirmed') AND a.occurred_on>=b.payroll_month AND a.occurred_on<DATE_ADD(b.payroll_month,INTERVAL 1 MONTH)),0),COALESCE((SELECT SUM(COALESCE(a.duration_days,0)) FROM attendance_records a WHERE a.employee_id=e.id AND a.record_type='公休' AND a.active=1 AND a.status IN ('pending','approved','recorded','confirmed') AND a.occurred_on>=b.payroll_month AND a.occurred_on<DATE_ADD(b.payroll_month,INTERVAL 1 MONTH)),0),e.pay_basis,COALESCE(e.entry_salary,e.current_salary),i.work_hours,i.base_salary,i.bonus_amount,i.attendance_deduction,i.other_deduction,i.social_security,i.tax_amount,i.net_salary,i.manual_net_salary,i.status FROM payroll_items i JOIN payroll_batches b ON b.id=i.payroll_batch_id JOIN employees e ON e.id=i.employee_id WHERE b.payroll_month=? AND b.status<>'cancelled' ORDER BY e.employee_no`, month+"-01")
	if err != nil {
		return nil, apperror.Wrap(err, "查询月度工资失败")
	}
	defer rows.Close()
	result := make([]response.PayrollRecord, 0)
	for rows.Next() {
		var v response.PayrollRecord
		var entrySalary sql.NullFloat64
		var joinedOn sql.NullString
		var requiredAttendanceDays int
		var leaveDays, publicRestDays float64
		var workHours, base, bonus, attendance, other, social, tax, net float64
		var manual sql.NullFloat64
		var state string
		if err := rows.Scan(&v.ID, &v.BatchID, &v.Month, &v.EmployeeNo, &v.Name, &v.EmploymentType, &joinedOn, &requiredAttendanceDays, &leaveDays, &publicRestDays, &v.PayBasis, &entrySalary, &workHours, &base, &bonus, &attendance, &other, &social, &tax, &net, &manual, &state); err != nil {
			return nil, apperror.Wrap(err, "读取月度工资失败")
		}
		if joinedOn.Valid {
			v.JoinedOn = joinedOn.String
		}
		v.RequiredAttendanceDays = requiredAttendanceDays
		v.LeaveDays, v.PublicRestDays = Decimal(leaveDays), Decimal(publicRestDays)
		// 工时是数量而非金额，界面和导出都不应带人民币符号。
		v.WorkHours, v.WorkHoursValue = Decimal(workHours), Decimal(workHours)
		if entrySalary.Valid {
			v.EntrySalary = Money(entrySalary.Float64)
		}
		v.BaseSalary, v.Bonus, v.AttendanceDeduction, v.OtherDeduction, v.Deduction = Money(base), Money(bonus), Money(attendance), Money(other), Money(attendance+other)
		v.SocialSecurity, v.Tax, v.NetSalary = Money(social), Money(tax), Money(net)
		if manual.Valid {
			v.ManualNetSalary, v.ManualNetSalaryValue = Money(manual.Float64), Decimal(manual.Float64)
		}
		v.BaseSalaryValue, v.BonusValue, v.AttendanceDeductionValue, v.OtherDeductionValue, v.SocialSecurityValue, v.TaxValue = Decimal(base), Decimal(bonus), Decimal(attendance), Decimal(other), Decimal(social), Decimal(tax)
		v.Status, v.StatusClass = PayrollStatus(state)
		v.Seq = len(result) + 1
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
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(i.base_salary),0),COALESCE(SUM(i.attendance_deduction+i.other_deduction+i.social_security+i.tax_amount),0),COALESCE(SUM(i.net_salary),0),COUNT(i.id),COALESCE(SUM(i.status='paid'),0) FROM payroll_batches b LEFT JOIN payroll_items i ON i.payroll_batch_id=b.id WHERE b.payroll_month=? AND b.status<>'cancelled'`, month+"-01").Scan(&gross, &deductions, &net, &v.EmployeeCount, &v.PaidCount)
	if err != nil {
		return response.PayrollSummary{}, apperror.Wrap(err, "统计月度工资失败")
	}
	v.Month, v.GrossAmount, v.DeductionAmount, v.NetAmount = month, Money(gross), Money(deductions), Money(net)
	err = s.DB.QueryRowContext(ctx, `SELECT id,status FROM payroll_batches WHERE payroll_month=? AND status<>'cancelled'`, month+"-01").Scan(&v.BatchID, &v.BatchStatus)
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

// ConfirmPayrollItem 只确认一条工资明细；同批次全部确认后自动确认批次。
func (s *Store) ConfirmPayrollItem(ctx context.Context, itemID uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始单独确认工资事务失败")
	}
	defer tx.Rollback()
	var batchID uint64
	var itemStatus, batchStatus string
	err = tx.QueryRowContext(ctx, `SELECT i.payroll_batch_id,i.status,b.status FROM payroll_items i JOIN payroll_batches b ON b.id=i.payroll_batch_id WHERE i.id=? FOR UPDATE`, itemID).Scan(&batchID, &itemStatus, &batchStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return apperror.Wrap(err, "读取工资明细状态失败")
	}
	if itemStatus == "paid" || batchStatus == "paid" {
		return apperror.New("工资已经发放，不能重复确认")
	}
	if batchStatus == "cancelled" {
		return apperror.New("工资批次已取消，不能确认")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE payroll_items SET status='confirmed' WHERE id=? AND status<>'paid'`, itemID); err != nil {
		return apperror.Wrap(err, "确认工资明细失败")
	}
	var pending int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payroll_items WHERE payroll_batch_id=? AND status NOT IN ('confirmed','paid')`, batchID).Scan(&pending); err != nil {
		return apperror.Wrap(err, "读取待确认工资数量失败")
	}
	if pending == 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE payroll_batches SET status='confirmed',confirmed_at=NOW() WHERE id=? AND status<>'paid'`, batchID); err != nil {
			return apperror.Wrap(err, "确认工资批次失败")
		}
	}
	return tx.Commit()
}

func validatePayroll(v request.PayrollUpdate) error {
	for _, n := range []float64{v.WorkHours, v.BaseSalary, v.Bonus, v.AttendanceDeduction, v.OtherDeduction, v.SocialSecurity, v.Tax} {
		if n < 0 || math.IsNaN(n) || math.IsInf(n, 0) {
			return apperror.New("工资金额必须是大于等于0的有效数字")
		}
	}
	if !v.ValidManualNetSalary() {
		return apperror.New("手动实发工资必须是大于等于0的有效数字")
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
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存工资事务失败")
	}
	defer tx.Rollback()
	var payBasis string
	var payRate float64
	if err := tx.QueryRowContext(ctx, `SELECT e.pay_basis,COALESCE(e.current_salary,0) FROM payroll_items i JOIN employees e ON e.id=i.employee_id WHERE i.id=?`, id).Scan(&payBasis, &payRate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, apperror.Wrap(err, "读取工资计薪方式失败")
	}
	if payBasis == "hourly" {
		v.BaseSalary = math.Round(payRate*v.WorkHours*100) / 100
	} else if payBasis == "daily" {
		v.BaseSalary = math.Round(payRate*v.WorkHours/8*100) / 100
	} else if payBasis == "half_daily" {
		v.BaseSalary = math.Round(payRate*v.WorkHours/4*100) / 100
	}
	net := v.BaseSalary + v.Bonus - v.AttendanceDeduction - v.OtherDeduction - v.SocialSecurity - v.Tax
	if v.ManualNetSalary != nil {
		net = math.Round(*v.ManualNetSalary*100) / 100
	} else {
		net = math.Round(net*100) / 100
	}
	var result sql.Result
	if v.ManualNetSalary == nil {
		result, err = tx.ExecContext(ctx, `UPDATE payroll_items SET work_hours=?,base_salary=?,bonus_amount=?,attendance_deduction=?,other_deduction=?,social_security=?,tax_amount=?,net_salary=?,manual_net_salary=NULL WHERE id=? AND status<>'paid'`, v.WorkHours, v.BaseSalary, v.Bonus, v.AttendanceDeduction, v.OtherDeduction, v.SocialSecurity, v.Tax, net, id)
	} else {
		result, err = tx.ExecContext(ctx, `UPDATE payroll_items SET work_hours=?,base_salary=?,bonus_amount=?,attendance_deduction=?,other_deduction=?,social_security=?,tax_amount=?,net_salary=?,manual_net_salary=? WHERE id=? AND status<>'paid'`, v.WorkHours, v.BaseSalary, v.Bonus, v.AttendanceDeduction, v.OtherDeduction, v.SocialSecurity, v.Tax, net, net, id)
	}
	if err != nil {
		return 0, apperror.Wrap(err, "保存工资明细失败")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, apperror.Wrap(err, "读取工资保存结果失败")
	}
	if affected == 0 {
		// MySQL默认只返回“实际发生变化”的行数；金额与原值相同时仍然是有效保存。
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payroll_items WHERE id=? AND status<>'paid'`, id).Scan(&exists); err != nil {
			return 0, apperror.Wrap(err, "读取工资明细状态失败")
		}
		if exists == 0 {
			return 0, ErrNotFound
		}
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
	var batchID uint64
	if err := tx.QueryRowContext(ctx, `SELECT payroll_batch_id FROM payroll_items WHERE id=?`, itemID).Scan(&batchID); err != nil {
		return apperror.Wrap(err, "读取工资批次失败")
	}
	return refreshBatchByID(ctx, tx, batchID)
}

func refreshBatchByID(ctx context.Context, tx *sql.Tx, batchID uint64) error {
	_, err := tx.ExecContext(ctx, `UPDATE payroll_batches b SET employee_count=(SELECT COUNT(*) FROM payroll_items i WHERE i.payroll_batch_id=b.id),gross_amount=(SELECT COALESCE(SUM(base_salary),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id),deduction_amount=(SELECT COALESCE(SUM(attendance_deduction+other_deduction+social_security+tax_amount),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id),net_amount=(SELECT COALESCE(SUM(net_salary),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id) WHERE b.id=?`, batchID)
	if err != nil {
		return apperror.Wrap(err, "同步工资批次汇总失败")
	}
	return nil
}

// DeletePayrollItems 删除尚未发放的工资条，并同步所属批次的汇总金额。
func (s *Store) DeletePayrollItems(ctx context.Context, itemIDs []uint64, actor uint64) (int, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	seen := make(map[uint64]struct{}, len(itemIDs))
	ids := make([]uint64, 0, len(itemIDs))
	for _, id := range itemIDs {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return 0, apperror.New("请选择要删除的工资明细")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始删除工资事务失败")
	}
	defer tx.Rollback()
	batches := make(map[uint64]struct{})
	for _, id := range ids {
		var batchID uint64
		var status, employeeName string
		err := tx.QueryRowContext(ctx, `SELECT i.payroll_batch_id,i.status,e.name FROM payroll_items i JOIN employees e ON e.id=i.employee_id WHERE i.id=? FOR UPDATE`, id).Scan(&batchID, &status, &employeeName)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		if err != nil {
			return 0, apperror.Wrap(err, "读取工资明细失败")
		}
		if status == "paid" {
			return 0, apperror.New("已发放工资不能删除")
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM payroll_items WHERE id=? AND status<>'paid'`, id); err != nil {
			return 0, apperror.Wrap(err, "删除工资明细失败")
		}
		if err := miniAudit(ctx, tx, actor, "payroll", id, "删除工资："+employeeName); err != nil {
			return 0, err
		}
		batches[batchID] = struct{}{}
	}
	for batchID := range batches {
		if err := refreshBatchByID(ctx, tx, batchID); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交删除工资事务失败")
	}
	return len(ids), nil
}

func requirePayrollAccount(ctx context.Context, tx *sql.Tx, accountID uint64) error {
	var found uint64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM ledger_accounts WHERE id=? AND enabled=1 FOR SHARE`, accountID).Scan(&found); errors.Is(err, sql.ErrNoRows) {
		return apperror.New("付款账户不存在或已停用")
	} else if err != nil {
		return apperror.Wrap(err, "读取付款账户失败")
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
	if batchStatus == "cancelled" {
		// 作废只隐藏历史数据，不删除原工资条；重新生成时复用同一批次并重置未生效字段。
		if _, err := tx.ExecContext(ctx, `UPDATE payroll_batches SET status='draft',employee_count=0,gross_amount=0,deduction_amount=0,net_amount=0,confirmed_by=NULL,confirmed_at=NULL,paid_at=NULL,ledger_entry_id=NULL,remark=CONCAT(LEFT(remark,400),'；已作废批次重新生成') WHERE id=?`, batchID); err != nil {
			return 0, apperror.Wrap(err, "恢复作废工资批次失败")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE payroll_items SET status='draft',work_hours=0,base_salary=0,bonus_amount=0,attendance_deduction=0,other_deduction=0,social_security=0,tax_amount=0,net_salary=0,manual_net_salary=NULL,ledger_entry_id=NULL,paid_at=NULL WHERE payroll_batch_id=?`, batchID); err != nil {
			return 0, apperror.Wrap(err, "重置作废工资明细失败")
		}
		batchStatus = "draft"
	}
	if batchStatus == "paid" {
		if employeeID == 0 {
			return 0, apperror.New("本月工资已经发放，不能再次生成")
		}
		if employeeID > 0 {
			var itemStatus string
			if err := tx.QueryRowContext(ctx, `SELECT status FROM payroll_items WHERE payroll_batch_id=? AND employee_id=?`, batchID, employeeID).Scan(&itemStatus); err == nil && itemStatus == "paid" {
				// 单独生成同一员工是幂等操作，已发放后不再重复创建或覆盖。
				return uint64(batchID), nil
			}
		}
	}
	if batchStatus == "cancelled" {
		return 0, apperror.New("本月工资批次已取消，不能生成")
	}
	// 生成工资前同步已到生效日期的最新调薪，保证员工档案当前工资和工资快照一致。
	if _, err := tx.ExecContext(ctx, `UPDATE employees e SET e.current_salary=(SELECT sa.after_salary FROM salary_adjustments sa WHERE sa.employee_id=e.id AND sa.effective_on<=? ORDER BY sa.effective_on DESC,sa.id DESC LIMIT 1) WHERE e.active=1 AND EXISTS (SELECT 1 FROM salary_adjustments sa2 WHERE sa2.employee_id=e.id AND sa2.effective_on<=?)`, start, start); err != nil {
		return 0, apperror.Wrap(err, "同步已生效调薪失败")
	}
	restDays := s.MonthlyRestDays
	if restDays <= 0 {
		restDays = 3
	}
	// 统一按30天计薪，默认3天公休即27个计薪日；员工单独设置公休时按30减去个人公休计算。
	employeeRequiredDaysExpr := `GREATEST(30-COALESCE(e.monthly_rest_days,` + strconv.Itoa(restDays) + `),1)`
	// 入离职跨越部分月份时，实际在岗日数仍按当月在岗区间折算，但满勤基准固定为30减公休天数。
	monthlyDaysExpr := `DATEDIFF(LEAST(COALESCE(DATE_SUB(e.left_on,INTERVAL 1 DAY),DATE_SUB(?,INTERVAL 1 DAY)),DATE_SUB(?,INTERVAL 1 DAY)),GREATEST(COALESCE(e.joined_on,?),?))+1`
	monthCalendarDays := end.AddDate(0, 0, -1).Day()
	// 月薪按“当月自然日-公休天数”作为应出勤天数；入离职跨越部分月份时按实际在岗日数折算。
	partialPublicRestDays := `(SELECT COALESCE(SUM(COALESCE(a0.duration_days,0)),0) FROM attendance_records a0 WHERE a0.employee_id=e.id AND a0.record_type='公休' AND a0.active=1 AND a0.status IN ('pending','approved','recorded','confirmed') AND a0.occurred_on>=? AND a0.occurred_on<? AND (e.joined_on IS NULL OR a0.occurred_on>=e.joined_on) AND (e.left_on IS NULL OR a0.occurred_on<e.left_on))`
	partialLeaveDays := `(SELECT COALESCE(SUM(COALESCE(a0.duration_days,0)),0) FROM attendance_records a0 WHERE a0.employee_id=e.id AND a0.category='leave' AND a0.record_type NOT IN ('公休','节假日') AND a0.active=1 AND a0.status IN ('pending','approved','recorded','confirmed') AND a0.occurred_on>=? AND a0.occurred_on<? AND (e.joined_on IS NULL OR a0.occurred_on>=e.joined_on) AND (e.left_on IS NULL OR a0.occurred_on<e.left_on))`
	partialRestDays := `(CASE WHEN (` + monthlyDaysExpr + `)>=` + employeeRequiredDaysExpr + ` THEN GREATEST(COALESCE(e.monthly_rest_days,` + strconv.Itoa(restDays) + `),` + partialPublicRestDays + `) ELSE ` + partialPublicRestDays + ` END)`
	actualAttendanceDays := `GREATEST((` + monthlyDaysExpr + `)-` + partialRestDays + `-` + partialLeaveDays + `,0)`
	fullMonthCondition := `(` + monthlyDaysExpr + `)>=` + strconv.Itoa(monthCalendarDays)
	monthlyBaseExpr := `CASE WHEN e.pay_basis='monthly' THEN CASE WHEN ` + fullMonthCondition + ` THEN ROUND(COALESCE(e.current_salary,0),2) ELSE ROUND(COALESCE(e.current_salary,0)/` + employeeRequiredDaysExpr + `*` + actualAttendanceDays + `,2) END ELSE 0 END`
	itemArgs := []any{batchID, end, end, start, start, end, end, start, start, end, end, start, start, start, end, start, end, start, end, start, end, start}
	itemFilter := ""
	if employeeID > 0 {
		itemFilter = " AND id=?"
		itemArgs = append(itemArgs, employeeID)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO payroll_items(payroll_batch_id,employee_id,base_salary) SELECT ?,e.id,`+monthlyBaseExpr+` FROM employees e WHERE e.active=1 AND (e.employment_status IN ('active','probation') OR (e.employment_status='left' AND e.left_on>=?)) AND (e.joined_on IS NULL OR e.joined_on<?) AND (e.left_on IS NULL OR e.left_on>=?)`+itemFilter+` ON DUPLICATE KEY UPDATE base_salary=IF(status='paid',base_salary,VALUES(base_salary))`, itemArgs...)
	if err != nil {
		return 0, apperror.Wrap(err, "生成员工工资明细失败")
	}
	if batchStatus == "paid" {
		var draftCount int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payroll_items WHERE payroll_batch_id=? AND status='draft'`, batchID).Scan(&draftCount); err != nil {
			return 0, apperror.Wrap(err, "检查新增工资明细失败")
		}
		if draftCount > 0 {
			if _, err := tx.ExecContext(ctx, `UPDATE payroll_batches SET status='draft',confirmed_by=NULL,confirmed_at=NULL,paid_at=NULL WHERE id=?`, batchID); err != nil {
				return 0, apperror.Wrap(err, "恢复新增工资批次失败")
			}
			batchStatus = "draft"
		}
	}
	baseArgs := []any{end, end, start, start, end, end, start, start, end, end, start, start, start, end, start, end, start, end, batchID}
	baseFilter := ""
	if employeeID > 0 {
		baseFilter = " AND i.employee_id=?"
		baseArgs = append(baseArgs, employeeID)
	}
	_, err = tx.ExecContext(ctx, `UPDATE payroll_items i JOIN employees e ON e.id=i.employee_id SET i.base_salary=`+monthlyBaseExpr+` WHERE i.payroll_batch_id=? AND i.status<>'paid'`+baseFilter, baseArgs...)
	if err != nil {
		return 0, apperror.Wrap(err, "同步员工基本工资失败")
	}
	// 临时工的基本工资来自每日工时记录；工时统一保存为小时，按小时、天或半天自动换算。
	temporaryArgs := []any{start, end, start, end, batchID}
	temporaryFilter := ""
	if employeeID > 0 {
		temporaryFilter = " AND i.employee_id=?"
		temporaryArgs = append(temporaryArgs, employeeID)
	}
	_, err = tx.ExecContext(ctx, `UPDATE payroll_items i JOIN employees e ON e.id=i.employee_id SET i.work_hours=COALESCE((SELECT SUM(w.hours) FROM employee_work_logs w WHERE w.employee_id=e.id AND w.active=1 AND w.work_date>=? AND w.work_date<? AND (e.joined_on IS NULL OR w.work_date>=e.joined_on) AND (e.left_on IS NULL OR w.work_date<e.left_on)),i.work_hours),i.base_salary=ROUND(COALESCE(e.current_salary,0)*COALESCE((SELECT SUM(w2.hours) FROM employee_work_logs w2 WHERE w2.employee_id=e.id AND w2.active=1 AND w2.work_date>=? AND w2.work_date<? AND (e.joined_on IS NULL OR w2.work_date>=e.joined_on) AND (e.left_on IS NULL OR w2.work_date<e.left_on)),i.work_hours)*CASE e.pay_basis WHEN 'daily' THEN 1/8 WHEN 'half_daily' THEN 1/4 ELSE 1 END,2) WHERE i.payroll_batch_id=? AND i.status<>'paid' AND e.employment_type='part_time' AND e.pay_basis IN ('hourly','daily','half_daily')`+temporaryFilter, temporaryArgs...)
	if err != nil {
		return 0, apperror.Wrap(err, "同步小时工每日工时失败")
	}
	adjustDeleteArgs := []any{batchID}
	adjustDeleteFilter := ""
	if employeeID > 0 {
		adjustDeleteFilter = " AND i.employee_id=?"
		adjustDeleteArgs = append(adjustDeleteArgs, employeeID)
	}
	_, err = tx.ExecContext(ctx, `UPDATE payroll_adjustments a JOIN payroll_items i ON i.id=a.payroll_item_id SET a.amount=0,a.description='已作废工资重建关联' WHERE i.payroll_batch_id=? AND i.status<>'paid'`+adjustDeleteFilter, adjustDeleteArgs...)
	if err != nil {
		return 0, apperror.Wrap(err, "作废原工资关联明细失败")
	}
	// 公休额度按员工单独计算：完整自然月时，只有超过个人公休天数的部分才扣款。
	priorPublicRest := `COALESCE((SELECT SUM(p.duration_days) FROM attendance_records p WHERE p.employee_id=a.employee_id AND p.record_type='公休' AND p.active=1 AND p.status IN ('pending','approved','recorded','confirmed') AND p.occurred_on>=? AND p.occurred_on<? AND (p.occurred_on<a.occurred_on OR (p.occurred_on=a.occurred_on AND p.id<a.id))),0)`
	publicRestExcess := `GREATEST(` + priorPublicRest + `+a.duration_days-COALESCE(e.monthly_rest_days,` + strconv.Itoa(restDays) + `),0)-GREATEST(` + priorPublicRest + `-COALESCE(e.monthly_rest_days,` + strconv.Itoa(restDays) + `),0)`
	partialMonthCondition := `(` + monthlyDaysExpr + `)<` + strconv.Itoa(monthCalendarDays)
	// 未满应出勤天数时，公休和请假已从实际出勤基数中扣除，这里不再重复扣款。
	attendanceDays := `(CASE WHEN ` + partialMonthCondition + ` THEN 0 WHEN a.record_type='公休' THEN ` + publicRestExcess + ` WHEN a.record_type='节假日' THEN 0 WHEN a.category='leave' THEN a.duration_days ELSE 0 END)`
	_, err = tx.ExecContext(ctx, `INSERT INTO payroll_adjustments(payroll_item_id,attendance_record_id,adjustment_type,amount,description) SELECT i.id,a.id,'deduction',CASE WHEN e.pay_basis='monthly' THEN ROUND(`+attendanceDays+`*COALESCE(e.current_salary,0)/`+employeeRequiredDaysExpr+`,2) ELSE 0 END,CONCAT('考勤扣款：',a.record_type,' ',IF(a.duration_days>0,CONCAT(a.duration_days,'天'),CONCAT(a.duration_minutes,'分钟'))) FROM payroll_items i JOIN employees e ON e.id=i.employee_id JOIN attendance_records a ON a.employee_id=i.employee_id AND a.occurred_on>=? AND a.occurred_on<? AND (e.joined_on IS NULL OR a.occurred_on>=e.joined_on) AND (e.left_on IS NULL OR a.occurred_on<e.left_on) AND a.status IN ('pending','approved','recorded','confirmed') AND a.active=1 WHERE i.payroll_batch_id=? AND i.status<>'paid' AND (?=0 OR i.employee_id=?) ON DUPLICATE KEY UPDATE amount=VALUES(amount),description=VALUES(description)`, end, end, start, start, start, end, start, end, start, end, batchID, employeeID, employeeID)
	if err != nil {
		return 0, apperror.Wrap(err, "关联请假与异常记录失败")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO payroll_adjustments(payroll_item_id,attendance_record_id,adjustment_type,amount,description) SELECT i.id,a.id,'bonus',COALESCE(a.subsidy_amount,0),CONCAT('特殊请假补助：',a.record_type) FROM payroll_items i JOIN employees e ON e.id=i.employee_id JOIN attendance_records a ON a.employee_id=i.employee_id AND a.occurred_on>=? AND a.occurred_on<? AND (e.joined_on IS NULL OR a.occurred_on>=e.joined_on) AND (e.left_on IS NULL OR a.occurred_on<e.left_on) AND (a.salary_effect IN ('subsidy','deduct_and_subsidy') OR a.record_type='特殊休息') AND a.status IN ('pending','approved','recorded','confirmed') AND a.active=1 WHERE i.payroll_batch_id=? AND i.status<>'paid' AND (?=0 OR i.employee_id=?) ON DUPLICATE KEY UPDATE amount=VALUES(amount),description=VALUES(description)`, start, end, batchID, employeeID, employeeID)
	if err != nil {
		return 0, apperror.Wrap(err, "关联异常补助失败")
	}
	_, err = tx.ExecContext(ctx, `UPDATE payroll_items i SET i.bonus_amount=COALESCE((SELECT SUM(a.amount) FROM payroll_adjustments a WHERE a.payroll_item_id=i.id AND a.adjustment_type='bonus'),0),i.attendance_deduction=COALESCE((SELECT SUM(a.amount) FROM payroll_adjustments a WHERE a.payroll_item_id=i.id AND a.adjustment_type='deduction'),0),i.net_salary=CASE WHEN i.manual_net_salary IS NOT NULL THEN i.manual_net_salary ELSE i.base_salary+i.bonus_amount-i.attendance_deduction-i.other_deduction-i.social_security-i.tax_amount END WHERE i.payroll_batch_id=? AND i.status<>'paid' AND (?=0 OR i.employee_id=?)`, batchID, employeeID, employeeID)
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
	if accountID == 0 {
		return 0, apperror.New("请选择付款账户")
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
	if err := requirePayrollAccount(ctx, tx, accountID); err != nil {
		return 0, err
	}
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

// PayTemporaryPayroll 直接写入一笔临时工工资台账，不关联员工和工资批次。
func (s *Store) PayTemporaryPayroll(ctx context.Context, input request.TemporaryPayrollInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if err := input.Validate(); err != nil {
		return 0, err
	}
	date, _ := time.Parse("2006-01-02", input.OccurredOn)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始临时工发放事务失败")
	}
	defer tx.Rollback()
	if err := requirePayrollAccount(ctx, tx, input.AccountID); err != nil {
		return 0, err
	}
	var categoryID sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT id FROM ledger_categories WHERE direction='expense' AND name='工资薪酬' AND enabled=1 LIMIT 1`).Scan(&categoryID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, apperror.Wrap(err, "读取工资台账分类失败")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries(occurred_on,account_id,category_id,direction,amount,summary,counterparty,remark) VALUES(?,?,?,'expense',?,?,?,?)`, date, input.AccountID, categoryID, input.Amount, "临时工工资-"+input.Name, input.Name, input.Remark)
	if err != nil {
		return 0, apperror.Wrap(err, "写入临时工工资台账失败")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, apperror.Wrap(err, "读取临时工工资台账ID失败")
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交临时工发放事务失败")
	}
	return uint64(id), nil
}

// PayPayrollItems 一次发放选中的工资明细，并为每名员工生成独立工资台账支出。
func (s *Store) PayPayrollItems(ctx context.Context, itemIDs []uint64, accountID uint64, occurredOn string, paymentRemarks map[uint64]string) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if accountID == 0 {
		return 0, apperror.New("请选择付款账户")
	}
	date, err := time.Parse("2006-01-02", occurredOn)
	if err != nil {
		return 0, apperror.New("发放日期格式必须是 YYYY-MM-DD")
	}
	uniqueIDs := make([]uint64, 0, len(itemIDs))
	seen := make(map[uint64]struct{}, len(itemIDs))
	for _, itemID := range itemIDs {
		if itemID == 0 {
			continue
		}
		if _, ok := seen[itemID]; ok {
			continue
		}
		seen[itemID] = struct{}{}
		uniqueIDs = append(uniqueIDs, itemID)
	}
	if len(uniqueIDs) == 0 {
		return 0, apperror.New("请选择要发放的工资")
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始批量发放工资事务失败")
	}
	defer tx.Rollback()
	if err := requirePayrollAccount(ctx, tx, accountID); err != nil {
		return 0, err
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(uniqueIDs)), ",")
	args := make([]any, len(uniqueIDs))
	for i, itemID := range uniqueIDs {
		args[i] = itemID
	}
	rows, err := tx.QueryContext(ctx, `SELECT i.id,i.payroll_batch_id,i.status,i.net_salary,b.status,DATE_FORMAT(b.payroll_month,'%Y-%m'),e.name FROM payroll_items i JOIN payroll_batches b ON b.id=i.payroll_batch_id JOIN employees e ON e.id=i.employee_id WHERE i.id IN (`+placeholders+") FOR UPDATE", args...)
	if err != nil {
		return 0, apperror.Wrap(err, "读取选中工资明细失败")
	}
	type selectedItem struct {
		id, batchID                      uint64
		status, batchStatus, month, name string
		amount                           float64
	}
	items := make([]selectedItem, 0, len(uniqueIDs))
	for rows.Next() {
		var item selectedItem
		if err := rows.Scan(&item.id, &item.batchID, &item.status, &item.amount, &item.batchStatus, &item.month, &item.name); err != nil {
			rows.Close()
			return 0, apperror.Wrap(err, "读取选中工资明细失败")
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, apperror.Wrap(err, "遍历选中工资明细失败")
	}
	rows.Close()
	if len(items) != len(uniqueIDs) {
		return 0, ErrNotFound
	}
	batchID := items[0].batchID
	batchStatus, month := items[0].batchStatus, items[0].month
	unpaidItems := make([]selectedItem, 0, len(items))
	for _, item := range items {
		if item.batchID != batchID || item.batchStatus != batchStatus {
			return 0, apperror.New("所选工资不属于同一工资批次")
		}
		if item.status != "paid" {
			unpaidItems = append(unpaidItems, item)
		}
	}
	if len(unpaidItems) == 0 {
		return 0, tx.Commit()
	}
	if batchStatus != "confirmed" {
		return 0, apperror.New("请先确认本月工资，再批量发放")
	}
	var categoryID sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT id FROM ledger_categories WHERE direction='expense' AND name='工资薪酬' AND enabled=1 LIMIT 1`).Scan(&categoryID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, apperror.Wrap(err, "读取工资台账分类失败")
	}
	var entryID int64
	for _, item := range unpaidItems {
		if item.amount <= 0 {
			return 0, apperror.New("所选员工实发工资必须大于0")
		}
		remark := strings.TrimSpace(paymentRemarks[item.id])
		if len([]rune(remark)) > 500 {
			return 0, apperror.New("收款备注最多填写500字")
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries(occurred_on,account_id,category_id,direction,amount,summary,counterparty,remark) VALUES(?,?,?,'expense',?,?,?,?)`, date, accountID, categoryID, item.amount, month+"员工工资-"+item.name, item.name, remark)
		if err != nil {
			return 0, apperror.Wrap(err, "写入员工工资台账支出失败")
		}
		entryID, err = result.LastInsertId()
		if err != nil {
			return 0, apperror.Wrap(err, "读取员工工资台账ID失败")
		}
		if _, err = tx.ExecContext(ctx, `UPDATE payroll_items SET status='paid',ledger_entry_id=?,paid_at=NOW() WHERE id=? AND status<>'paid'`, entryID, item.id); err != nil {
			return 0, apperror.Wrap(err, "更新员工工资明细状态失败")
		}
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
		return 0, apperror.Wrap(err, "提交批量工资发放事务失败")
	}
	return uint64(entryID), nil
}

// PayPayrollItem 只发放一条已确认的工资明细，已发放明细重复调用不会再次入账。
func (s *Store) PayPayrollItem(ctx context.Context, itemID, accountID uint64, occurredOn, paymentRemark string) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if accountID == 0 {
		return 0, apperror.New("请选择付款账户")
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
	if err := requirePayrollAccount(ctx, tx, accountID); err != nil {
		return 0, err
	}
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
	paymentRemark = strings.TrimSpace(paymentRemark)
	if len([]rune(paymentRemark)) > 500 {
		return 0, apperror.New("收款备注最多填写500字")
	}
	var categoryID sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT id FROM ledger_categories WHERE direction='expense' AND name='工资薪酬' AND enabled=1 LIMIT 1`).Scan(&categoryID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, apperror.Wrap(err, "读取工资台账分类失败")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries(occurred_on,account_id,category_id,direction,amount,summary,counterparty,remark) VALUES(?,?,?,'expense',?,?,?,?)`, date, accountID, categoryID, amount, month+"员工工资-"+employeeName, employeeName, paymentRemark)
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
