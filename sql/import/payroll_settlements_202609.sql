-- 可选：先执行 employee_import_202609.sql，再在后台“生成 2026-09 工资”后执行本文件。
-- 两名离职员工已在线下结算：固定实发并标记为已发放，不创建重复台账支出。
START TRANSACTION;
UPDATE payroll_items i
JOIN payroll_batches b ON b.id=i.payroll_batch_id
JOIN employees e ON e.id=i.employee_id
SET i.manual_net_salary=1236.00,i.net_salary=1236.00,i.status='paid',i.paid_at=COALESCE(i.paid_at,NOW()),i.remark='线下已发放1236元（固定实发）'
WHERE b.payroll_month='2026-09-01' AND e.employee_no='IMPORT-202609-021';
UPDATE payroll_items i
JOIN payroll_batches b ON b.id=i.payroll_batch_id
JOIN employees e ON e.id=i.employee_id
SET i.manual_net_salary=748.00,i.net_salary=748.00,i.status='paid',i.paid_at=COALESCE(i.paid_at,NOW()),i.remark='线下已发放748元（固定实发）'
WHERE b.payroll_month='2026-09-01' AND e.employee_no='IMPORT-202609-022';
UPDATE payroll_batches b
SET gross_amount=(SELECT COALESCE(SUM(i.base_salary+i.bonus_amount),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id),
    deduction_amount=(SELECT COALESCE(SUM(i.attendance_deduction+i.other_deduction+i.social_security+i.tax_amount),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id),
    net_amount=(SELECT COALESCE(SUM(i.net_salary),0) FROM payroll_items i WHERE i.payroll_batch_id=b.id)
WHERE b.payroll_month='2026-09-01' AND b.status<>'paid';
COMMIT;
