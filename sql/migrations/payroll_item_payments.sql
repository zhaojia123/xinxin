-- 已有项目执行一次即可；从头执行 sql/schema.sql 则不需要再次执行。
-- 作用：记录每条工资明细的发放流水，支持单独发放员工工资并避免重复入账。
SET @db_name = DATABASE();

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

-- 线上旧库先保证字段和索引存在；是否补充外键不影响单独发放功能。
