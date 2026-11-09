package mysql

import (
	"context"
	"database/sql"
	"strings"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/apperror"
)

func defaultProfitShares() []response.ProfitShare {
	return []response.ProfitShare{{Name: "甲", Rate: "90.00", SortOrder: 1}, {Name: "乙", Rate: "10.00", SortOrder: 2}}
}

// ProfitShares 返回指定月份生效的人员比例：当月覆盖优先，其次使用公共规则。
func (s *Store) ProfitShares(ctx context.Context, month string) ([]response.ProfitShare, error) {
	items, _, err := s.ProfitSharesWithSource(ctx, month)
	return items, err
}

// ProfitSharesWithSource 同时返回是否命中了当月专属配置，供页面提示当前设置来源。
func (s *Store) ProfitSharesWithSource(ctx context.Context, month string) ([]response.ProfitShare, bool, error) {
	if err := s.ready(); err != nil {
		return nil, false, err
	}
	monthly, err := s.profitShareRows(ctx, `SELECT id,participant_name,share_rate,sort_order FROM stall_profit_share_participants WHERE share_month=? AND active=1 ORDER BY sort_order,id`, NormalizeMonth(month)+"-01")
	if err != nil {
		// 数据库迁移尚未执行时保留旧的90/10口径，避免页面因扩展表缺失无法打开。
		if strings.Contains(err.Error(), "Error 1146") || strings.Contains(err.Error(), "doesn't exist") {
			return defaultProfitShares(), false, nil
		}
		return nil, false, apperror.Wrap(err, "读取当月分成参与方失败")
	}
	if len(monthly) > 0 {
		return monthly, true, nil
	}
	global, err := s.profitShareRows(ctx, `SELECT id,participant_name,share_rate,sort_order FROM stall_profit_share_defaults WHERE active=1 ORDER BY sort_order,id`)
	if err != nil {
		if strings.Contains(err.Error(), "Error 1146") || strings.Contains(err.Error(), "doesn't exist") {
			return defaultProfitShares(), false, nil
		}
		return nil, false, apperror.Wrap(err, "读取公共分成参与方失败")
	}
	if len(global) == 0 {
		return defaultProfitShares(), false, nil
	}
	return global, false, nil
}

func (s *Store) profitShareRows(ctx context.Context, query string, args ...any) ([]response.ProfitShare, error) {
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []response.ProfitShare{}
	for rows.Next() {
		var item response.ProfitShare
		var rate float64
		if err := rows.Scan(&item.ID, &item.Name, &rate, &item.SortOrder); err != nil {
			return nil, err
		}
		item.Rate = stallRate(rate)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// SaveProfitShares 保存公共规则或当月覆盖，读取时当月覆盖优先于公共规则。
func (s *Store) SaveProfitShares(ctx context.Context, actor uint64, input request.ProfitShareParticipantsInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始保存分成参与方事务失败")
	}
	defer tx.Rollback()
	if input.Scope == "global" {
		if err := replaceProfitShares(ctx, tx, `UPDATE stall_profit_share_defaults SET active=0 WHERE active=1`, `INSERT INTO stall_profit_share_defaults(participant_name,share_rate,sort_order,active) VALUES(?,?,?,1) ON DUPLICATE KEY UPDATE share_rate=VALUES(share_rate),sort_order=VALUES(sort_order),active=1`, input.Participants); err != nil {
			return err
		}
	} else if err := replaceProfitShares(ctx, tx, `UPDATE stall_profit_share_participants SET active=0 WHERE share_month=? AND active=1`, `INSERT INTO stall_profit_share_participants(share_month,participant_name,share_rate,sort_order,active) VALUES(?,?,?,?,1) ON DUPLICATE KEY UPDATE share_rate=VALUES(share_rate),sort_order=VALUES(sort_order),active=1`, input.Participants, input.Month+"-01"); err != nil {
		return err
	}
	targetType, targetText := "stall_profit_share_defaults", "公共规则"
	if input.Scope == "monthly" {
		targetType, targetText = "stall_profit_share_participants", input.Month
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_text,detail) VALUES(?,?,?,?,?,?)`, actor, "stalls", "保存利润分成", targetType, targetText, "分成比例合计100%"); err != nil {
		return apperror.Wrap(err, "记录分成设置日志失败")
	}
	return tx.Commit()
}

func replaceProfitShares(ctx context.Context, tx *sql.Tx, deactivateSQL, insertSQL string, participants []request.ProfitShareInput, args ...any) error {
	if _, err := tx.ExecContext(ctx, deactivateSQL, args...); err != nil {
		return apperror.Wrap(err, "清理原分成参与方失败")
	}
	for index, item := range participants {
		values := []any{item.Name, item.Rate, index + 1}
		if len(args) > 0 {
			values = append([]any{args[0]}, values...)
		}
		if _, err := tx.ExecContext(ctx, insertSQL, values...); err != nil {
			return apperror.Wrap(err, "保存分成参与方失败")
		}
	}
	return nil
}

// DeleteMonthlyProfitShares 删除当月覆盖，让该月重新使用公共分成规则。
func (s *Store) DeleteMonthlyProfitShares(ctx context.Context, actor uint64, month string) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始恢复公共分成事务失败")
	}
	defer tx.Rollback()
	month = NormalizeMonth(month)
	if _, err := tx.ExecContext(ctx, `UPDATE stall_profit_share_participants SET active=0 WHERE share_month=? AND active=1`, month+"-01"); err != nil {
		return apperror.Wrap(err, "清理当月分成覆盖失败")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_text,detail) VALUES(?,?,?,?,?,?)`, actor, "stalls", "恢复公共利润分成", "stall_profit_share_participants", month, "删除当月覆盖，改用公共规则"); err != nil {
		return apperror.Wrap(err, "记录恢复公共分成日志失败")
	}
	return tx.Commit()
}

// ApplyProfitShares 将当月结余按参与方比例拆分；亏损月份同时给出每位参与方的承担金额。
func ApplyProfitShares(summary *response.StallSummary, shares []response.ProfitShare) {
	if len(shares) == 0 {
		shares = defaultProfitShares()
	}
	amount := stallNumber(summary.DistributableBalance)
	summary.ProfitShares = make([]response.ProfitShare, 0, len(shares))
	summary.JiaCommission, summary.YiCommission = "0.00", "0.00"
	for _, item := range shares {
		shareAmount := amount * stallNumber(item.Rate) / 100
		item.Amount, item.LossAmount = "0.00", "0.00"
		if amount < 0 {
			item.LossAmount = Decimal(-shareAmount)
		} else {
			item.Amount = Decimal(shareAmount)
		}
		if amount >= 0 && item.Name == "甲" {
			summary.JiaCommission = item.Amount
		}
		if amount >= 0 && item.Name == "乙" {
			summary.YiCommission = item.Amount
		}
		summary.ProfitShares = append(summary.ProfitShares, item)
	}
}
