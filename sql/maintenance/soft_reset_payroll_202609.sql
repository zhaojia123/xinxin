-- 2026-09 工资与指定异常记录安全清理脚本。
-- 只做假删除：不执行 DELETE，不物理删除工资、台账或请假记录。
-- 执行前请确认数据库连接的是当前项目数据库。

SET @payroll_month = '2026-09-01';
SET @special_employee_name = '邢爱莲';
SET @special_rest_date = '2026-09-12';

-- 先预览将被处理的工资批次、工资台账和特殊休息记录。
SELECT id, payroll_month, status, ledger_entry_id
FROM payroll_batches
WHERE payroll_month = @payroll_month;

SELECT l.id, l.occurred_on, l.amount, l.summary, l.active
FROM ledger_entries l
LEFT JOIN payroll_batches b ON b.ledger_entry_id = l.id
LEFT JOIN payroll_items pi ON pi.ledger_entry_id = l.id
LEFT JOIN payroll_batches ib ON ib.id = pi.payroll_batch_id
WHERE l.active = 1
  AND (b.payroll_month = @payroll_month OR ib.payroll_month = @payroll_month);

SELECT ar.id, e.name, ar.occurred_on, ar.record_type, ar.salary_effect, ar.subsidy_amount, ar.active
FROM attendance_records ar
JOIN employees e ON e.id = ar.employee_id
WHERE e.name = @special_employee_name
  AND ar.occurred_on = @special_rest_date
  AND ar.record_type = '特殊休息'
  AND ar.active = 1;

START TRANSACTION;

-- 工资批次和工资明细保留原记录，标记整个当月批次为作废。
UPDATE payroll_batches
SET status = 'cancelled',
    remark = CONCAT(LEFT(remark, 400), '；管理员作废，准备重新生成'),
    updated_at = NOW()
WHERE payroll_month = @payroll_month
  AND status <> 'cancelled';

-- 工资发放产生的台账只做假删除，普通台账不会被处理。
UPDATE ledger_entries l
LEFT JOIN payroll_batches b ON b.ledger_entry_id = l.id
LEFT JOIN payroll_items pi ON pi.ledger_entry_id = l.id
LEFT JOIN payroll_batches ib ON ib.id = pi.payroll_batch_id
SET l.active = 0,
    l.deleted_at = COALESCE(l.deleted_at, NOW()),
    l.deleted_by = NULL,
    l.updated_at = NOW()
WHERE l.active = 1
  AND (b.payroll_month = @payroll_month OR ib.payroll_month = @payroll_month);

-- 12号特殊休息改为正常公休：仅隐藏该员工对应的异常记录。
UPDATE attendance_records ar
JOIN employees e ON e.id = ar.employee_id
SET ar.active = 0,
    ar.deleted_at = COALESCE(ar.deleted_at, NOW()),
    ar.deleted_by = NULL,
    ar.updated_at = NOW()
WHERE e.name = @special_employee_name
  AND ar.occurred_on = @special_rest_date
  AND ar.record_type = '特殊休息'
  AND ar.active = 1;

COMMIT;

-- 脚本执行后重启服务，再在 2026-09 月度工资页面点击“生成本月工资”。
