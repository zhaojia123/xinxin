package mysql

import (
	"context"
	"strings"

	"friends-records/api/request"
	"friends-records/api/response"
	"friends-records/internal/apperror"
)

func defaultProfitShares() []response.ProfitShare {
	return []response.ProfitShare{{Name: "甲", Rate: "90.00", SortOrder: 1}, {Name: "乙", Rate: "10.00", SortOrder: 2}}
}

// ProfitShares 返回指定月份的人员比例；未配置时兼容原有甲90%、乙10%。
func (s *Store) ProfitShares(ctx context.Context, month string) ([]response.ProfitShare, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,participant_name,share_rate,sort_order FROM stall_profit_share_participants WHERE share_month=? AND active=1 ORDER BY sort_order,id`, NormalizeMonth(month)+"-01")
	if err != nil {
		// 数据库迁移尚未执行时保留旧的90/10口径，避免页面因扩展表缺失无法打开。
		if strings.Contains(err.Error(), "Error 1146") || strings.Contains(err.Error(), "doesn't exist") {
			return defaultProfitShares(), nil
		}
		return nil, apperror.Wrap(err, "读取分成参与方失败")
	}
	defer rows.Close()
	items := []response.ProfitShare{}
	for rows.Next() {
		var item response.ProfitShare
		var rate float64
		if err := rows.Scan(&item.ID, &item.Name, &rate, &item.SortOrder); err != nil {
			return nil, apperror.Wrap(err, "读取分成参与方失败")
		}
		item.Rate = stallRate(rate)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历分成参与方失败")
	}
	if len(items) == 0 {
		return defaultProfitShares(), nil
	}
	return items, nil
}

// SaveProfitShares 替换指定月份的分成参与方，先完成校验再在一个事务内生效。
func (s *Store) SaveProfitShares(ctx context.Context, actor uint64, input request.ProfitShareParticipantsInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return apperror.Wrap(err, "开始保存分成参与方事务失败")
	}
	defer tx.Rollback()
	month := input.Month + "-01"
	if _, err := tx.ExecContext(ctx, `UPDATE stall_profit_share_participants SET active=0 WHERE share_month=? AND active=1`, month); err != nil {
		return apperror.Wrap(err, "清理原分成参与方失败")
	}
	for index, item := range input.Participants {
		if _, err := tx.ExecContext(ctx, `INSERT INTO stall_profit_share_participants(share_month,participant_name,share_rate,sort_order,active) VALUES(?,?,?,?,1) ON DUPLICATE KEY UPDATE share_rate=VALUES(share_rate),sort_order=VALUES(sort_order),active=1`, month, item.Name, item.Rate, index+1); err != nil {
			return apperror.Wrap(err, "保存分成参与方失败")
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs(admin_user_id,module,action,target_type,target_text,detail) VALUES(?,?,?,?,?,?)`, actor, "stalls", "保存利润分成", "stall_profit_share_participants", input.Month, "分成比例合计100%"); err != nil {
		return apperror.Wrap(err, "记录分成设置日志失败")
	}
	return tx.Commit()
}

// ApplyProfitShares 将当月可分配余额按参与方比例拆分；亏损月份所有参与方金额为0。
func ApplyProfitShares(summary *response.StallSummary, shares []response.ProfitShare) {
	if len(shares) == 0 {
		shares = defaultProfitShares()
	}
	amount := stallNumber(summary.DistributableBalance)
	if amount < 0 {
		amount = 0
	}
	summary.ProfitShares = make([]response.ProfitShare, 0, len(shares))
	summary.JiaCommission, summary.YiCommission = "0.00", "0.00"
	for _, item := range shares {
		item.Amount = Decimal(amount * stallNumber(item.Rate) / 100)
		if item.Name == "甲" {
			summary.JiaCommission = item.Amount
		}
		if item.Name == "乙" {
			summary.YiCommission = item.Amount
		}
		summary.ProfitShares = append(summary.ProfitShares, item)
	}
}
