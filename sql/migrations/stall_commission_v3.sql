-- 档口抽成扩展：固定抽成金额，以及仅自营分成档口使用的成本明细。
SET @schema_name = DATABASE();

SELECT IF(COUNT(*)=0,'ALTER TABLE stall_accounts ADD COLUMN fixed_commission_amount DECIMAL(14,2) NOT NULL DEFAULT 0 AFTER monthly_rent','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_accounts' AND column_name='fixed_commission_amount'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_monthly_rules ADD COLUMN fixed_commission_amount DECIMAL(14,2) NOT NULL DEFAULT 0 AFTER monthly_rent','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_monthly_rules' AND column_name='fixed_commission_amount'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS stall_cost_entries (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    stall_id BIGINT UNSIGNED NOT NULL,
    cost_month DATE NOT NULL,
    cost_type VARCHAR(32) NOT NULL DEFAULT 'other',
    all_self_operated TINYINT(1) NOT NULL DEFAULT 0,
    amount DECIMAL(14,2) NOT NULL DEFAULT 0,
    remark VARCHAR(255) NOT NULL DEFAULT '',
    ledger_entry_id BIGINT UNSIGNED NULL,
    active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_stall_cost_entries_month (cost_month, active),
    KEY idx_stall_cost_entries_stall_month (stall_id, cost_month, active),
    CONSTRAINT fk_stall_cost_entries_stall FOREIGN KEY (stall_id) REFERENCES stall_accounts(id) ON DELETE RESTRICT,
    CONSTRAINT fk_stall_cost_entries_ledger FOREIGN KEY (ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='自营档口月度成本明细';

SELECT IF(COUNT(*)=0,'ALTER TABLE stall_cost_entries ADD COLUMN all_self_operated TINYINT(1) NOT NULL DEFAULT 0 AFTER cost_type','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_cost_entries' AND column_name='all_self_operated'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 将旧月度核算中已经填写的自营档口成本迁入明细表，避免升级后丢失九月历史成本。
INSERT INTO stall_cost_entries(stall_id,cost_month,cost_type,amount,remark)
SELECT t.stall_id,t.turnover_month,'other',t.cost_amount,'由旧月度核算成本迁移'
FROM stall_turnovers t
JOIN stall_accounts s ON s.id=t.stall_id AND s.active=1 AND s.allocation_mode='profit_share'
WHERE t.active=1 AND t.cost_amount>0
  AND NOT EXISTS (
    SELECT 1 FROM stall_cost_entries c
    WHERE c.stall_id=t.stall_id AND c.cost_month=t.turnover_month AND c.active=1
  );
