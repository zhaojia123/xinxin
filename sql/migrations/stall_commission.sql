-- 档口管理与月度营业额抽成。负责人信息直接保存在档口表，不单独建人员表。
CREATE TABLE IF NOT EXISTS stall_accounts (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    stall_no VARCHAR(64) NOT NULL COMMENT '档口编号',
    name VARCHAR(128) NOT NULL COMMENT '档口名称',
    responsible_name VARCHAR(64) NOT NULL DEFAULT '' COMMENT '档口负责人',
    responsible_mobile VARCHAR(32) NOT NULL DEFAULT '' COMMENT '负责人电话，可为空',
    monthly_rent DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '月租',
    commission_rate DECIMAL(8,4) NOT NULL DEFAULT 0 COMMENT '抽成比例，0.22表示22%',
    turnover_rate DECIMAL(8,4) NOT NULL DEFAULT 0 COMMENT '营业额比例，0.22表示22%',
    remark VARCHAR(500) NOT NULL DEFAULT '',
    active TINYINT(1) NOT NULL DEFAULT 1,
    deleted_at DATETIME NULL,
    deleted_by BIGINT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_stall_accounts_no (stall_no),
    KEY idx_stall_accounts_active (active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='档口基础资料';

CREATE TABLE IF NOT EXISTS stall_turnovers (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    stall_id BIGINT UNSIGNED NOT NULL,
    turnover_month DATE NOT NULL COMMENT '月份第一天',
    school_amount DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '学校统计营业额',
    card_amount DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '卡机营业额',
    huilaimi_amount DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '惠来米营业额',
    remark VARCHAR(500) NOT NULL DEFAULT '',
    active TINYINT(1) NOT NULL DEFAULT 1,
    deleted_at DATETIME NULL,
    deleted_by BIGINT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_stall_turnovers_month (stall_id, turnover_month),
    KEY idx_stall_turnovers_month (turnover_month, active),
    CONSTRAINT fk_stall_turnovers_stall FOREIGN KEY (stall_id) REFERENCES stall_accounts(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='档口月度营业额';
