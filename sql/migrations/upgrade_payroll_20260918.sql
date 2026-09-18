-- 工资功能升级（已有项目执行本文件一次即可）
-- 适用于已经执行过基础表结构和 employee_payroll_rules.sql 的数据库。
-- 本文件可重复执行，已存在的字段和索引会自动跳过。

SET @db_name = DATABASE();

-- 1. 保存员工入职时工资快照，与当前工资区分。
SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE employees ADD COLUMN entry_salary DECIMAL(12,2) NULL COMMENT ''入职时工资或时薪，作为历史快照'' AFTER left_on',
  'SELECT 1')
  FROM information_schema.columns
  WHERE table_schema=@db_name AND table_name='employees' AND column_name='entry_salary');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 旧员工没有入职快照时，用当前工资初始化；以后调薪不会覆盖它。
UPDATE employees
SET entry_salary=current_salary
WHERE entry_salary IS NULL AND current_salary IS NOT NULL;

-- 2. 记录每条工资明细对应的发放流水，支持单独发放并防止重复入账。
SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE payroll_items ADD COLUMN ledger_entry_id BIGINT UNSIGNED NULL COMMENT ''该员工工资发放关联的台账流水ID'' AFTER status',
  'SELECT 1')
  FROM information_schema.columns
  WHERE table_schema=@db_name AND table_name='payroll_items' AND column_name='ledger_entry_id');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE payroll_items ADD COLUMN paid_at DATETIME NULL COMMENT ''该员工工资发放时间'' AFTER ledger_entry_id',
  'SELECT 1')
  FROM information_schema.columns
  WHERE table_schema=@db_name AND table_name='payroll_items' AND column_name='paid_at');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE payroll_items ADD KEY idx_payroll_items_ledger (ledger_entry_id)',
  'SELECT 1')
  FROM information_schema.statistics
  WHERE table_schema=@db_name AND table_name='payroll_items' AND index_name='idx_payroll_items_ledger');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 执行完成后再重启服务；不要对已经发放的工资重新生成或修改。
