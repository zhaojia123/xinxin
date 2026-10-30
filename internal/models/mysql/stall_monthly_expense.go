package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/apperror"
)

type monthlyExpenseRow struct {
	water, electricity   float64
	waterLedger          sql.NullInt64
	electricityLedger    sql.NullInt64
	conradLedger         sql.NullInt64
	cardIncomeLedger     sql.NullInt64
	huilaimiIncomeLedger sql.NullInt64
}

func (s *Store) payrollAmount(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, month string) (float64, error) {
	var amount float64
	// 月度核算需要反映当前已保存的实发工资，不能等到汇总入账后才更新。
	// net_salary 已包含手动实发覆盖；特殊员工工资尚未入账时同样要计入当月人工成本。
	err := queryer.QueryRowContext(ctx, `SELECT
		COALESCE((SELECT SUM(i.net_salary) FROM payroll_items i JOIN payroll_batches b ON b.id=i.payroll_batch_id WHERE b.payroll_month=? AND b.status<>'cancelled'),0)
		+ COALESCE((SELECT SUM(s.amount) FROM special_payroll_records s WHERE s.payroll_month=?),0)`, month+"-01", month+"-01").Scan(&amount)
	if err != nil {
		return 0, apperror.Wrap(err, "统计当月工资失败")
	}
	return amount, nil
}

func (s *Store) MonthlyOperatingExpense(ctx context.Context, month string) (response.StallMonthlyExpense, error) {
	if err := s.ready(); err != nil {
		return response.StallMonthlyExpense{}, err
	}
	month = NormalizeMonth(month)
	automatic, err := s.payrollAmount(ctx, s.DB, month)
	if err != nil {
		return response.StallMonthlyExpense{}, err
	}
	v := response.StallMonthlyExpense{Month: month, AutomaticPayrollAmount: Decimal(automatic)}
	var row monthlyExpenseRow
	err = s.DB.QueryRowContext(ctx, `SELECT water_amount,electricity_amount,water_ledger_entry_id,electricity_ledger_entry_id,conrad_ledger_entry_id,card_income_ledger_entry_id,huilaimi_income_ledger_entry_id FROM stall_monthly_expenses WHERE expense_month=?`, month+"-01").Scan(&row.water, &row.electricity, &row.waterLedger, &row.electricityLedger, &row.conradLedger, &row.cardIncomeLedger, &row.huilaimiIncomeLedger)
	if errors.Is(err, sql.ErrNoRows) {
		v.PayrollAmount, v.TotalAmount = Decimal(automatic), Decimal(automatic)
		return v, nil
	}
	if err != nil {
		return response.StallMonthlyExpense{}, apperror.Wrap(err, "读取当月经营支出失败")
	}
	v.WaterAmount, v.ElectricityAmount = Decimal(row.water), Decimal(row.electricity)
	// 人工成本始终取工资明细的实发合计，避免旧版月度手工覆盖掩盖临时工工资。
	v.PayrollAmount = Decimal(automatic)
	v.TotalAmount = Decimal(row.water + row.electricity + stallNumber(v.PayrollAmount))
	v.LedgerPosted = row.waterLedger.Valid || row.electricityLedger.Valid || row.conradLedger.Valid || row.cardIncomeLedger.Valid || row.huilaimiIncomeLedger.Valid
	return v, nil
}

func (s *Store) SaveMonthlyOperatingExpense(ctx context.Context, actor uint64, input request.StallMonthlyExpenseInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始保存经营支出事务失败")
	}
	defer tx.Rollback()
	// 防御性处理：即使调用方遗漏校验，空水电也按0写入 DECIMAL 列。
	if strings.TrimSpace(input.WaterAmount) == "" {
		input.WaterAmount = "0"
	}
	if strings.TrimSpace(input.ElectricityAmount) == "" {
		input.ElectricityAmount = "0"
	}
	var row monthlyExpenseRow
	err = tx.QueryRowContext(ctx, `SELECT water_amount,electricity_amount,water_ledger_entry_id,electricity_ledger_entry_id,conrad_ledger_entry_id,card_income_ledger_entry_id,huilaimi_income_ledger_entry_id FROM stall_monthly_expenses WHERE expense_month=? FOR UPDATE`, input.Month+"-01").Scan(&row.water, &row.electricity, &row.waterLedger, &row.electricityLedger, &row.conradLedger, &row.cardIncomeLedger, &row.huilaimiIncomeLedger)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return apperror.Wrap(err, "锁定经营支出失败")
	}
	if (row.waterLedger.Valid && stallNumber(input.WaterAmount) != row.water) || (row.electricityLedger.Valid && stallNumber(input.ElectricityAmount) != row.electricity) {
		return MiniInputError{"已入账的水电金额不能修改；人工成本按工资表实发金额自动汇总"}
	}
	// 保留字段兼容旧数据，但保存水电时清除旧版工资覆盖值。
	if _, err := tx.ExecContext(ctx, `INSERT INTO stall_monthly_expenses(expense_month,water_amount,electricity_amount,manual_payroll_amount) VALUES(?,?,?,NULL) ON DUPLICATE KEY UPDATE water_amount=VALUES(water_amount),electricity_amount=VALUES(electricity_amount),manual_payroll_amount=NULL`, input.Month+"-01", input.WaterAmount, input.ElectricityAmount); err != nil {
		return apperror.Wrap(err, "保存经营支出失败")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_text,detail) VALUES(?,?,?,?,?,?)`, actor, "stalls", "保存经营支出", "stall_monthly_expenses", input.Month, "水电与工资核算"); err != nil {
		return err
	}
	return tx.Commit()
}

func expenseCategory(ctx context.Context, tx *sql.Tx, name string) (uint64, error) {
	result, err := tx.ExecContext(ctx, `INSERT INTO ledger_categories(direction,name,enabled,sort_order) VALUES('expense',?,1,50) ON DUPLICATE KEY UPDATE enabled=1,id=LAST_INSERT_ID(id)`, name)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return uint64(id), err
}

func incomeCategory(ctx context.Context, tx *sql.Tx, name string) (uint64, error) {
	result, err := tx.ExecContext(ctx, `INSERT INTO ledger_categories(direction,name,enabled,sort_order) VALUES('income',?,1,50) ON DUPLICATE KEY UPDATE enabled=1,id=LAST_INSERT_ID(id)`, name)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return uint64(id), err
}

// PostMonthlyOperatingExpenseLedger 按月度核算口径写入水电、档口成本、康莱德支出及营业收入。
func (s *Store) PostMonthlyOperatingExpenseLedger(ctx context.Context, actor uint64, input request.StallMonthlyExpenseLedgerInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	// 非自营档口扣除成本和我方抽成后的余额，是当月需要对外结算的支出。
	_, stallSummary, err := s.StallsWithTurnover(ctx, input.Month, "")
	if err != nil {
		return err
	}
	otherUnallocatedBalance := stallNumber(stallSummary.OtherStallUnallocatedBalance)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始经营支出入账事务失败")
	}
	defer tx.Rollback()
	if input.AccountID == 0 {
		input.AccountID, err = ensureSystemLedgerAccount(ctx, tx)
		if err != nil {
			return err
		}
	} else if err := requirePayrollAccount(ctx, tx, input.AccountID); err != nil {
		return err
	}
	// 即使本月未填写水电，也允许直接确认营业收入、康莱德抽点和档口成本入账。
	if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO stall_monthly_expenses(expense_month,water_amount,electricity_amount,manual_payroll_amount) VALUES(?,?,?,NULL)`, input.Month+"-01", 0, 0); err != nil {
		return apperror.Wrap(err, "初始化当月经营支出失败")
	}
	var row monthlyExpenseRow
	err = tx.QueryRowContext(ctx, `SELECT water_amount,electricity_amount,water_ledger_entry_id,electricity_ledger_entry_id,conrad_ledger_entry_id,card_income_ledger_entry_id,huilaimi_income_ledger_entry_id FROM stall_monthly_expenses WHERE expense_month=? FOR UPDATE`, input.Month+"-01").Scan(&row.water, &row.electricity, &row.waterLedger, &row.electricityLedger, &row.conradLedger, &row.cardIncomeLedger, &row.huilaimiIncomeLedger)
	if err != nil {
		return apperror.Wrap(err, "读取经营支出失败")
	}
	if row.waterLedger.Valid || row.electricityLedger.Valid || row.conradLedger.Valid || row.cardIncomeLedger.Valid || row.huilaimiIncomeLedger.Valid {
		return MiniInputError{"本月经营支出已经计入台账，不能重复入账"}
	}
	var cardAmount, huilaimiAmount float64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(card_amount),0),COALESCE(SUM(huilaimi_amount),0) FROM stall_turnovers WHERE turnover_month=? AND active=1`, input.Month+"-01").Scan(&cardAmount, &huilaimiAmount); err != nil {
		return apperror.Wrap(err, "统计档口当月营业额失败")
	}
	conradRate, err := conradRateForMonth(ctx, tx, input.Month)
	if err != nil {
		return err
	}
	conradAmount := cardAmount * conradRate
	// 卡机按原始营业额入收入，康莱德抽点以独立支出入账，确保流水可分别核对且不会重复扣减。
	cardIncomeAmount := cardAmount
	post := func(direction, category, summary string, amount float64) (uint64, error) {
		if amount <= 0 {
			return 0, nil
		}
		categoryID, err := expenseCategory(ctx, tx, category)
		if direction == "income" {
			categoryID, err = incomeCategory(ctx, tx, category)
		}
		if err != nil {
			return 0, err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries(occurred_on,account_id,category_id,direction,amount,summary,counterparty,operator_id,remark) VALUES(?,?,?,?,?,?,?,?,?)`, input.OccurredOn, input.AccountID, categoryID, direction, amount, input.Month+summary, "", actor, "档口月度核算自动入账")
		if err != nil {
			return 0, err
		}
		id, err := result.LastInsertId()
		return uint64(id), err
	}
	waterID, err := post("expense", "水费", "水费", row.water)
	if err != nil {
		return apperror.Wrap(err, "写入水费台账失败")
	}
	electricityID, err := post("expense", "电费", "电费", row.electricity)
	if err != nil {
		return apperror.Wrap(err, "写入电费台账失败")
	}
	conradID, err := post("expense", "康莱德抽点", "康莱德卡机抽点", conradAmount)
	if err != nil {
		return apperror.Wrap(err, "写入康莱德抽点台账失败")
	}
	cardIncomeID, err := post("income", "卡机营业收入", "卡机扣康莱德后营业收入", cardIncomeAmount)
	if err != nil {
		return apperror.Wrap(err, "写入卡机营业收入台账失败")
	}
	huilaimiIncomeID, err := post("income", "汇来米营业收入", "汇来米营业收入", huilaimiAmount)
	if err != nil {
		return apperror.Wrap(err, "写入汇来米营业收入台账失败")
	}
	var stallCostAmount float64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM stall_cost_entries WHERE cost_month=? AND active=1 AND ledger_entry_id IS NULL`, input.Month+"-01").Scan(&stallCostAmount); err != nil {
		return apperror.Wrap(err, "统计未入账档口成本失败")
	}
	stallCostID, err := post("expense", "档口成本", "全部档口成本", stallCostAmount)
	if err != nil {
		return apperror.Wrap(err, "写入档口成本台账失败")
	}
	if stallCostID != 0 {
		// 月度统一入账后，将每条成本关联至同一笔支出，避免成本页再次重复入账。
		if _, err := tx.ExecContext(ctx, `UPDATE stall_cost_entries SET ledger_entry_id=? WHERE cost_month=? AND active=1 AND ledger_entry_id IS NULL`, stallCostID, input.Month+"-01"); err != nil {
			return apperror.Wrap(err, "关联档口成本台账失败")
		}
	}
	otherBalanceID, err := post("expense", "非自营档口未分配余额", "非自营档口未分配余额", otherUnallocatedBalance)
	if err != nil {
		return apperror.Wrap(err, "写入非自营档口未分配余额台账失败")
	}
	if waterID == 0 && electricityID == 0 && conradID == 0 && cardIncomeID == 0 && huilaimiIncomeID == 0 && stallCostID == 0 && otherBalanceID == 0 {
		return MiniInputError{"本月没有可确认入账的水电、康莱德抽点、档口成本、非自营未分配余额或营业收入"}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE stall_monthly_expenses SET water_ledger_entry_id=NULLIF(?,0),electricity_ledger_entry_id=NULLIF(?,0),conrad_ledger_entry_id=NULLIF(?,0),card_income_ledger_entry_id=NULLIF(?,0),huilaimi_income_ledger_entry_id=NULLIF(?,0) WHERE expense_month=?`, waterID, electricityID, conradID, cardIncomeID, huilaimiIncomeID, input.Month+"-01"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_text,detail) VALUES(?,?,?,?,?,?)`, actor, "stalls", "月度核算确认入账", "stall_monthly_expenses", input.Month, "水电、档口成本、非自营未分配余额、康莱德支出及卡机、汇来米收入确认入账；工资由工资模块发放时入账"); err != nil {
		return err
	}
	return tx.Commit()
}

// RevokeMonthlyOperatingExpenseLedger 作废指定月份的自动流水，并恢复成本记录的未入账状态。
func (s *Store) RevokeMonthlyOperatingExpenseLedger(ctx context.Context, actor uint64, month string) error {
	if err := s.ready(); err != nil {
		return err
	}
	month = NormalizeMonth(month)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始撤销月度核算事务失败")
	}
	defer tx.Rollback()
	var row monthlyExpenseRow
	err = tx.QueryRowContext(ctx, `SELECT water_amount,electricity_amount,water_ledger_entry_id,electricity_ledger_entry_id,conrad_ledger_entry_id,card_income_ledger_entry_id,huilaimi_income_ledger_entry_id FROM stall_monthly_expenses WHERE expense_month=? FOR UPDATE`, month+"-01").Scan(&row.water, &row.electricity, &row.waterLedger, &row.electricityLedger, &row.conradLedger, &row.cardIncomeLedger, &row.huilaimiIncomeLedger)
	if errors.Is(err, sql.ErrNoRows) {
		return MiniInputError{"本月尚未确认月度核算入账"}
	}
	if err != nil {
		return apperror.Wrap(err, "读取月度核算入账状态失败")
	}
	ids := make([]int64, 0, 5)
	for _, id := range []sql.NullInt64{row.waterLedger, row.electricityLedger, row.conradLedger, row.cardIncomeLedger, row.huilaimiIncomeLedger} {
		if id.Valid && id.Int64 > 0 {
			ids = append(ids, id.Int64)
		}
	}
	// 档口成本和非自营未分配余额没有单独存入月度表，通过自动流水标识反查，避免影响手工入账记录。
	costRows, err := tx.QueryContext(ctx, `SELECT l.id FROM ledger_entries l WHERE l.occurred_on>=? AND l.occurred_on<DATE_ADD(?,INTERVAL 1 MONTH) AND l.summary IN (?,?) AND l.remark='档口月度核算自动入账' FOR UPDATE`, month+"-01", month+"-01", month+"全部档口成本", month+"非自营档口未分配余额")
	if err != nil {
		return apperror.Wrap(err, "读取月度核算档口成本流水失败")
	}
	for costRows.Next() {
		var costLedgerID int64
		if err = costRows.Scan(&costLedgerID); err != nil {
			costRows.Close()
			return apperror.Wrap(err, "读取月度核算档口成本流水失败")
		}
		ids = append(ids, costLedgerID)
	}
	if err = costRows.Err(); err != nil {
		costRows.Close()
		return apperror.Wrap(err, "遍历月度核算档口成本流水失败")
	}
	costRows.Close()
	ids = uniqueInt64s(ids)
	if len(ids) == 0 {
		return MiniInputError{"本月尚未确认月度核算入账"}
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+1)
	args = append(args, actor)
	for _, id := range ids {
		args = append(args, id)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ledger_entries SET active=0,deleted_at=NOW(),deleted_by=? WHERE id IN (`+placeholders+`) AND active=1`, args...); err != nil {
		return apperror.Wrap(err, "作废月度核算流水失败")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE stall_cost_entries SET ledger_entry_id=NULL WHERE cost_month=? AND ledger_entry_id IN (`+placeholders+`)`, append([]any{month + "-01"}, int64SliceToAny(ids)...)...); err != nil {
		return apperror.Wrap(err, "恢复档口成本入账状态失败")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE stall_monthly_expenses SET water_ledger_entry_id=NULL,electricity_ledger_entry_id=NULL,conrad_ledger_entry_id=NULL,card_income_ledger_entry_id=NULL,huilaimi_income_ledger_entry_id=NULL WHERE expense_month=?`, month+"-01"); err != nil {
		return apperror.Wrap(err, "恢复月度核算状态失败")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_text,detail) VALUES(?,?,?,?,?,?)`, actor, "stalls", "撤销月度核算入账", "stall_monthly_expenses", month, "自动流水已作废，档口成本恢复未入账"); err != nil {
		return apperror.Wrap(err, "写入撤销月度核算日志失败")
	}
	return tx.Commit()
}

func int64SliceToAny(values []int64) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value > 0 {
			if _, exists := seen[value]; !exists {
				seen[value] = struct{}{}
				result = append(result, value)
			}
		}
	}
	return result
}
