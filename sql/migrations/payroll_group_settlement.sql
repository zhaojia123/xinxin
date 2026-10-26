-- 工资按人员类型汇总入账：正式工、临时工、特殊员工每月各一条流水。
CREATE TABLE IF NOT EXISTS payroll_settlement_entries (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '工资汇总入账ID',
    payroll_month DATE NOT NULL,
    payroll_type ENUM('full_time','part_time','special') NOT NULL,
    ledger_entry_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_payroll_settlement_month_type (payroll_month, payroll_type),
    UNIQUE KEY uk_payroll_settlement_ledger (ledger_entry_id),
    CONSTRAINT fk_payroll_settlement_ledger FOREIGN KEY (ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工资按人员类型汇总入账关联';

CREATE TABLE IF NOT EXISTS special_payroll_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '特殊员工工资ID',
    payroll_month DATE NOT NULL,
    employee_name VARCHAR(64) NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    remark VARCHAR(500) NOT NULL DEFAULT '',
    status ENUM('draft','paid') NOT NULL DEFAULT 'draft',
    ledger_entry_id BIGINT UNSIGNED NULL,
    paid_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_special_payroll_month (payroll_month, status),
    KEY idx_special_payroll_ledger (ledger_entry_id),
    CONSTRAINT fk_special_payroll_ledger FOREIGN KEY (ledger_entry_id) REFERENCES ledger_entries(id) ON DELETE SET NULL,
    CONSTRAINT chk_special_payroll_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='未建档特殊员工工资明细';
