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

// SpecialStalls 返回自营档口，用于“全部自营档口”成本的默认归属。
func (s *Store) SpecialStalls(ctx context.Context) ([]response.Stall, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,stall_no,name FROM stall_accounts WHERE active=1 AND allocation_mode='profit_share' ORDER BY id`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询自营档口失败")
	}
	defer rows.Close()
	items := make([]response.Stall, 0)
	for rows.Next() {
		var item response.Stall
		if err := rows.Scan(&item.ID, &item.StallNo, &item.Name); err != nil {
			return nil, apperror.Wrap(err, "读取自营档口失败")
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// CostStalls 返回全部档口，供成本页面按自营和非自营分别维护。
func (s *Store) CostStalls(ctx context.Context) ([]response.Stall, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,stall_no,name,allocation_mode FROM stall_accounts WHERE active=1 ORDER BY allocation_mode='profit_share' DESC,id`)
	if err != nil {
		return nil, apperror.Wrap(err, "查询档口成本选项失败")
	}
	defer rows.Close()
	items := make([]response.Stall, 0)
	for rows.Next() {
		var item response.Stall
		if err := rows.Scan(&item.ID, &item.StallNo, &item.Name, &item.AllocationMode); err != nil {
			return nil, apperror.Wrap(err, "读取档口成本选项失败")
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// StallCosts 查询指定月份的自营和非自营档口成本。
func (s *Store) StallCosts(ctx context.Context, month, keyword string) ([]response.StallCostEntry, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	query := `SELECT c.id,DATE_FORMAT(c.cost_month,'%Y-%m'),c.stall_id,s.name,s.allocation_mode,c.cost_type,c.all_self_operated,c.amount,c.rice_oil_amount,c.supplier_name,c.remark,c.ledger_entry_id
		FROM stall_cost_entries c JOIN stall_accounts s ON s.id=c.stall_id
		WHERE c.active=1 AND c.cost_month=?`
	args := []any{NormalizeMonth(month) + "-01"}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		query += ` AND (s.name LIKE ? OR c.supplier_name LIKE ? OR c.cost_type LIKE ? OR c.remark LIKE ?)`
		like := "%" + keyword + "%"
		args = append(args, like, like, like, like)
	}
	query += ` ORDER BY c.cost_month DESC,c.id DESC`
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperror.Wrap(err, "查询档口成本失败")
	}
	defer rows.Close()
	items := make([]response.StallCostEntry, 0)
	for rows.Next() {
		var item response.StallCostEntry
		var ledgerID sql.NullInt64
		var mode string
		if err := rows.Scan(&item.ID, &item.Month, &item.StallID, &item.StallName, &mode, &item.CostType, &item.AllSelfOperated, &item.IngredientAmount, &item.RiceOilAmount, &item.SupplierName, &item.Remark, &ledgerID); err != nil {
			return nil, apperror.Wrap(err, "读取档口成本失败")
		}
		item.Amount = item.IngredientAmount
		item.TotalAmount = stallCostTotal(item.IngredientAmount, item.RiceOilAmount)
		item.Seq = len(items) + 1
		item.LedgerPosted = ledgerID.Valid
		item.SelfOperated = mode == "profit_share"
		if item.AllSelfOperated {
			item.StallName = "全部自营档口"
			item.SelfOperated = true
		}
		item.CostTypeLabel = stallCostTypeLabel(item.CostType)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历档口成本失败")
	}
	return items, nil
}

// stallCostTotal 统一计算食材和米油合计，避免月度核算、列表和入账出现不同金额。
func stallCostTotal(ingredient, riceOil string) string {
	food, _ := strconv.ParseFloat(ingredient, 64)
	rice, _ := strconv.ParseFloat(riceOil, 64)
	return Decimal(food + rice)
}

func stallCostTypeLabel(value string) string {
	switch value {
	case "ingredient":
		return "食材成本"
	case "rice_oil":
		return "米油成本"
	case "water":
		return "水费"
	case "electricity":
		return "电费"
	case "labor":
		return "人工"
	default:
		return "其他"
	}
}

// SaveStallCost 保存档口成本；非自营成本只记录和入账，不会参与甲乙分配。
func (s *Store) SaveStallCost(ctx context.Context, id, actor uint64, input request.StallCostInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存档口成本事务失败")
	}
	defer tx.Rollback()
	var name string
	if input.AllSelfOperated {
		if err := tx.QueryRowContext(ctx, `SELECT id FROM stall_accounts WHERE active=1 AND allocation_mode='profit_share' ORDER BY id LIMIT 1`).Scan(&input.StallID); err == sql.ErrNoRows {
			return 0, MiniInputError{"请先设置至少一个自营档口"}
		} else if err != nil {
			return 0, err
		}
		name = "全部自营档口"
	} else if err := tx.QueryRowContext(ctx, `SELECT name FROM stall_accounts WHERE id=? AND active=1`, input.StallID).Scan(&name); err == sql.ErrNoRows {
		return 0, ErrNotFound
	} else if err != nil {
		return 0, err
	}
	if id == 0 {
		result, err := tx.ExecContext(ctx, `INSERT INTO stall_cost_entries(stall_id,cost_month,cost_type,all_self_operated,amount,rice_oil_amount,supplier_name,remark) VALUES(?,?,?,?,?,?,?,?)`, input.StallID, input.Month+"-01", input.CostType, input.AllSelfOperated, input.IngredientAmount, input.RiceOilAmount, input.SupplierName, input.Remark)
		if err != nil {
			return 0, apperror.Wrap(err, "新增档口成本失败")
		}
		last, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}
		id = uint64(last)
	} else {
		result, err := tx.ExecContext(ctx, `UPDATE stall_cost_entries SET stall_id=?,cost_month=?,cost_type=?,all_self_operated=?,amount=?,rice_oil_amount=?,supplier_name=?,remark=? WHERE id=? AND active=1 AND ledger_entry_id IS NULL`, input.StallID, input.Month+"-01", input.CostType, input.AllSelfOperated, input.IngredientAmount, input.RiceOilAmount, input.SupplierName, input.Remark, id)
		if err != nil {
			return 0, apperror.Wrap(err, "修改档口成本失败")
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return 0, MiniInputError{"成本已入账或不存在，不能修改"}
		}
	}
	if err := miniAudit(ctx, tx, actor, "stall_costs", id, "档口成本："+name); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// SaveStallCostSpecialAdjustment 将一个档口当月的手动特殊调整维护为单个汇总值。
// 供货商对账数据不会受影响；已入账的调整不能改动，避免台账与成本金额不一致。
func (s *Store) SaveStallCostSpecialAdjustment(ctx context.Context, actor uint64, input request.StallCostSpecialAdjustmentInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存特殊调整事务失败")
	}
	defer tx.Rollback()

	var stallName string
	if err := tx.QueryRowContext(ctx, `SELECT name FROM stall_accounts WHERE id=? AND active=1`, input.StallID).Scan(&stallName); err == sql.ErrNoRows {
		return 0, ErrNotFound
	} else if err != nil {
		return 0, apperror.Wrap(err, "读取档口失败")
	}

	rows, err := tx.QueryContext(ctx, `SELECT id,ledger_entry_id FROM stall_cost_entries WHERE stall_id=? AND cost_month=? AND supplier_name='' AND all_self_operated=0 AND active=1 FOR UPDATE`, input.StallID, input.Month+"-01")
	if err != nil {
		return 0, apperror.Wrap(err, "读取特殊调整失败")
	}
	defer rows.Close()
	ids := make([]uint64, 0, 1)
	for rows.Next() {
		var id uint64
		var ledgerID sql.NullInt64
		if err := rows.Scan(&id, &ledgerID); err != nil {
			return 0, apperror.Wrap(err, "读取特殊调整失败")
		}
		if ledgerID.Valid {
			return 0, MiniInputError{"该档口已有已入账的特殊调整，不能直接修改"}
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, apperror.Wrap(err, "遍历特殊调整失败")
	}
	if err := rows.Close(); err != nil {
		return 0, apperror.Wrap(err, "关闭特殊调整失败")
	}

	if len(ids) > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE stall_cost_entries SET active=0 WHERE stall_id=? AND cost_month=? AND supplier_name='' AND all_self_operated=0 AND active=1`, input.StallID, input.Month+"-01"); err != nil {
			return 0, apperror.Wrap(err, "清空旧特殊调整失败")
		}
	}
	amount := stallNumber(input.Amount)
	if amount == 0 {
		if len(ids) > 0 {
			if err := miniAudit(ctx, tx, actor, "stall_costs", ids[0], "清空档口特殊调整："+stallName); err != nil {
				return 0, err
			}
		}
		return 0, tx.Commit()
	}

	result, err := tx.ExecContext(ctx, `INSERT INTO stall_cost_entries(stall_id,cost_month,cost_type,all_self_operated,amount,rice_oil_amount,supplier_name,remark) VALUES(?,?,?,0,?,?,?,?)`, input.StallID, input.Month+"-01", "other", input.Amount, "0", "", "档口成本直接调整")
	if err != nil {
		return 0, apperror.Wrap(err, "保存特殊调整失败")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, apperror.Wrap(err, "读取特殊调整编号失败")
	}
	if err := miniAudit(ctx, tx, actor, "stall_costs", uint64(id), "更新档口特殊调整："+stallName); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return uint64(id), nil
}

// SaveSupplierMatrixCost 保存供货商-档口矩阵的一个单元格；清空两个金额会删除未入账单元格。
func (s *Store) SaveSupplierMatrixCost(ctx context.Context, actor uint64, input request.SupplierMatrixCostInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存供货商对账事务失败")
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM supplier_accounts WHERE name=? AND active=1`, input.SupplierName).Scan(&exists); err == sql.ErrNoRows {
		return 0, MiniInputError{"供货商不存在或已停用"}
	} else if err != nil {
		return 0, apperror.Wrap(err, "校验供货商失败")
	}
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM stall_accounts WHERE id=? AND active=1`, input.StallID).Scan(&exists); err == sql.ErrNoRows {
		return 0, ErrNotFound
	} else if err != nil {
		return 0, apperror.Wrap(err, "校验档口失败")
	}

	rows, err := tx.QueryContext(ctx, `SELECT id,ledger_entry_id FROM stall_cost_entries WHERE stall_id=? AND cost_month=? AND supplier_name=? AND active=1 FOR UPDATE`, input.StallID, input.Month+"-01", input.SupplierName)
	if err != nil {
		return 0, apperror.Wrap(err, "读取供货商档口对账失败")
	}
	defer rows.Close()
	type existingCost struct {
		id     uint64
		posted bool
	}
	existing := make([]existingCost, 0, 1)
	for rows.Next() {
		var id uint64
		var ledgerID sql.NullInt64
		if err := rows.Scan(&id, &ledgerID); err != nil {
			return 0, apperror.Wrap(err, "读取供货商档口对账失败")
		}
		existing = append(existing, existingCost{id: id, posted: ledgerID.Valid})
	}
	if err := rows.Err(); err != nil {
		return 0, apperror.Wrap(err, "遍历供货商档口对账失败")
	}
	if err := rows.Close(); err != nil {
		return 0, apperror.Wrap(err, "关闭供货商档口对账失败")
	}
	if len(existing) > 1 {
		return 0, MiniInputError{"该供货商和档口存在多条历史对账，请先在档口成本页整理后再编辑"}
	}

	ingredient := stallNumber(input.IngredientAmount)
	riceOil := stallNumber(input.RiceOilAmount)
	total := ingredient + riceOil
	if len(existing) == 0 {
		if total == 0 {
			return 0, tx.Commit()
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO stall_cost_entries(stall_id,cost_month,cost_type,all_self_operated,amount,rice_oil_amount,supplier_name,remark) VALUES(?,?,?,0,?,?,?,?)`, input.StallID, input.Month+"-01", "ingredient", input.IngredientAmount, input.RiceOilAmount, input.SupplierName, "供货商对账矩阵录入")
		if err != nil {
			return 0, apperror.Wrap(err, "新增供货商档口对账失败")
		}
		lastID, err := result.LastInsertId()
		if err != nil {
			return 0, apperror.Wrap(err, "读取供货商档口对账编号失败")
		}
		if err := miniAudit(ctx, tx, actor, "supplier_costs", uint64(lastID), "供货商对账："+input.SupplierName); err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return uint64(lastID), nil
	}

	if existing[0].posted {
		return 0, MiniInputError{"该对账已入账，不能修改"}
	}
	if total == 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE stall_cost_entries SET active=0 WHERE id=?`, existing[0].id); err != nil {
			return 0, apperror.Wrap(err, "清空供货商档口对账失败")
		}
		if err := miniAudit(ctx, tx, actor, "supplier_costs", existing[0].id, "清空供货商对账："+input.SupplierName); err != nil {
			return 0, err
		}
		return 0, tx.Commit()
	}
	if _, err := tx.ExecContext(ctx, `UPDATE stall_cost_entries SET amount=?,rice_oil_amount=? WHERE id=?`, input.IngredientAmount, input.RiceOilAmount, existing[0].id); err != nil {
		return 0, apperror.Wrap(err, "更新供货商档口对账失败")
	}
	if err := miniAudit(ctx, tx, actor, "supplier_costs", existing[0].id, "更新供货商对账："+input.SupplierName); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return existing[0].id, nil
}

// DeleteStallCost 软删除未入账的成本，保留已入账支出的可追溯关系。
func (s *Store) DeleteStallCost(ctx context.Context, id, actor uint64) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var name string
	if err := tx.QueryRowContext(ctx, `SELECT s.name FROM stall_cost_entries c JOIN stall_accounts s ON s.id=c.stall_id WHERE c.id=? AND c.active=1 AND c.ledger_entry_id IS NULL FOR UPDATE`, id).Scan(&name); err == sql.ErrNoRows {
		return MiniInputError{"成本已入账或不存在，不能删除"}
	} else if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE stall_cost_entries SET active=0 WHERE id=?`, id); err != nil {
		return err
	}
	if err := miniAudit(ctx, tx, actor, "stall_costs", id, "删除档口成本："+name); err != nil {
		return err
	}
	return tx.Commit()
}

// PostStallCostLedger 将一条自营档口成本写为支出流水，入账后锁定该明细。
func (s *Store) PostStallCostLedger(ctx context.Context, actor uint64, input request.StallCostLedgerInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始档口成本入账事务失败")
	}
	defer tx.Rollback()
	if err := requirePayrollAccount(ctx, tx, input.AccountID); err != nil {
		return err
	}
	var month, name, costType, remark string
	var amount float64
	var ledgerID sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT DATE_FORMAT(c.cost_month,'%Y-%m'),s.name,c.cost_type,c.amount+c.rice_oil_amount,c.remark,c.ledger_entry_id FROM stall_cost_entries c JOIN stall_accounts s ON s.id=c.stall_id WHERE c.id=? AND c.active=1 FOR UPDATE`, input.CostID).Scan(&month, &name, &costType, &amount, &remark, &ledgerID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return apperror.Wrap(err, "读取档口成本失败")
	}
	if ledgerID.Valid {
		return MiniInputError{"该成本已计入台账，不能重复入账"}
	}
	if amount <= 0 {
		return MiniInputError{"成本金额为0，无需入账"}
	}
	categoryID, err := expenseCategory(ctx, tx, "档口成本")
	if err != nil {
		return apperror.Wrap(err, "创建档口成本分类失败")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries(occurred_on,account_id,category_id,direction,amount,summary,counterparty,operator_id,remark) VALUES(?,?,?,'expense',?,?,?,?,?)`, input.OccurredOn, input.AccountID, categoryID, amount, month+"档口成本-"+name, name, actor, stallCostTypeLabel(costType)+"："+remark)
	if err != nil {
		return apperror.Wrap(err, "写入档口成本台账失败")
	}
	entryID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE stall_cost_entries SET ledger_entry_id=? WHERE id=?`, entryID, input.CostID); err != nil {
		return err
	}
	if err := miniAudit(ctx, tx, actor, "stall_costs", input.CostID, "档口成本入账："+name); err != nil {
		return err
	}
	return tx.Commit()
}
