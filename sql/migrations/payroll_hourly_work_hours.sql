-- 已有项目执行一次即可；从头执行 sql/schema.sql 时已包含该字段。
SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE payroll_items ADD COLUMN work_hours DECIMAL(10,2) NOT NULL DEFAULT 0 AFTER employee_id',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'payroll_items' AND column_name = 'work_hours');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
