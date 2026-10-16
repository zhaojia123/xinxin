-- 小时工每日工时记录；已有项目执行一次即可。
CREATE TABLE IF NOT EXISTS employee_work_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    employee_id BIGINT UNSIGNED NOT NULL COMMENT '小时工员工ID',
    work_date DATE NOT NULL COMMENT '工作日期',
    hours DECIMAL(6,2) NOT NULL COMMENT '当天工时，半天可填4',
    remark VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
    active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    deleted_by BIGINT UNSIGNED NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_employee_work_date (employee_id, work_date),
    KEY idx_work_logs_date (work_date),
    CONSTRAINT fk_work_logs_employee FOREIGN KEY (employee_id) REFERENCES employees (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小时工每日工时记录';
