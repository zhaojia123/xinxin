-- 2026年9月特殊离职员工结算
-- 可在 Navicat 中直接执行；如果工资明细尚未生成，生成后再执行一次本文件即可。
-- 华丞合：线下已发放1236元；邢爱莲：线下已发放748元。

SET @db_name = DATABASE();
SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE payroll_items ADD COLUMN manual_net_salary DECIMAL(12,2) NULL COMMENT ''人工固定实发金额，优先于自动计算'' AFTER net_salary',
    'SELECT 1')
    FROM information_schema.columns
    WHERE table_schema=@db_name AND table_name='payroll_items' AND column_name='manual_net_salary');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

START TRANSACTION;

UPDATE payroll_items i
JOIN payroll_batches b ON b.id=i.payroll_batch_id
JOIN employees e ON e.id=i.employee_id
SET i.manual_net_salary=1236.00,
    i.net_salary=1236.00,
    i.status='paid',
    i.paid_at=COALESCE(i.paid_at,NOW()),
    i.remark='线下已发放1236元（固定实发）'
WHERE b.payroll_month='2026-09-01'
  AND e.employee_no='IMPORT-202609-021';

UPDATE payroll_items i
JOIN payroll_batches b ON b.id=i.payroll_batch_id
JOIN employees e ON e.id=i.employee_id
SET i.manual_net_salary=748.00,
    i.net_salary=748.00,
    i.status='paid',
    i.paid_at=COALESCE(i.paid_at,NOW()),
    i.remark='线下已发放748元（固定实发）'
WHERE b.payroll_month='2026-09-01'
  AND e.employee_no='IMPORT-202609-022';

UPDATE payroll_batches b
SET gross_amount=(SELECT COALESCE(SUM(i.base_salary+i.bonus_amount),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id),
    deduction_amount=(SELECT COALESCE(SUM(i.attendance_deduction+i.other_deduction+i.social_security+i.tax_amount),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id),
    net_amount=(SELECT COALESCE(SUM(i.net_salary),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id)
WHERE b.payroll_month='2026-09-01'
  AND b.status<>'paid';

COMMIT;
