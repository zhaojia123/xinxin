-- 月度核算：水电、康莱德卡机抽点和营业收入确认入账；工资仅用于核算，仍由工资发放模块入账。
CREATE TABLE IF NOT EXISTS stall_monthly_expenses (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    expense_month DATE NOT NULL,
    water_amount DECIMAL(14,2) NOT NULL DEFAULT 0,
    electricity_amount DECIMAL(14,2) NOT NULL DEFAULT 0,
    manual_payroll_amount DECIMAL(14,2) NULL DEFAULT NULL,
    water_ledger_entry_id BIGINT UNSIGNED NULL,
    electricity_ledger_entry_id BIGINT UNSIGNED NULL,
    payroll_ledger_entry_id BIGINT UNSIGNED NULL,
    conrad_ledger_entry_id BIGINT UNSIGNED NULL,
    card_income_ledger_entry_id BIGINT UNSIGNED NULL,
    huilaimi_income_ledger_entry_id BIGINT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_stall_monthly_expenses_month (expense_month),
    CONSTRAINT fk_stall_monthly_expenses_water_ledger FOREIGN KEY (water_ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL,
    CONSTRAINT fk_stall_monthly_expenses_electricity_ledger FOREIGN KEY (electricity_ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL,
    CONSTRAINT fk_stall_monthly_expenses_payroll_ledger FOREIGN KEY (payroll_ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL,
    CONSTRAINT fk_stall_monthly_expenses_conrad_ledger FOREIGN KEY (conrad_ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL,
    CONSTRAINT fk_stall_monthly_expenses_card_income_ledger FOREIGN KEY (card_income_ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL,
    CONSTRAINT fk_stall_monthly_expenses_huilaimi_income_ledger FOREIGN KEY (huilaimi_income_ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='档口月度核算经营支出';

-- 老库已存在该表时补齐月度收入和康莱德支出关联列，可重复执行。
SET @schema_name = DATABASE();
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_monthly_expenses ADD COLUMN conrad_ledger_entry_id BIGINT UNSIGNED NULL AFTER payroll_ledger_entry_id, ADD CONSTRAINT fk_stall_monthly_expenses_conrad_ledger FOREIGN KEY (conrad_ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_monthly_expenses' AND column_name='conrad_ledger_entry_id'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_monthly_expenses ADD COLUMN card_income_ledger_entry_id BIGINT UNSIGNED NULL AFTER conrad_ledger_entry_id, ADD CONSTRAINT fk_stall_monthly_expenses_card_income_ledger FOREIGN KEY (card_income_ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_monthly_expenses' AND column_name='card_income_ledger_entry_id'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SELECT IF(COUNT(*)=0,'ALTER TABLE stall_monthly_expenses ADD COLUMN huilaimi_income_ledger_entry_id BIGINT UNSIGNED NULL AFTER card_income_ledger_entry_id, ADD CONSTRAINT fk_stall_monthly_expenses_huilaimi_income_ledger FOREIGN KEY (huilaimi_income_ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_monthly_expenses' AND column_name='huilaimi_income_ledger_entry_id'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

INSERT INTO ledger_categories(direction,name,enabled,sort_order) VALUES
    ('expense','水费',1,50),
    ('expense','电费',1,51),
    ('expense','康莱德抽点',1,52),
    ('income','卡机营业收入',1,53),
    ('income','汇来米营业收入',1,54)
ON DUPLICATE KEY UPDATE enabled=1;
