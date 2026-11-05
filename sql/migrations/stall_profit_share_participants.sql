-- 月度利润分成参与方。每个月独立保存，修改未来月份不会重算历史月份。
CREATE TABLE IF NOT EXISTS stall_profit_share_participants (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    share_month DATE NOT NULL COMMENT '生效月份，每月1日',
    participant_name VARCHAR(64) NOT NULL COMMENT '参与方名称',
    share_rate DECIMAL(10,8) NOT NULL COMMENT '分成比例，小数保存，如0.9表示90%',
    sort_order INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '展示排序',
    active TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否生效',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_stall_profit_share_month_name (share_month, participant_name),
    KEY idx_stall_profit_share_month_active (share_month, active, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='档口月度利润分成参与方';

-- 初始化已存在历史月份的默认比例；重复执行不会覆盖后续人工设置。
INSERT IGNORE INTO stall_profit_share_participants (share_month, participant_name, share_rate, sort_order, active)
SELECT turnover_month, '甲', 0.90000000, 1, 1
FROM stall_turnovers
GROUP BY turnover_month;

INSERT IGNORE INTO stall_profit_share_participants (share_month, participant_name, share_rate, sort_order, active)
SELECT turnover_month, '乙', 0.10000000, 2, 1
FROM stall_turnovers
GROUP BY turnover_month;
