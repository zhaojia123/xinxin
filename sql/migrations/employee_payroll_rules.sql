-- 已有项目执行一次即可；如果从头执行 sql/schema.sql，则不需要再次执行本文件。
-- 作用：允许工资为空、记录月薪/时薪，并让请假/特殊休息参与工资计算。
SET @db_name = DATABASE();

SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE employees ADD COLUMN pay_basis ENUM(''monthly'',''daily'',''hourly'') NOT NULL DEFAULT ''monthly'' COMMENT ''月薪、日薪或时薪'' AFTER current_salary',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema=@db_name AND table_name='employees' AND column_name='pay_basis');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 月薪暂未填写的员工保留 NULL，后续补录工资时不会被误认为 0 元。
ALTER TABLE employees MODIFY current_salary DECIMAL(12,2) NULL COMMENT '当前工资、日薪或时薪，未填写时为空';
ALTER TABLE employees MODIFY pay_basis ENUM('monthly','daily','hourly') NOT NULL DEFAULT 'monthly' COMMENT '月薪、日薪或时薪';

SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE attendance_records ADD COLUMN salary_effect ENUM(''none'',''deduct'',''subsidy'',''deduct_and_subsidy'') NOT NULL DEFAULT ''none'' COMMENT ''工资影响方式'' AFTER duration_days',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema=@db_name AND table_name='attendance_records' AND column_name='salary_effect');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE attendance_records ADD COLUMN subsidy_amount DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT ''异常补助金额'' AFTER salary_effect',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema=@db_name AND table_name='attendance_records' AND column_name='subsidy_amount');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 同一条特殊休息可以同时产生“扣款”和“补助”两条工资明细。
-- 先创建新索引，再删除旧索引，避免旧索引仍被工资明细外键依赖而无法删除。
SET @has_new_key = (SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema=@db_name AND table_name='payroll_adjustments' AND index_name='uk_payroll_adjustment_attendance_type');
SET @sql = IF(@has_new_key = 0,
    'ALTER TABLE payroll_adjustments ADD UNIQUE KEY uk_payroll_adjustment_attendance_type (payroll_item_id, attendance_record_id, adjustment_type)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @has_old_key = (SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema=@db_name AND table_name='payroll_adjustments' AND index_name='uk_payroll_adjustment_attendance');
SET @sql = IF(@has_old_key > 0,
    'ALTER TABLE payroll_adjustments DROP INDEX uk_payroll_adjustment_attendance', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
