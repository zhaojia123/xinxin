-- 档口抽成模型升级：厦门理工知行2食堂。
-- 使用信息_schema判断列是否存在，可重复执行，不会因重复列报错。

SET @schema_name = DATABASE();

SELECT IF(COUNT(*)=0,'ALTER TABLE stall_accounts ADD COLUMN allocation_mode VARCHAR(24) NOT NULL DEFAULT ''custom'' AFTER turnover_rate','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_accounts' AND column_name='allocation_mode'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_accounts ADD COLUMN default_guaranteed_amount DECIMAL(14,2) NOT NULL DEFAULT 0 AFTER monthly_rent','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_accounts' AND column_name='default_guaranteed_amount'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_accounts ADD COLUMN profit_jia_rate DECIMAL(8,4) NOT NULL DEFAULT 0 AFTER allocation_mode','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_accounts' AND column_name='profit_jia_rate'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_accounts ADD COLUMN profit_yi_rate DECIMAL(8,4) NOT NULL DEFAULT 0 AFTER profit_jia_rate','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_accounts' AND column_name='profit_yi_rate'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_accounts ADD COLUMN guaranteed_rate DECIMAL(8,4) NOT NULL DEFAULT 0 AFTER profit_yi_rate','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_accounts' AND column_name='guaranteed_rate'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_accounts ADD COLUMN base_amount_rate DECIMAL(8,4) NOT NULL DEFAULT 0 AFTER guaranteed_rate','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_accounts' AND column_name='base_amount_rate'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_accounts ADD COLUMN base_amount_factor DECIMAL(8,4) NOT NULL DEFAULT 0 AFTER base_amount_rate','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_accounts' AND column_name='base_amount_factor'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_accounts ADD COLUMN total_amount_rate DECIMAL(8,4) NOT NULL DEFAULT 0 AFTER base_amount_factor','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_accounts' AND column_name='total_amount_rate'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT IF(COUNT(*)=0,'ALTER TABLE stall_turnovers ADD COLUMN cost_amount DECIMAL(14,2) NOT NULL DEFAULT 0 AFTER huilaimi_amount','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_turnovers' AND column_name='cost_amount'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_turnovers ADD COLUMN guaranteed_amount DECIMAL(14,2) NOT NULL DEFAULT 0 AFTER cost_amount','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_turnovers' AND column_name='guaranteed_amount'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS stall_commission_settings (
    turnover_month DATE NOT NULL,
    canteen_name VARCHAR(128) NOT NULL DEFAULT '厦门理工知行2食堂',
    conrad_rate DECIMAL(8,4) NOT NULL DEFAULT 0,
    remark VARCHAR(500) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (turnover_month)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='档口月度抽成设置';

CREATE TABLE IF NOT EXISTS stall_monthly_rules (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    stall_id BIGINT UNSIGNED NOT NULL,
    turnover_month DATE NOT NULL,
    monthly_rent DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '当月月租覆盖值',
    default_guaranteed_amount DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '当月保底金额覆盖值',
    allocation_mode VARCHAR(24) NOT NULL DEFAULT 'custom',
    profit_jia_rate DECIMAL(8,4) NOT NULL DEFAULT 0,
    profit_yi_rate DECIMAL(8,4) NOT NULL DEFAULT 0,
    guaranteed_rate DECIMAL(8,4) NOT NULL DEFAULT 0,
    base_amount_rate DECIMAL(8,4) NOT NULL DEFAULT 0,
    base_amount_factor DECIMAL(8,4) NOT NULL DEFAULT 0,
    total_amount_rate DECIMAL(8,4) NOT NULL DEFAULT 0,
    remark VARCHAR(500) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_stall_monthly_rules (stall_id, turnover_month),
    KEY idx_stall_monthly_rules_month (turnover_month),
    CONSTRAINT fk_stall_monthly_rules_stall FOREIGN KEY (stall_id) REFERENCES stall_accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='档口当月规则覆盖';

-- 不在可重复执行的迁移中自动回填比例，避免覆盖用户已经保存的档口规则。
-- 初始档口规则由导入脚本或页面维护。
