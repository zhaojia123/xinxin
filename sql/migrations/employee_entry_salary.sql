-- 已有项目执行一次即可；从头执行 sql/schema.sql 则不需要再次执行。
-- 作用：保存员工入职时的工资快照，与当前工资区分。
SET @db_name = DATABASE();
SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE employees ADD COLUMN entry_salary DECIMAL(12,2) NULL COMMENT ''入职时工资或时薪，作为历史快照'' AFTER left_on',
  'SELECT 1')
  FROM information_schema.columns
  WHERE table_schema=@db_name AND table_name='employees' AND column_name='entry_salary');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 老数据没有历史工资时，用当前工资初始化；后续调薪不会覆盖该字段。
UPDATE employees
SET entry_salary=current_salary
WHERE entry_salary IS NULL AND current_salary IS NOT NULL;
