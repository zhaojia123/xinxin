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
	manual               sql.NullFloat64
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
	err = s.DB.QueryRowContext(ctx, `SELECT water_amount,electricity_amount,manual_payroll_amount,water_ledger_entry_id,electricity_ledger_entry_id,conrad_ledger_entry_id,card_income_ledger_entry_id,huilaimi_income_ledger_entry_id FROM stall_monthly_expenses WHERE expense_month=?`, month+"-01").Scan(&row.water, &row.electricity, &row.manual, &row.waterLedger, &row.electricityLedger, &row.conradLedger, &row.cardIncomeLedger, &row.huilaimiIncomeLedger)
	if errors.Is(err, sql.ErrNoRows) {
		v.PayrollAmount, v.TotalAmount = Decimal(automatic), Decimal(automatic)
		return v, nil
	}
	if err != nil {
		return response.StallMonthlyExpense{}, apperror.Wrap(err, "读取当月经营支出失败")
	}
	v.WaterAmount, v.ElectricityAmount = Decimal(row.water), Decimal(row.electricity)
	v.PayrollAmount = Decimal(automatic)
	if row.manual.Valid {
		v.ManualPayrollAmount, v.ManualPayrollAmountSet, v.PayrollAmount = Decimal(row.manual.Float64), true, Decimal(row.manual.Float64)
	}
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
	err = tx.QueryRowContext(ctx, `SELECT water_amount,electricity_amount,manual_payroll_amount,water_ledger_entry_id,electricity_ledger_entry_id,conrad_ledger_entry_id,card_income_ledger_entry_id,huilaimi_income_ledger_entry_id FROM stall_monthly_expenses WHERE expense_month=? FOR UPDATE`, input.Month+"-01").Scan(&row.water, &row.electricity, &row.manual, &row.waterLedger, &row.electricityLedger, &row.conradLedger, &row.cardIncomeLedger, &row.huilaimiIncomeLedger)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return apperror.Wrap(err, "锁定经营支出失败")
	}
	if (row.waterLedger.Valid && stallNumber(input.WaterAmount) != row.water) || (row.electricityLedger.Valid && stallNumber(input.ElectricityAmount) != row.electricity) {
		return MiniInputError{"已入账的水电金额不能修改；人工金额可单独调整且不会写入台账"}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO stall_monthly_expenses(expense_month,water_amount,electricity_amount,manual_payroll_amount) VALUES(?,?,?,NULLIF(?,'')) ON DUPLICATE KEY UPDATE water_amount=VALUES(water_amount),electricity_amount=VALUES(electricity_amount),manual_payroll_amount=VALUES(manual_payroll_amount)`, input.Month+"-01", input.WaterAmount, input.ElectricityAmount, input.ManualPayrollAmount); err != nil {
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
	err = tx.QueryRowContext(ctx, `SELECT water_amount,electricity_amount,manual_payroll_amount,water_ledger_entry_id,electricity_ledger_entry_id,conrad_ledger_entry_id,card_income_ledger_entry_id,huilaimi_income_ledger_entry_id FROM stall_monthly_expenses WHERE expense_month=? FOR UPDATE`, input.Month+"-01").Scan(&row.water, &row.electricity, &row.manual, &row.waterLedger, &row.electricityLedger, &row.conradLedger, &row.cardIncomeLedger, &row.huilaimiIncomeLedger)
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
	if waterID == 0 && electricityID == 0 && conradID == 0 && cardIncomeID == 0 && huilaimiIncomeID == 0 && stallCostID == 0 {
		return MiniInputError{"本月没有可确认入账的水电、康莱德抽点、档口成本或营业收入"}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE stall_monthly_expenses SET water_ledger_entry_id=NULLIF(?,0),electricity_ledger_entry_id=NULLIF(?,0),conrad_ledger_entry_id=NULLIF(?,0),card_income_ledger_entry_id=NULLIF(?,0),huilaimi_income_ledger_entry_id=NULLIF(?,0) WHERE expense_month=?`, waterID, electricityID, conradID, cardIncomeID, huilaimiIncomeID, input.Month+"-01"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_text,detail) VALUES(?,?,?,?,?,?)`, actor, "stalls", "月度核算确认入账", "stall_monthly_expenses", input.Month, "水电、档口成本、康莱德支出及卡机、汇来米收入确认入账；工资由工资模块发放时入账"); err != nil {
		return err
	}
	return tx.Commit()
}
