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

func stallNumber(value string) float64 {
	n, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return n
}

func stallRate(value float64) string { return Decimal(value * 100) }

// calculateStall 按三方规则计算单个档口：康莱德只看学校卡机，其他规则看卡机+汇来米。
func calculateStall(item *response.Stall, conradRate float64) {
	card := stallNumber(item.SchoolCardAmount)
	huilaimi := stallNumber(item.HuilaimiAmount)
	total := card + huilaimi
	cost := stallNumber(item.CostAmount)
	if cost < 0 {
		cost = 0
	}
	conrad := card * conradRate
	net := total - cost - conrad
	jia, yi, rent, guaranteed, base, totalRate := 0.0, 0.0, stallNumber(item.MonthlyRent), 0.0, 0.0, 0.0
	if item.AllocationMode == "profit_share" {
		if net < 0 {
			net = 0
		}
		jia = net * stallNumber(item.ProfitJiaRate) / 100
		yi = net * stallNumber(item.ProfitYiRate) / 100
	} else if item.AllocationMode == "custom" {
		guaranteed = stallNumber(item.GuaranteedAmount) * stallNumber(item.GuaranteedRate) / 100
		base = total * stallNumber(item.BaseAmountRate) / 100 * stallNumber(item.BaseAmountFactor) / 100
		totalRate = total * stallNumber(item.TotalAmountRate) / 100
	}
	own := jia + yi + rent + guaranteed + base + totalRate
	item.NetTurnover = Decimal(net)
	item.TotalTurnover = Decimal(total)
	item.ConradCommission = Decimal(conrad)
	item.JiaCommission = Decimal(jia)
	item.YiCommission = Decimal(yi)
	item.OwnCommission = Decimal(own)
	item.RentCommission = Decimal(rent)
	item.GuaranteedCommission = Decimal(guaranteed)
	item.BaseCommission = Decimal(base)
	item.TotalRateCommission = Decimal(totalRate)
	item.TotalCommission = Decimal(conrad + own)
	item.Remaining = Decimal(total - cost - conrad - own)
}

func scanStall(row interface{ Scan(...any) error }, conradRate float64) (response.Stall, error) {
	var item response.Stall
	var turnoverID sql.NullInt64
	var month sql.NullString
	var mode sql.NullString
	var rent, defaultGuaranteed, jiaRate, yiRate, guaranteedRate, baseRate, baseFactor, totalRate sql.NullFloat64
	var card, huilaimi, cost, guaranteedAmount sql.NullFloat64
	if err := row.Scan(&item.ID, &item.StallNo, &item.Name, &item.ResponsibleName, &item.ResponsibleMobile, &rent, &defaultGuaranteed, &mode, &jiaRate, &yiRate, &guaranteedRate, &baseRate, &baseFactor, &totalRate, &item.Remark, &turnoverID, &month, &card, &huilaimi, &cost, &guaranteedAmount); err != nil {
		return response.Stall{}, err
	}
	item.MonthlyRent = Decimal(rent.Float64)
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
	if guaranteedAmount.Float64 == 0 {
		item.GuaranteedAmount = item.DefaultGuaranteedAmount
	}
	calculateStall(&item, conradRate)
	return item, nil
}

func stallSelect() string {
	return `SELECT s.id,s.stall_no,s.name,s.responsible_name,s.responsible_mobile,s.monthly_rent,s.default_guaranteed_amount,
 s.allocation_mode,s.profit_jia_rate,s.profit_yi_rate,s.guaranteed_rate,s.base_amount_rate,s.base_amount_factor,s.total_amount_rate,s.remark,
 t.id,DATE_FORMAT(t.turnover_month,'%Y-%m'),COALESCE(t.card_amount,0),COALESCE(t.huilaimi_amount,0),COALESCE(t.cost_amount,0),COALESCE(t.guaranteed_amount,0)`
}

func stallMonthlySelect() string {
	return `SELECT s.id,s.stall_no,s.name,s.responsible_name,s.responsible_mobile,
 COALESCE(r.monthly_rent,s.monthly_rent),COALESCE(r.default_guaranteed_amount,s.default_guaranteed_amount),
 COALESCE(r.allocation_mode,s.allocation_mode),COALESCE(r.profit_jia_rate,s.profit_jia_rate),COALESCE(r.profit_yi_rate,s.profit_yi_rate),
 COALESCE(r.guaranteed_rate,s.guaranteed_rate),COALESCE(r.base_amount_rate,s.base_amount_rate),COALESCE(r.base_amount_factor,s.base_amount_factor),COALESCE(r.total_amount_rate,s.total_amount_rate),s.remark,
 t.id,DATE_FORMAT(t.turnover_month,'%Y-%m'),COALESCE(t.card_amount,0),COALESCE(t.huilaimi_amount,0),COALESCE(t.cost_amount,0),COALESCE(t.guaranteed_amount,0)`
}

// StallSettings 查询指定月份的康莱德公司抽成比例；未设置时返回0。
func (s *Store) StallSettings(ctx context.Context, month string) (response.StallSettings, error) {
	if err := s.ready(); err != nil {
		return response.StallSettings{}, err
	}
	var item response.StallSettings
	var rate sql.NullFloat64
	if err := s.DB.QueryRowContext(ctx, `SELECT DATE_FORMAT(turnover_month,'%Y-%m'),canteen_name,conrad_rate,remark FROM stall_commission_settings WHERE turnover_month=?`, month+"-01").Scan(&item.Month, &item.CanteenName, &rate, &item.Remark); err == sql.ErrNoRows {
		return response.StallSettings{Month: month, CanteenName: "厦门理工知行2食堂", ConradRate: "0"}, nil
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
		item, err := scanStall(rows, 0)
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
	_, err := s.DB.ExecContext(ctx, `INSERT INTO stall_monthly_rules(stall_id,turnover_month,monthly_rent,default_guaranteed_amount,allocation_mode,profit_jia_rate,profit_yi_rate,guaranteed_rate,base_amount_rate,base_amount_factor,total_amount_rate,remark) VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE monthly_rent=VALUES(monthly_rent),default_guaranteed_amount=VALUES(default_guaranteed_amount),allocation_mode=VALUES(allocation_mode),profit_jia_rate=VALUES(profit_jia_rate),profit_yi_rate=VALUES(profit_yi_rate),guaranteed_rate=VALUES(guaranteed_rate),base_amount_rate=VALUES(base_amount_rate),base_amount_factor=VALUES(base_amount_factor),total_amount_rate=VALUES(total_amount_rate),remark=VALUES(remark)`, v.StallID, v.Month+"-01", v.MonthlyRent, v.DefaultGuaranteedAmount, v.AllocationMode, v.ProfitJiaRate, v.ProfitYiRate, v.GuaranteedRate, v.BaseAmountRate, v.BaseAmountFactor, v.TotalAmountRate, v.Remark)
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
	settings, err := s.StallSettings(ctx, month)
	if err != nil {
		return nil, response.StallSummary{}, err
	}
	conradRate := stallNumber(settings.ConradRate) / 100
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
		item, err := scanStall(rows, conradRate)
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
		summary.ConradCommission = Decimal(stallNumber(summary.ConradCommission) + stallNumber(item.ConradCommission))
		summary.JiaCommission = Decimal(stallNumber(summary.JiaCommission) + stallNumber(item.JiaCommission))
		summary.YiCommission = Decimal(stallNumber(summary.YiCommission) + stallNumber(item.YiCommission))
		summary.OwnCommission = Decimal(stallNumber(summary.OwnCommission) + stallNumber(item.OwnCommission))
		summary.TotalCommission = Decimal(stallNumber(summary.TotalCommission) + stallNumber(item.TotalCommission))
		summary.Remaining = Decimal(stallNumber(summary.Remaining) + stallNumber(item.Remaining))
	}
	if err := rows.Err(); err != nil {
		return nil, response.StallSummary{}, apperror.Wrap(err, "遍历档口列表失败")
	}
	return items, summary, nil
}

func (s *Store) StallRecord(ctx context.Context, id uint64) (response.Stall, error) {
	if err := s.ready(); err != nil {
		return response.Stall{}, err
	}
	row := s.DB.QueryRowContext(ctx, stallSelect()+` FROM stall_accounts s LEFT JOIN stall_turnovers t ON 1=0 WHERE s.id=? AND s.active=1`, id)
	item, err := scanStall(row, 0)
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
	values := []any{v.StallNo, v.Name, v.ResponsibleName, v.ResponsibleMobile, v.MonthlyRent, v.DefaultGuaranteedAmount, v.AllocationMode, v.ProfitJiaRate, v.ProfitYiRate, v.GuaranteedRate, v.BaseAmountRate, v.BaseAmountFactor, v.TotalAmountRate, v.Remark}
	if id == 0 {
		result, err := tx.ExecContext(ctx, `INSERT INTO stall_accounts(stall_no,name,responsible_name,responsible_mobile,monthly_rent,default_guaranteed_amount,allocation_mode,profit_jia_rate,profit_yi_rate,guaranteed_rate,base_amount_rate,base_amount_factor,total_amount_rate,remark) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, values...)
		if err != nil {
			return 0, err
		}
		last, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}
		id = uint64(last)
	} else {
		result, err := tx.ExecContext(ctx, `UPDATE stall_accounts SET stall_no=?,name=?,responsible_name=?,responsible_mobile=?,monthly_rent=?,default_guaranteed_amount=?,allocation_mode=?,profit_jia_rate=?,profit_yi_rate=?,guaranteed_rate=?,base_amount_rate=?,base_amount_factor=?,total_amount_rate=?,remark=? WHERE id=? AND active=1`, append(values, id)...)
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
	_, err := s.DB.ExecContext(ctx, `INSERT INTO stall_turnovers(stall_id,turnover_month,school_amount,card_amount,huilaimi_amount,cost_amount,guaranteed_amount,remark) VALUES(?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE school_amount=VALUES(school_amount),card_amount=VALUES(card_amount),huilaimi_amount=VALUES(huilaimi_amount),cost_amount=VALUES(cost_amount),guaranteed_amount=VALUES(guaranteed_amount),remark=VALUES(remark),active=1,deleted_at=NULL,deleted_by=NULL`, v.StallID, v.Month+"-01", v.CardAmount, v.CardAmount, v.HuilaimiAmount, v.CostAmount, v.GuaranteedAmount, v.Remark)
	if err != nil {
		return 0, err
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT id FROM stall_turnovers WHERE stall_id=? AND turnover_month=?`, v.StallID, v.Month+"-01").Scan(&id); err != nil {
		return 0, apperror.Wrap(err, "读取营业额记录失败")
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_id,target_text,detail) VALUES(?,?,?,?,?,?,?)`, actor, "stalls", "保存营业额", "stall_turnovers", id, v.Month, "档口月度营业额")
	return id, err
}

func (s *Store) DeleteStallTurnover(ctx context.Context, id, actor uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE stall_turnovers SET active=0,deleted_at=NOW(),deleted_by=? WHERE id=? AND active=1`, actor, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_id,target_text,detail) VALUES(?,?,?,?,?,?,?)`, actor, "stalls", "删除营业额", "stall_turnovers", id, "营业额记录", "档口月度营业额")
	return err
}

// PostStallCommissionLedger 将档口自营抽成写入收入台账；康莱德是外部抽成方，不计入本方收入。
func (s *Store) PostStallCommissionLedger(ctx context.Context, actor uint64, v request.StallLedgerInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始档口抽成入账事务失败")
	}
	defer tx.Rollback()
	var stallID uint64
	var stallName, responsible, mode string
	var rent, defaultGuaranteed, jiaRate, yiRate, guaranteedRate, baseRate, baseFactor, totalRate, card, huilaimi, cost, guaranteedAmount float64
	if err := tx.QueryRowContext(ctx, `SELECT s.id,s.name,s.responsible_name,COALESCE(r.allocation_mode,s.allocation_mode),COALESCE(r.monthly_rent,s.monthly_rent),COALESCE(r.default_guaranteed_amount,s.default_guaranteed_amount),COALESCE(r.profit_jia_rate,s.profit_jia_rate),COALESCE(r.profit_yi_rate,s.profit_yi_rate),COALESCE(r.guaranteed_rate,s.guaranteed_rate),COALESCE(r.base_amount_rate,s.base_amount_rate),COALESCE(r.base_amount_factor,s.base_amount_factor),COALESCE(r.total_amount_rate,s.total_amount_rate),COALESCE(t.card_amount,0),COALESCE(t.huilaimi_amount,0),COALESCE(t.cost_amount,0),COALESCE(t.guaranteed_amount,0) FROM stall_accounts s LEFT JOIN stall_turnovers t ON t.stall_id=s.id AND t.turnover_month=? AND t.active=1 LEFT JOIN stall_monthly_rules r ON r.stall_id=s.id AND r.turnover_month=? WHERE s.id=? AND s.active=1`, v.Month+"-01", v.Month+"-01", v.StallID).Scan(&stallID, &stallName, &responsible, &mode, &rent, &defaultGuaranteed, &jiaRate, &yiRate, &guaranteedRate, &baseRate, &baseFactor, &totalRate, &card, &huilaimi, &cost, &guaranteedAmount); err == sql.ErrNoRows {
		return ErrNotFound
	} else if err != nil {
		return apperror.Wrap(err, "读取档口抽成数据失败")
	}
	if guaranteedAmount == 0 {
		guaranteedAmount = defaultGuaranteed
	}
	var conradRate float64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(conrad_rate,0) FROM stall_commission_settings WHERE turnover_month=?`, v.Month+"-01").Scan(&conradRate); err != nil && err != sql.ErrNoRows {
		return apperror.Wrap(err, "读取康莱德抽成比例失败")
	}
	item := response.Stall{AllocationMode: mode, MonthlyRent: Decimal(rent), DefaultGuaranteedAmount: Decimal(defaultGuaranteed), ProfitJiaRate: stallRate(jiaRate), ProfitYiRate: stallRate(yiRate), GuaranteedRate: stallRate(guaranteedRate), BaseAmountRate: stallRate(baseRate), BaseAmountFactor: stallRate(baseFactor), TotalAmountRate: stallRate(totalRate), SchoolCardAmount: Decimal(card), HuilaimiAmount: Decimal(huilaimi), CostAmount: Decimal(cost), GuaranteedAmount: Decimal(guaranteedAmount)}
	calculateStall(&item, conradRate)
	amount := stallNumber(item.OwnCommission)
	if amount <= 0 {
		return apperror.New("当前档口本方抽成金额为0，无需写入台账")
	}
	var accountID uint64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM ledger_accounts WHERE id=? AND enabled=1 FOR UPDATE`, v.AccountID).Scan(&accountID); err == sql.ErrNoRows {
		return apperror.New("入账账户不存在或已停用")
	} else if err != nil {
		return apperror.Wrap(err, "读取入账账户失败")
	}
	var categoryID uint64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM ledger_categories WHERE direction='income' AND name='档口抽成' LIMIT 1`).Scan(&categoryID); err == sql.ErrNoRows {
		result, insertErr := tx.ExecContext(ctx, `INSERT INTO ledger_categories(direction,name,enabled,sort_order) VALUES('income','档口抽成',1,999)`)
		if insertErr != nil {
			return apperror.Wrap(insertErr, "创建档口抽成分类失败")
		}
		categoryID64, idErr := result.LastInsertId()
		if idErr != nil {
			return apperror.Wrap(idErr, "读取档口抽成分类失败")
		}
		categoryID = uint64(categoryID64)
	} else if err != nil {
		return apperror.Wrap(err, "读取档口抽成分类失败")
	}
	voucher := "STALL-" + strconv.FormatUint(stallID, 10) + "-" + v.Month
	summary := v.Month + "档口自营抽成-" + stallName
	remark := "本方抽成=" + item.OwnCommission + "；甲=" + item.JiaCommission + "，乙=" + item.YiCommission + "；康莱德另计=" + item.ConradCommission
	var entryID uint64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM ledger_entries WHERE voucher_no=? AND active=1 FOR UPDATE`, voucher).Scan(&entryID); err == sql.ErrNoRows {
		result, insertErr := tx.ExecContext(ctx, `INSERT INTO ledger_entries(occurred_on,account_id,category_id,direction,amount,summary,counterparty,voucher_no,remark) VALUES(?,?,?,'income',?,?,?,?,?)`, v.OccurredOn, accountID, categoryID, amount, summary, responsible, voucher, remark)
		if insertErr != nil {
			return apperror.Wrap(insertErr, "写入档口抽成台账失败")
		}
		id, idErr := result.LastInsertId()
		if idErr != nil {
			return apperror.Wrap(idErr, "读取档口抽成台账编号失败")
		}
		entryID = uint64(id)
	} else if err != nil {
		return apperror.Wrap(err, "查询档口抽成台账失败")
	} else if _, err := tx.ExecContext(ctx, `UPDATE ledger_entries SET occurred_on=?,account_id=?,category_id=?,amount=?,summary=?,counterparty=?,remark=? WHERE id=? AND active=1`, v.OccurredOn, accountID, categoryID, amount, summary, responsible, remark, entryID); err != nil {
		return apperror.Wrap(err, "更新档口抽成台账失败")
	}
	if err := miniAudit(ctx, tx, actor, "stalls", entryID, "档口自营抽成入账："+stallName); err != nil {
		return err
	}
	return tx.Commit()
}
