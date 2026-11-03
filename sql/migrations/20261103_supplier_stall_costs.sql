-- 供货商对账与档口成本拆分：旧 amount 继续表示食材费用，米油单独保存。
-- 本脚本可重复执行，升级已有数据库时不影响历史成本。
SET @schema_name = DATABASE();

CREATE TABLE IF NOT EXISTS supplier_accounts (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(64) NOT NULL,
    contact_name VARCHAR(64) NOT NULL DEFAULT '',
    mobile VARCHAR(32) NOT NULL DEFAULT '',
    remark VARCHAR(255) NOT NULL DEFAULT '',
    active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_supplier_accounts_name (name),
    KEY idx_supplier_accounts_active (active, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='供货商资料';

SELECT IF(COUNT(*)=0,'ALTER TABLE stall_cost_entries ADD COLUMN rice_oil_amount DECIMAL(14,2) NOT NULL DEFAULT 0 AFTER amount','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_cost_entries' AND column_name='rice_oil_amount'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_cost_entries ADD COLUMN supplier_name VARCHAR(64) NOT NULL DEFAULT '''' AFTER rice_oil_amount','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_cost_entries' AND column_name='supplier_name'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_cost_entries ADD KEY idx_stall_cost_entries_supplier_month (supplier_name,cost_month,active)','SELECT 1') INTO @sql FROM information_schema.statistics WHERE table_schema=@schema_name AND table_name='stall_cost_entries' AND index_name='idx_stall_cost_entries_supplier_month'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
