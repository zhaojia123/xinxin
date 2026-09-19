-- 2026-09-18：工资手动实发 + 付款账户类型/备注。
-- 可在当前数据库直接执行；重复执行也不会重复添加字段。
SET @db_name = DATABASE();

SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE payroll_items ADD COLUMN manual_net_salary DECIMAL(12,2) NULL COMMENT ''手动实发工资；为空时按算法计算'' AFTER net_salary',
  'SELECT 1')
  FROM information_schema.columns
  WHERE table_schema=@db_name AND table_name='payroll_items' AND column_name='manual_net_salary');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE ledger_accounts ADD COLUMN remark VARCHAR(255) NOT NULL DEFAULT '''' COMMENT ''账户备注，如QQ号、微信号或银行卡后四位'' AFTER enabled',
  'SELECT 1')
  FROM information_schema.columns
  WHERE table_schema=@db_name AND table_name='ledger_accounts' AND column_name='remark');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE employees ADD COLUMN monthly_rest_days TINYINT UNSIGNED NULL COMMENT ''每月公休天数；为空时跟随系统配置'' AFTER pay_basis',
  'SELECT 1')
  FROM information_schema.columns
  WHERE table_schema=@db_name AND table_name='employees' AND column_name='monthly_rest_days');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

ALTER TABLE ledger_accounts
  MODIFY COLUMN account_type ENUM('bank','wechat','alipay','qq','cash','other') NOT NULL DEFAULT 'bank' COMMENT '银行卡、微信、支付宝、QQ、现金或其他';
