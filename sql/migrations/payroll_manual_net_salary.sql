-- 已有项目执行一次即可；从头执行 sql/schema.sql 则不需要再次执行。
-- 作用：允许个别已结算或特殊员工固定实发金额，并在重新生成工资时保留该金额。
SET @db_name = DATABASE();
SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE payroll_items ADD COLUMN manual_net_salary DECIMAL(12,2) NULL COMMENT ''人工固定实发金额，优先于自动计算'' AFTER net_salary',
    'SELECT 1')
    FROM information_schema.columns
    WHERE table_schema=@db_name AND table_name='payroll_items' AND column_name='manual_net_salary');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
