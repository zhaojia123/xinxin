-- 可选：先执行 employee_import_202609.sql，再在后台“生成 2026-09 工资”后执行本文件。
-- 只覆盖仍为草稿的两条工资明细，不会修改已经确认或发放的工资。
START TRANSACTION;
UPDATE payroll_items i
JOIN payroll_batches b ON b.id=i.payroll_batch_id
JOIN employees e ON e.id=i.employee_id
SET i.net_salary=1236.00,i.remark='Excel导入：已结算1236元'
WHERE b.payroll_month='2026-09-01' AND e.employee_no='IMPORT-202609-021' AND i.status='draft';
UPDATE payroll_items i
JOIN payroll_batches b ON b.id=i.payroll_batch_id
JOIN employees e ON e.id=i.employee_id
SET i.net_salary=748.00,i.remark='Excel导入：已结算748元'
WHERE b.payroll_month='2026-09-01' AND e.employee_no='IMPORT-202609-022' AND i.status='draft';
UPDATE payroll_batches b
SET gross_amount=(SELECT COALESCE(SUM(i.base_salary+i.bonus_amount),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id),
    deduction_amount=(SELECT COALESCE(SUM(i.attendance_deduction+i.other_deduction+i.social_security+i.tax_amount),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id),
    net_amount=(SELECT COALESCE(SUM(i.net_salary),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id)
WHERE b.payroll_month='2026-09-01' AND b.status='draft';
COMMIT;
