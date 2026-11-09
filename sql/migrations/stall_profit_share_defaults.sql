-- 公共利润分成规则：没有当月覆盖时，所有月份都自动使用该规则。
CREATE TABLE IF NOT EXISTS stall_profit_share_defaults (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    participant_name VARCHAR(64) NOT NULL COMMENT '参与方名称',
    share_rate DECIMAL(10,8) NOT NULL COMMENT '分成比例，小数保存，如0.8表示80%',
    sort_order INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '展示排序',
    active TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否生效',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_stall_profit_share_default_name (participant_name),
    KEY idx_stall_profit_share_default_active (active, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='档口公共利润分成参与方';

-- 首次升级时继承最近一个已设置月份的参与方，避免以后每月重复添加。
-- 用变量记录迁移前是否已有公共规则，重复执行迁移时不覆盖人工维护的公共规则。
SET @profit_share_default_count := (SELECT COUNT(*) FROM stall_profit_share_defaults);

INSERT INTO stall_profit_share_defaults (participant_name, share_rate, sort_order, active)
SELECT participant_name, share_rate, sort_order, active
FROM stall_profit_share_participants
WHERE share_month = (SELECT latest_month FROM (SELECT MAX(share_month) AS latest_month FROM stall_profit_share_participants WHERE active = 1) AS latest)
  AND active = 1
  AND @profit_share_default_count = 0;

-- 如没有历史月度设置，则提供兼容原先口径的默认参与方。
SET @profit_share_default_count := (SELECT COUNT(*) FROM stall_profit_share_defaults);

INSERT INTO stall_profit_share_defaults (participant_name, share_rate, sort_order, active)
SELECT '甲', 0.90000000, 1, 1 WHERE @profit_share_default_count = 0
UNION ALL
SELECT '乙', 0.10000000, 2, 1 WHERE @profit_share_default_count = 0;
