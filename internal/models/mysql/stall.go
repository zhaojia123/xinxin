package mysql

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/apperror"
)

const defaultConradRate = 0.03

func stallNumber(value string) float64 {
	n, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return n
}

func stallRate(value float64) string { return Decimal(value * 100) }

// conradRateForMonth 读取当月统一康莱德卡机抽点；未配置时默认3%。
func conradRateForMonth(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, month string) (float64, error) {
	var rate sql.NullFloat64
	err := queryer.QueryRowContext(ctx, `SELECT conrad_rate FROM stall_commission_settings WHERE turnover_month=?`, month+"-01").Scan(&rate)
	if err == sql.ErrNoRows {
		return defaultConradRate, nil
	}
	if err != nil {
		return 0, apperror.Wrap(err, "读取康莱德抽点比例失败")
	}
	return rate.Float64, nil
}

// hasStallAutomaticCommissionRule 判断当前档口是否已配置自动抽成规则。
func hasStallAutomaticCommissionRule(item *response.Stall) bool {
	switch item.AllocationMode {
	case "profit_share":
		return stallNumber(item.ProfitJiaRate) != 0 || stallNumber(item.ProfitYiRate) != 0
	case "custom":
		return stallNumber(item.MonthlyRent) != 0 ||
			stallNumber(item.FixedCommissionAmount) != 0 ||
			stallNumber(item.GuaranteedRate) != 0 ||
			(stallNumber(item.BaseAmountRate) != 0 && stallNumber(item.BaseAmountFactor) != 0) ||
			stallNumber(item.TotalAmountRate) != 0
	default:
		return false
	}
}

// calculateStall 只按档口自身营业额、成本和规则计算；康莱德统一在月度汇总时扣除。
func calculateStall(item *response.Stall, _ float64) {
	card := stallNumber(item.SchoolCardAmount)
	huilaimi := stallNumber(item.HuilaimiAmount)
	total := card + huilaimi
	cost := stallNumber(item.CostAmount)
	if cost < 0 {
		cost = 0
	}
	// 档口列表不参与康莱德比例计算，避免每个档口分别扣减造成口径混乱。
	conrad := 0.0
	net := total
	availableAfterCost := net - cost
	jia, yi, rent, fixed, guaranteed, base, totalRate := 0.0, 0.0, stallNumber(item.MonthlyRent), stallNumber(item.FixedCommissionAmount), 0.0, 0.0, 0.0
	own := 0.0
	if item.AllocationMode == "profit_share" {
		// 自营档口先扣自身成本、房租和固定扣减，再把剩余金额按甲乙固定比例分配。
		own = availableAfterCost - rent - fixed
		// 甲乙净额分成是全局固定规则，不使用历史档口中可变的比例。
		item.ProfitJiaRate, item.ProfitYiRate = "90.00", "10.00"
		// 亏损同样按 90/10 保留在档口和月度核算中，避免被静默丢弃。
		jia = own * 0.9
		yi = own * 0.1
	} else if item.AllocationMode == "custom" {
		guaranteed = stallNumber(item.GuaranteedAmount) * stallNumber(item.GuaranteedRate) / 100
		// 非自营档口的比例抽成只以本档口营业额为基数。
		base = net * stallNumber(item.BaseAmountRate) / 100 * stallNumber(item.BaseAmountFactor) / 100
		totalRate = net * stallNumber(item.TotalAmountRate) / 100
		own = rent + fixed + guaranteed + base + totalRate
		jia = own * 0.9
		yi = own * 0.1
	}
	// 非自营档口没有单独抽点规则时，默认将全部可计算额按甲乙分配；
	// 当月手填金额视为该月覆盖规则，优先于默认全额分配。
	canManualOwnCommission := item.AllocationMode == "custom" && !hasStallAutomaticCommissionRule(item)
	if canManualOwnCommission {
		own = net
		if item.ManualOwnCommissionSet {
			own = stallNumber(item.ManualOwnCommission)
		}
		jia, yi = own*0.9, own*0.1
	}
	item.NetTurnover = Decimal(net)
	// 所有档口的成本都要进入余额计算；自营档口还会继续扣除房租和固定扣减。
	item.CostDeductedTurnover = Decimal(availableAfterCost)
	item.TotalTurnover = Decimal(total)
	item.ConradCommission = Decimal(conrad)
	item.JiaCommission = Decimal(jia)
	item.YiCommission = Decimal(yi)
	item.OwnCommission = Decimal(own)
	item.CanManualOwnCommission = canManualOwnCommission
	item.RentCommission = Decimal(rent + fixed)
	item.GuaranteedCommission = Decimal(guaranteed)
	item.BaseCommission = Decimal(base)
	item.TotalRateCommission = Decimal(totalRate)
	item.TotalCommission = Decimal(conrad + own)
	remaining := availableAfterCost - own
	if item.AllocationMode == "profit_share" {
		remaining -= rent + fixed
	}
	item.Remaining = Decimal(remaining)
}

// ApplyMonthlyOperatingExpense 以各档口已计算出的甲乙可抽成金额为基数，扣除水电和人工后统一分配。
func ApplyMonthlyOperatingExpense(summary *response.StallSummary, expense response.StallMonthlyExpense) {
	// 自营档口的金额已扣自身成本、房租等；其他档口的金额已按其抽点规则计算。
	// 康莱德只在月度汇总中按全部卡机统一扣一次，不参与任何单档口计算。
	operating := stallNumber(expense.TotalAmount) + stallNumber(summary.ConradCommission)
	commissionBeforeExpense := stallNumber(summary.SelfOperatedCommission) + stallNumber(summary.OtherStallCommission)
	// 自营成本已在单档口计算时扣除；这里继续扣水电和人工等当月经营支出。
	result := commissionBeforeExpense - operating
	summary.OperatingExpense = Decimal(operating)
	summary.CommissionBeforeOperatingExpense = Decimal(commissionBeforeExpense)
	summary.BalanceBeforeExpenses = Decimal(commissionBeforeExpense)
	summary.DistributableBalance = Decimal(result)
	summary.MonthlyLoss, summary.HasMonthlyLoss = "0.00", result < 0
	if result < 0 {
		summary.MonthlyLoss = Decimal(-result)
		summary.JiaCommission, summary.YiCommission, summary.OwnCommission = "0.00", "0.00", "0.00"
		return
	}
	summary.JiaCommission, summary.YiCommission = Decimal(result*0.9), Decimal(result*0.1)
	summary.OwnCommission = Decimal(result)
	summary.TotalCommission = Decimal(stallNumber(summary.ConradCommission) + result)
}

func scanStall(row interface{ Scan(...any) error }) (response.Stall, error) {
	var item response.Stall
	var turnoverID sql.NullInt64
	var month sql.NullString
	var mode sql.NullString
	var rent, fixed, defaultGuaranteed, jiaRate, yiRate, guaranteedRate, baseRate, baseFactor, totalRate sql.NullFloat64
	var card, huilaimi, cost, guaranteedAmount, manualOwnCommission sql.NullFloat64
	if err := row.Scan(&item.ID, &item.StallNo, &item.Name, &item.ResponsibleName, &item.ResponsibleMobile, &rent, &fixed, &defaultGuaranteed, &mode, &jiaRate, &yiRate, &guaranteedRate, &baseRate, &baseFactor, &totalRate, &item.Remark, &turnoverID, &month, &card, &huilaimi, &cost, &guaranteedAmount, &manualOwnCommission); err != nil {
		return response.Stall{}, err
	}
	item.MonthlyRent = Decimal(rent.Float64)
	item.FixedCommissionAmount = Decimal(fixed.Float64)
	item.DefaultGuaranteedAmount = Decimal(defaultGuaranteed.Float64)
	item.AllocationMode = mode.String
	if item.AllocationMode == "" {
		item.AllocationMode = "custom"
	}
	item.ProfitJiaRate = stallRate(jiaRate.Float64)
	item.ProfitYiRate = stallRate(yiRate.Float64)
	item.GuaranteedRate = stallRate(guaranteedRate.Float64)
	item.BaseAmountRate = stallRate(baseRate.Float64)
	item.BaseAmountFactor = stallRate(baseFactor.Float64)
	item.TotalAmountRate = stallRate(totalRate.Float64)
	item.Month = month.String
	item.TurnoverID = uint64(turnoverID.Int64)
	item.SchoolCardAmount = Decimal(card.Float64)
	item.HuilaimiAmount = Decimal(huilaimi.Float64)
	item.CostAmount = Decimal(cost.Float64)
	item.GuaranteedAmount = Decimal(guaranteedAmount.Float64)
	item.ManualOwnCommission = Decimal(manualOwnCommission.Float64)
	item.ManualOwnCommissionSet = manualOwnCommission.Valid
	if guaranteedAmount.Float64 == 0 {
		item.GuaranteedAmount = item.DefaultGuaranteedAmount
	}
	calculateStall(&item, 0)
	return item, nil
}

func stallSelect() string {
	return `SELECT s.id,s.stall_no,s.name,s.responsible_name,s.responsible_mobile,s.monthly_rent,s.fixed_commission_amount,s.default_guaranteed_amount,
 s.allocation_mode,s.profit_jia_rate,s.profit_yi_rate,s.guaranteed_rate,s.base_amount_rate,s.base_amount_factor,s.total_amount_rate,s.remark,
	 t.id,DATE_FORMAT(t.turnover_month,'%Y-%m'),COALESCE(t.card_amount,0),COALESCE(t.huilaimi_amount,0),COALESCE((SELECT SUM(c.amount) FROM stall_cost_entries c WHERE c.stall_id=s.id AND c.cost_month=t.turnover_month AND c.active=1),0),COALESCE(t.guaranteed_amount,0),t.manual_own_commission`
}

func stallMonthlySelect() string {
	return `SELECT s.id,s.stall_no,s.name,s.responsible_name,s.responsible_mobile,
	 COALESCE(r.monthly_rent,s.monthly_rent),COALESCE(r.fixed_commission_amount,s.fixed_commission_amount),COALESCE(r.default_guaranteed_amount,s.default_guaranteed_amount),
 COALESCE(r.allocation_mode,s.allocation_mode),COALESCE(r.profit_jia_rate,s.profit_jia_rate),COALESCE(r.profit_yi_rate,s.profit_yi_rate),
 COALESCE(r.guaranteed_rate,s.guaranteed_rate),COALESCE(r.base_amount_rate,s.base_amount_rate),COALESCE(r.base_amount_factor,s.base_amount_factor),COALESCE(r.total_amount_rate,s.total_amount_rate),s.remark,
	 t.id,DATE_FORMAT(t.turnover_month,'%Y-%m'),COALESCE(t.card_amount,0),COALESCE(t.huilaimi_amount,0),COALESCE((SELECT SUM(c.amount) FROM stall_cost_entries c WHERE c.stall_id=s.id AND c.cost_month=t.turnover_month AND c.active=1),0),COALESCE(t.guaranteed_amount,0),t.manual_own_commission`
}

// StallSettings 查询指定月份的康莱德公司抽成比例；未设置时默认3%。
func (s *Store) StallSettings(ctx context.Context, month string) (response.StallSettings, error) {
	if err := s.ready(); err != nil {
		return response.StallSettings{}, err
	}
	var item response.StallSettings
	var rate sql.NullFloat64
	if err := s.DB.QueryRowContext(ctx, `SELECT DATE_FORMAT(turnover_month,'%Y-%m'),canteen_name,conrad_rate,remark FROM stall_commission_settings WHERE turnover_month=?`, month+"-01").Scan(&item.Month, &item.CanteenName, &rate, &item.Remark); err == sql.ErrNoRows {
		return response.StallSettings{Month: month, CanteenName: "厦门理工知行2食堂", ConradRate: "3.00"}, nil
	} else if err != nil {
		return response.StallSettings{}, apperror.Wrap(err, "查询档口抽成设置失败")
	}
	item.ConradRate = stallRate(rate.Float64)
	return item, nil
}

// StallsMaster 只查询档口基础配置，不带任何月份营业额。
func (s *Store) StallsMaster(ctx context.Context, keyword string) ([]response.Stall, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	keyword = strings.TrimSpace(keyword)
	query := stallSelect() + ` FROM stall_accounts s LEFT JOIN stall_turnovers t ON 1=0 WHERE s.active=1`
	args := make([]any, 0, 4)
	if keyword != "" {
		query += ` AND (s.stall_no LIKE ? OR s.name LIKE ? OR s.responsible_name LIKE ? OR s.responsible_mobile LIKE ?)`
		value := "%" + keyword + "%"
		args = append(args, value, value, value, value)
	}
	query += ` ORDER BY s.id`
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperror.Wrap(err, "查询档口设置失败")
	}
	defer rows.Close()
	items := make([]response.Stall, 0)
	for rows.Next() {
		item, err := scanStall(rows)
		if err != nil {
			return nil, apperror.Wrap(err, "读取档口设置失败")
		}
		item.Seq = len(items) + 1
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历档口设置失败")
	}
	return items, nil
}

// SaveStallSettings 保存指定月份的康莱德公司抽成比例。
func (s *Store) SaveStallSettings(ctx context.Context, actor uint64, v request.StallSettingsInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO stall_commission_settings(turnover_month,canteen_name,conrad_rate,remark) VALUES(?,?,?,?) ON DUPLICATE KEY UPDATE canteen_name=VALUES(canteen_name),conrad_rate=VALUES(conrad_rate),remark=VALUES(remark)`, v.Month+"-01", v.CanteenName, v.ConradRate, v.Remark)
	if err != nil {
		return apperror.Wrap(err, "保存档口抽成设置失败")
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_text,detail) VALUES(?,?,?,?,?,?)`, actor, "stalls", "保存抽成设置", "stall_commission_settings", v.Month, "康莱德公司抽成比例")
	return err
}

// SaveStallMonthlyRule 保存指定档口指定月份的规则覆盖，不修改档口长期规则。
func (s *Store) SaveStallMonthlyRule(ctx context.Context, actor uint64, v request.StallMonthlyRuleInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO stall_monthly_rules(stall_id,turnover_month,monthly_rent,fixed_commission_amount,default_guaranteed_amount,allocation_mode,profit_jia_rate,profit_yi_rate,guaranteed_rate,base_amount_rate,base_amount_factor,total_amount_rate,remark) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE monthly_rent=VALUES(monthly_rent),fixed_commission_amount=VALUES(fixed_commission_amount),default_guaranteed_amount=VALUES(default_guaranteed_amount),allocation_mode=VALUES(allocation_mode),profit_jia_rate=VALUES(profit_jia_rate),profit_yi_rate=VALUES(profit_yi_rate),guaranteed_rate=VALUES(guaranteed_rate),base_amount_rate=VALUES(base_amount_rate),base_amount_factor=VALUES(base_amount_factor),total_amount_rate=VALUES(total_amount_rate),remark=VALUES(remark)`, v.StallID, v.Month+"-01", v.MonthlyRent, v.FixedCommissionAmount, v.DefaultGuaranteedAmount, v.AllocationMode, v.ProfitJiaRate, v.ProfitYiRate, v.GuaranteedRate, v.BaseAmountRate, v.BaseAmountFactor, v.TotalAmountRate, v.Remark)
	if err != nil {
		return apperror.Wrap(err, "保存档口当月规则失败")
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_id,target_text,detail) VALUES(?,?,?,?,?,?,?)`, actor, "stalls", "保存档口当月规则", "stall_monthly_rules", v.StallID, v.Month, "仅影响当月核算")
	return err
}

// StallsWithTurnover 查询档口和指定月份营业额，并返回三方汇总。
func (s *Store) StallsWithTurnover(ctx context.Context, month, keyword string) ([]response.Stall, response.StallSummary, error) {
	if err := s.ready(); err != nil {
		return nil, response.StallSummary{}, err
	}
	conradRate, err := conradRateForMonth(ctx, s.DB, month)
	if err != nil {
		return nil, response.StallSummary{}, err
	}
	keyword = strings.TrimSpace(keyword)
	query := stallMonthlySelect() + ` FROM stall_accounts s LEFT JOIN stall_turnovers t ON t.stall_id=s.id AND t.turnover_month=? AND t.active=1 LEFT JOIN stall_monthly_rules r ON r.stall_id=s.id AND r.turnover_month=? WHERE s.active=1`
	args := []any{month + "-01", month + "-01"}
	if keyword != "" {
		query += ` AND (s.stall_no LIKE ? OR s.name LIKE ? OR s.responsible_name LIKE ? OR s.responsible_mobile LIKE ?)`
		value := "%" + keyword + "%"
		args = append(args, value, value, value, value)
	}
	query += ` ORDER BY s.id`
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, response.StallSummary{}, apperror.Wrap(err, "查询档口列表失败")
	}
	defer rows.Close()
	items := make([]response.Stall, 0)
	var summary response.StallSummary
	for rows.Next() {
		item, err := scanStall(rows)
		if err != nil {
			return nil, response.StallSummary{}, apperror.Wrap(err, "读取档口列表失败")
		}
		item.Seq = len(items) + 1
		items = append(items, item)
		summary.StallCount++
		summary.TotalCardTurnover = Decimal(stallNumber(summary.TotalCardTurnover) + stallNumber(item.SchoolCardAmount))
		summary.TotalHuilaimi = Decimal(stallNumber(summary.TotalHuilaimi) + stallNumber(item.HuilaimiAmount))
		summary.TotalTurnover = Decimal(stallNumber(summary.TotalTurnover) + stallNumber(item.TotalTurnover))
		summary.TotalCost = Decimal(stallNumber(summary.TotalCost) + stallNumber(item.CostAmount))
		summary.JiaCommission = Decimal(stallNumber(summary.JiaCommission) + stallNumber(item.JiaCommission))
		summary.YiCommission = Decimal(stallNumber(summary.YiCommission) + stallNumber(item.YiCommission))
		summary.OwnCommission = Decimal(stallNumber(summary.OwnCommission) + stallNumber(item.OwnCommission))
		summary.TotalCommission = Decimal(stallNumber(summary.TotalCommission) + stallNumber(item.TotalCommission))
		summary.Remaining = Decimal(stallNumber(summary.Remaining) + stallNumber(item.Remaining))
		if item.AllocationMode == "profit_share" {
			summary.SpecialProfitPool = Decimal(stallNumber(summary.SpecialProfitPool) + stallNumber(item.NetTurnover))
			summary.SpecialStallCost = Decimal(stallNumber(summary.SpecialStallCost) + stallNumber(item.CostAmount))
			summary.BalanceBeforeExpenses = Decimal(stallNumber(summary.BalanceBeforeExpenses) + stallNumber(item.OwnCommission))
			summary.SelfOperatedCardNet = Decimal(stallNumber(summary.SelfOperatedCardNet) + stallNumber(item.SchoolCardAmount) - stallNumber(item.ConradCommission))
			summary.SelfOperatedHuilaimi = Decimal(stallNumber(summary.SelfOperatedHuilaimi) + stallNumber(item.HuilaimiAmount))
			summary.SelfOperatedRevenueAfterConrad = Decimal(stallNumber(summary.SelfOperatedRevenueAfterConrad) + stallNumber(item.SchoolCardAmount) - stallNumber(item.ConradCommission) + stallNumber(item.HuilaimiAmount))
			summary.SelfOperatedCommission = Decimal(stallNumber(summary.SelfOperatedCommission) + stallNumber(item.OwnCommission))
		} else {
			summary.BalanceBeforeExpenses = Decimal(stallNumber(summary.BalanceBeforeExpenses) + stallNumber(item.Remaining))
			summary.OtherStallRevenueAfterConrad = Decimal(stallNumber(summary.OtherStallRevenueAfterConrad) + stallNumber(item.NetTurnover))
			// 非自营档口只把本方实际抽成加入甲乙可分配额，营业额本身不重复计入。
			summary.OtherStallCommission = Decimal(stallNumber(summary.OtherStallCommission) + stallNumber(item.OwnCommission))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, response.StallSummary{}, apperror.Wrap(err, "遍历档口列表失败")
	}
	// 康莱德为全月统一成本：全部卡机营业额 × 当月比例，只在汇总层扣除一次。
	summary.ConradCommission = Decimal(stallNumber(summary.TotalCardTurnover) * conradRate)
	summary.TotalCardAfterConrad = Decimal(stallNumber(summary.TotalCardTurnover) - stallNumber(summary.ConradCommission))
	return items, summary, nil
}

func (s *Store) StallRecord(ctx context.Context, id uint64) (response.Stall, error) {
	if err := s.ready(); err != nil {
		return response.Stall{}, err
	}
	row := s.DB.QueryRowContext(ctx, stallSelect()+` FROM stall_accounts s LEFT JOIN stall_turnovers t ON 1=0 WHERE s.id=? AND s.active=1`, id)
	item, err := scanStall(row)
	if err == sql.ErrNoRows {
		return response.Stall{}, ErrNotFound
	}
	if err != nil {
		return response.Stall{}, apperror.Wrap(err, "查询档口详情失败")
	}
	return item, nil
}

func (s *Store) SaveStall(ctx context.Context, id, actor uint64, v request.StallInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存档口事务失败")
	}
	defer tx.Rollback()
	values := []any{v.StallNo, v.Name, v.ResponsibleName, v.ResponsibleMobile, v.MonthlyRent, v.FixedCommissionAmount, v.DefaultGuaranteedAmount, v.AllocationMode, v.ProfitJiaRate, v.ProfitYiRate, v.GuaranteedRate, v.BaseAmountRate, v.BaseAmountFactor, v.TotalAmountRate, v.Remark}
	if id == 0 {
		result, err := tx.ExecContext(ctx, `INSERT INTO stall_accounts(stall_no,name,responsible_name,responsible_mobile,monthly_rent,fixed_commission_amount,default_guaranteed_amount,allocation_mode,profit_jia_rate,profit_yi_rate,guaranteed_rate,base_amount_rate,base_amount_factor,total_amount_rate,remark) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, values...)
		if err != nil {
			return 0, err
		}
		last, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}
		id = uint64(last)
	} else {
		result, err := tx.ExecContext(ctx, `UPDATE stall_accounts SET stall_no=?,name=?,responsible_name=?,responsible_mobile=?,monthly_rent=?,fixed_commission_amount=?,default_guaranteed_amount=?,allocation_mode=?,profit_jia_rate=?,profit_yi_rate=?,guaranteed_rate=?,base_amount_rate=?,base_amount_factor=?,total_amount_rate=?,remark=? WHERE id=? AND active=1`, append(values, id)...)
		if err != nil {
			return 0, err
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return 0, ErrNotFound
		}
	}
	if err := miniAudit(ctx, tx, actor, "stalls", id, "档口："+v.Name); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (s *Store) DeleteStall(ctx context.Context, id, actor uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var name string
	if err := tx.QueryRowContext(ctx, `SELECT name FROM stall_accounts WHERE id=? AND active=1 FOR UPDATE`, id).Scan(&name); err == sql.ErrNoRows {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE stall_accounts SET active=0,deleted_at=NOW(),deleted_by=? WHERE id=?`, actor, id); err != nil {
		return err
	}
	if err := miniAudit(ctx, tx, actor, "stalls", id, "删除档口："+name); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SaveStallTurnover(ctx context.Context, id, actor uint64, v request.StallTurnoverInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO stall_turnovers(stall_id,turnover_month,school_amount,card_amount,huilaimi_amount,cost_amount,guaranteed_amount,manual_own_commission,remark) VALUES(?,?,?,?,?,?,?,NULLIF(?,''),?) ON DUPLICATE KEY UPDATE school_amount=VALUES(school_amount),card_amount=VALUES(card_amount),huilaimi_amount=VALUES(huilaimi_amount),cost_amount=VALUES(cost_amount),guaranteed_amount=VALUES(guaranteed_amount),manual_own_commission=VALUES(manual_own_commission),remark=VALUES(remark),active=1,deleted_at=NULL,deleted_by=NULL`, v.StallID, v.Month+"-01", v.CardAmount, v.CardAmount, v.HuilaimiAmount, v.CostAmount, v.GuaranteedAmount, v.ManualOwnCommission, v.Remark); err != nil {
		return 0, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM stall_turnovers WHERE stall_id=? AND turnover_month=?`, v.StallID, v.Month+"-01").Scan(&id); err != nil {
		return 0, apperror.Wrap(err, "读取营业额记录失败")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_id,target_text,detail) VALUES(?,?,?,?,?,?,?)`, actor, "stalls", "保存营业额", "stall_turnovers", id, v.Month, "档口月度营业额"); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) DeleteStallTurnover(ctx context.Context, id, actor uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE stall_turnovers SET active=0,deleted_at=NOW(),deleted_by=? WHERE id=? AND active=1`, actor, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_id,target_text,detail) VALUES(?,?,?,?,?,?,?)`, actor, "stalls", "删除营业额", "stall_turnovers", id, "营业额记录", "档口月度营业额"); err != nil {
		return err
	}
	return tx.Commit()
}
