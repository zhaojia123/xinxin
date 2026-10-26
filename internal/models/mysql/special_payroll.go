package mysql

import (
	"context"
	"database/sql"
	"errors"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/apperror"
)

// SpecialPayrolls 查询未建档员工的特殊工资，避免把已离开人员强行建档。
func (s *Store) SpecialPayrolls(ctx context.Context, month string) ([]response.SpecialPayrollRecord, response.SpecialPayrollSummary, error) {
	if err := s.ready(); err != nil {
		return nil, response.SpecialPayrollSummary{}, err
	}
	month = NormalizeMonth(month)
	rows, err := s.DB.QueryContext(ctx, `SELECT id,DATE_FORMAT(payroll_month,'%Y-%m'),employee_name,CAST(amount AS CHAR),remark,status,COALESCE(ledger_entry_id,0) FROM special_payroll_records WHERE payroll_month=? ORDER BY status='paid',id DESC`, month+"-01")
	if err != nil {
		return nil, response.SpecialPayrollSummary{}, apperror.Wrap(err, "查询特殊员工工资失败")
	}
	defer rows.Close()
	items := make([]response.SpecialPayrollRecord, 0)
	summary := response.SpecialPayrollSummary{Month: month}
	var amount float64
	for rows.Next() {
		var item response.SpecialPayrollRecord
		if err := rows.Scan(&item.ID, &item.Month, &item.Name, &item.Amount, &item.Remark, &item.Status, &item.LedgerID); err != nil {
			return nil, response.SpecialPayrollSummary{}, apperror.Wrap(err, "读取特殊员工工资失败")
		}
		items = append(items, item)
		summary.Count++
		if item.Status == "paid" {
			summary.PaidCount++
		} else {
			summary.DraftCount++
		}
		amount += stallNumber(item.Amount)
	}
	if err := rows.Err(); err != nil {
		return nil, response.SpecialPayrollSummary{}, apperror.Wrap(err, "遍历特殊员工工资失败")
	}
	summary.Amount = Money(amount)
	return items, summary, nil
}

func (s *Store) SaveSpecialPayroll(ctx context.Context, id, actor uint64, input request.SpecialPayrollInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if err := input.Validate(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存特殊员工工资事务失败")
	}
	defer tx.Rollback()
	var settled int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payroll_settlement_entries WHERE payroll_month=? AND payroll_type='special' FOR UPDATE`, input.Month+"-01").Scan(&settled); err != nil {
		return 0, apperror.Wrap(err, "读取特殊员工工资汇总状态失败")
	}
	if settled > 0 {
		return 0, MiniInputError{"本月特殊员工工资已汇总入账，请改用其他月份记录"}
	}
	if id == 0 {
		result, err := tx.ExecContext(ctx, `INSERT INTO special_payroll_records(payroll_month,employee_name,amount,remark) VALUES(?,?,?,?)`, input.Month+"-01", input.Name, input.Amount, input.Remark)
		if err != nil {
			return 0, apperror.Wrap(err, "新增特殊员工工资失败")
		}
		created, err := result.LastInsertId()
		if err != nil {
			return 0, apperror.Wrap(err, "读取特殊员工工资ID失败")
		}
		id = uint64(created)
	} else {
		result, err := tx.ExecContext(ctx, `UPDATE special_payroll_records SET payroll_month=?,employee_name=?,amount=?,remark=? WHERE id=? AND status='draft'`, input.Month+"-01", input.Name, input.Amount, input.Remark, id)
		if err != nil {
			return 0, apperror.Wrap(err, "修改特殊员工工资失败")
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return 0, MiniInputError{"特殊员工工资已入账或不存在，不能修改"}
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_id,target_text,detail) VALUES(?,?,?,?,?,?,?)`, actor, "payroll", "保存特殊员工工资", "special_payroll_records", id, input.Name, input.Month); err != nil {
		return 0, apperror.Wrap(err, "写入特殊员工工资操作日志失败")
	}
	if err = tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交特殊员工工资事务失败")
	}
	return id, nil
}

func (s *Store) DeleteSpecialPayroll(ctx context.Context, id, actor uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始删除特殊员工工资事务失败")
	}
	defer tx.Rollback()
	var name string
	if err := tx.QueryRowContext(ctx, `SELECT employee_name FROM special_payroll_records WHERE id=? AND status='draft' FOR UPDATE`, id).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MiniInputError{"特殊员工工资已入账或不存在，不能删除"}
		}
		return apperror.Wrap(err, "读取特殊员工工资失败")
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM special_payroll_records WHERE id=?`, id); err != nil {
		return apperror.Wrap(err, "删除特殊员工工资失败")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_id,target_text,detail) VALUES(?,?,?,?,?,?,?)`, actor, "payroll", "删除特殊员工工资", "special_payroll_records", id, name, "未入账记录"); err != nil {
		return apperror.Wrap(err, "写入特殊员工工资操作日志失败")
	}
	return tx.Commit()
}

// SettleSpecialPayroll 将当月未建档员工工资合成一条支出流水。
func (s *Store) SettleSpecialPayroll(ctx context.Context, actor uint64, input request.SpecialPayrollSettlementInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if err := input.Validate(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始特殊员工工资入账事务失败")
	}
	defer tx.Rollback()
	var existing uint64
	err = tx.QueryRowContext(ctx, `SELECT ledger_entry_id FROM payroll_settlement_entries WHERE payroll_month=? AND payroll_type='special' FOR UPDATE`, input.Month+"-01").Scan(&existing)
	if err == nil {
		return existing, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, apperror.Wrap(err, "读取特殊员工工资汇总失败")
	}
	var amount float64
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0),COUNT(*) FROM special_payroll_records WHERE payroll_month=? AND status='draft' FOR UPDATE`, input.Month+"-01").Scan(&amount, &count); err != nil {
		return 0, apperror.Wrap(err, "统计特殊员工工资失败")
	}
	if count == 0 || amount <= 0 {
		return 0, MiniInputError{"本月没有待发放的特殊员工工资"}
	}
	accountID, err := ensureSystemLedgerAccount(ctx, tx)
	if err != nil {
		return 0, err
	}
	categoryID, err := payrollCategoryID(ctx, tx)
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries(occurred_on,account_id,category_id,direction,amount,summary,counterparty,operator_id,remark) VALUES(?,?,?,'expense',?,?,?,?,'未建档员工工资汇总，可在详情查看人员明细')`, input.OccurredOn, accountID, categoryID, amount, input.Month+payrollSettlementLabel("special"), "", actor)
	if err != nil {
		return 0, apperror.Wrap(err, "写入特殊员工工资台账失败")
	}
	ledgerID, err := result.LastInsertId()
	if err != nil {
		return 0, apperror.Wrap(err, "读取特殊员工工资台账ID失败")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO payroll_settlement_entries(payroll_month,payroll_type,ledger_entry_id) VALUES(?, 'special', ?)`, input.Month+"-01", ledgerID); err != nil {
		return 0, apperror.Wrap(err, "保存特殊员工工资汇总关联失败")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE special_payroll_records SET status='paid',ledger_entry_id=?,paid_at=NOW() WHERE payroll_month=? AND status='draft'`, ledgerID, input.Month+"-01"); err != nil {
		return 0, apperror.Wrap(err, "更新特殊员工工资状态失败")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_text,detail) VALUES(?,?,?,?,?,?)`, actor, "payroll", "确认并汇总入账", "special_payroll_records", input.Month, "特殊员工工资汇总入账"); err != nil {
		return 0, apperror.Wrap(err, "写入特殊员工工资操作日志失败")
	}
	if err = tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交特殊员工工资入账事务失败")
	}
	return uint64(ledgerID), nil
}
