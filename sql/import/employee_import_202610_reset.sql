-- 员工基础资料重置导入：来源为“员工入职时间统计(1).xlsx”的“10月后厨人员工资表”。
--
-- 重要说明：
-- 1. 本脚本会清空业务数据（员工、组织、考勤、调薪、工资、台账、采购、小程序用户等）。
-- 2. 为保留后台登录账号，admin_users 表不会清空。
-- 3. 基础资料来自 10 月工资表；身份证、电话、入职时间和健康证到期日来自“厦门理工学院食堂员工和供应商信息表（9月份）”。
-- 4. 原表中的“白欣欣”出现两行（2026-08-23、2026-09-01），按同一员工处理并保留较早的入职时间 2026-08-23。
-- 5. 陈美花原表月薪为空，按用户确认先录入 4000 元。
-- 6. 公休设置：白欣欣、蒋宇凰、柯梦娜为 6 天；益禾堂岗位为 4 天；其他员工为 3 天。
-- 7. 9 月信息表只匹配到 16 名员工；吴升频、陈美花、申自秀、宋江同、向云为 10 月新增，资料暂留空。
-- 8. 为保存表格中的年龄，本脚本会给员工表补充可空 age 字段；10 月新增人员没有对应年龄资料时保持 NULL。
-- 9. 健康证来源只提供到期日，发证日期留空，证件图片不伪造，统一标记为“表格导入：未上传证件图片”。
-- 10. 请先确认数据库连接和当前库，再由管理员手动执行本文件；本文件不会被程序自动执行。

SET NAMES utf8mb4;

-- 执行前预览：确认当前业务数据规模。admin_users 仅用于登录，不在清理范围内。
SELECT
    (SELECT COUNT(*) FROM employees) AS old_employees,
    (SELECT COUNT(*) FROM payroll_batches) AS old_payroll_batches,
    (SELECT COUNT(*) FROM ledger_entries) AS old_ledger_entries,
    (SELECT COUNT(*) FROM attendance_records) AS old_attendance_records,
    (SELECT COUNT(*) FROM admin_users) AS preserved_admin_users;

SET FOREIGN_KEY_CHECKS = 0;

TRUNCATE TABLE audit_logs;
TRUNCATE TABLE mini_user_permissions;
TRUNCATE TABLE mini_users;
TRUNCATE TABLE ledger_attachments;
TRUNCATE TABLE payroll_adjustments;
TRUNCATE TABLE payroll_items;
TRUNCATE TABLE payroll_batches;
TRUNCATE TABLE ledger_entries;
TRUNCATE TABLE purchase_items;
TRUNCATE TABLE attendance_records;
TRUNCATE TABLE salary_adjustments;
TRUNCATE TABLE employment_changes;
TRUNCATE TABLE employee_attachments;
TRUNCATE TABLE employee_health_certificates;
TRUNCATE TABLE employees;
TRUNCATE TABLE positions;
TRUNCATE TABLE departments;
TRUNCATE TABLE ledger_accounts;
TRUNCATE TABLE ledger_categories;

SET FOREIGN_KEY_CHECKS = 1;

-- 年龄字段只在不存在时新增，重复执行时不会重复添加。
SET @age_column_exists = (
    SELECT COUNT(*)
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'employees'
      AND column_name = 'age'
);
SET @add_age_column_sql = IF(
    @age_column_exists = 0,
    'ALTER TABLE employees ADD COLUMN age TINYINT UNSIGNED NULL COMMENT ''员工年龄，来源表格'' AFTER name',
    'SELECT 1'
);
PREPARE add_age_column_statement FROM @add_age_column_sql;
EXECUTE add_age_column_statement;
DEALLOCATE PREPARE add_age_column_statement;

-- 允许仅有到期日的表格资料入库；后续上传完整健康证时仍可补充发证日期和图片。
ALTER TABLE employee_health_certificates
    MODIFY COLUMN issued_on DATE NULL COMMENT '发证日期，来源未提供时为空';

START TRANSACTION;

INSERT INTO departments (department_no, name, enabled, active, remark)
VALUES ('ZXX-001', '知行二餐厅', 1, 1, 'Excel导入：2026年10月员工基础资料');

INSERT INTO positions (position_no, department_id, name, enabled, active, remark)
SELECT src.position_no, d.id, src.name, 1, 1, 'Excel导入：2026年10月员工基础资料'
FROM (
    SELECT 'ZXX-P01' AS position_no, '经理' AS name
    UNION ALL SELECT 'ZXX-P02', '文员'
    UNION ALL SELECT 'ZXX-P03', '仓管'
    UNION ALL SELECT 'ZXX-P04', '厨师'
    UNION ALL SELECT 'ZXX-P05', '帮厨'
    UNION ALL SELECT 'ZXX-P06', '切配'
    UNION ALL SELECT 'ZXX-P07', '洗消'
    UNION ALL SELECT 'ZXX-P08', '益禾堂'
) AS src
JOIN departments AS d ON d.department_no = 'ZXX-001';

INSERT INTO employees
    (employee_no, name, gender, id_card, mobile, department_id, position_id,
     employment_status, employment_type, joined_on, regularized_on, left_on,
     entry_salary, current_salary, pay_basis, monthly_rest_days,
     education, hometown, remark, active)
SELECT
    src.employee_no,
    src.name,
    'unknown',
    '',
    '',
    d.id,
    p.id,
    'active',
    'full_time',
    src.joined_on,
    NULL,
    NULL,
    src.monthly_salary,
    src.monthly_salary,
    'monthly',
    CASE
        WHEN src.name IN ('白欣欣', '蒋宇凰', '柯梦娜') THEN 6
        WHEN src.position_name = '益禾堂' THEN 4
        ELSE 3
    END,
    '',
    '',
    'Excel导入：2026年10月员工基础资料',
    1
FROM (
    SELECT 'IMPORT-202610-001' AS employee_no, '白欣欣' AS name, '经理' AS position_name, '2026-08-23' AS joined_on, 9000.00 AS monthly_salary
    UNION ALL SELECT 'IMPORT-202610-002', '蒋宇凰', '文员', '2026-09-02', 5500.00
    UNION ALL SELECT 'IMPORT-202610-003', '柯梦娜', '仓管', '2026-09-11', 5500.00
    UNION ALL SELECT 'IMPORT-202610-004', '林剑锋', '厨师', '2026-08-29', 10000.00
    UNION ALL SELECT 'IMPORT-202610-005', '李龙海', '厨师', '2026-09-01', 7500.00
    UNION ALL SELECT 'IMPORT-202610-006', '张良有', '帮厨', '2026-09-01', 4000.00
    UNION ALL SELECT 'IMPORT-202610-007', '周茂盈', '帮厨', '2026-09-01', 5000.00
    UNION ALL SELECT 'IMPORT-202610-008', '甘美兰', '切配', '2026-09-01', 5000.00
    UNION ALL SELECT 'IMPORT-202610-009', '柏爱珍', '切配', '2026-09-01', 5000.00
    UNION ALL SELECT 'IMPORT-202610-010', '徐和妹', '切配', '2026-09-01', 5000.00
    UNION ALL SELECT 'IMPORT-202610-011', '甘知香', '切配', '2026-09-01', 5000.00
    UNION ALL SELECT 'IMPORT-202610-012', '林碧玉', '切配', '2026-09-01', 5000.00
    UNION ALL SELECT 'IMPORT-202610-013', '张雅玲', '切配', '2026-09-01', 4000.00
    UNION ALL SELECT 'IMPORT-202610-014', '张中荣', '洗消', '2026-09-08', 4000.00
    UNION ALL SELECT 'IMPORT-202610-015', '王健', '益禾堂', '2026-09-16', 4000.00
    UNION ALL SELECT 'IMPORT-202610-016', '肖诗琪', '益禾堂', '2026-09-01', 3500.00
    UNION ALL SELECT 'IMPORT-202610-017', '吴升频', '厨师', '2026-10-08', 8000.00
    UNION ALL SELECT 'IMPORT-202610-018', '陈美花', '切配', '2026-10-08', 4000.00
    UNION ALL SELECT 'IMPORT-202610-019', '申自秀', '洗消', '2026-10-08', 4000.00
    UNION ALL SELECT 'IMPORT-202610-020', '宋江同', '洗消', '2026-10-08', 4000.00
    UNION ALL SELECT 'IMPORT-202610-021', '向云', '洗消', '2026-10-08', 4000.00
) AS src
JOIN departments AS d ON d.department_no = 'ZXX-001'
JOIN positions AS p ON p.department_id = d.id AND p.name = src.position_name;

-- 将 9 月信息表中能匹配到的 16 名员工补充身份证、电话和入职时间。
UPDATE employees AS e
JOIN (
    SELECT '白欣欣' AS name, 30 AS age, '622826199607201964' AS id_card, '15193186221' AS mobile, '2026-08-24' AS joined_on
    UNION ALL SELECT '蒋宇凰', 22, '350583200408193745', '15587116534', '2026-09-02'
    UNION ALL SELECT '柯梦娜', 35, '350212199107093524', '15980971110', '2026-09-11'
    UNION ALL SELECT '林剑锋', 35, '350425199106171618', '18659229261', '2026-09-01'
    UNION ALL SELECT '李龙海', 28, '522728199804049811', '13058828444', '2026-09-01'
    UNION ALL SELECT '张良有', 62, '35042619641018001X', '18159647978', '2026-09-01'
    UNION ALL SELECT '周茂盈', 55, '350426197107251019', '17750670513', '2026-09-01'
    UNION ALL SELECT '甘美兰', 57, '362202196908057029', '17605085897', '2026-09-01'
    UNION ALL SELECT '柏爱珍', 59, '36220219670902702X', '18659272231', '2026-09-01'
    UNION ALL SELECT '徐和妹', 60, '362202196604177021', '15080349785', '2026-09-01'
    UNION ALL SELECT '甘知香', 62, '362202196412177027', '15159242275', '2026-09-01'
    UNION ALL SELECT '林碧玉', 54, '350426197208251026', '15159174896', '2026-09-01'
    UNION ALL SELECT '张雅玲', 49, '330719197712201123', '18059850266', '2026-09-01'
    UNION ALL SELECT '张中荣', 63, '41292819630525162X', '15660028872', '2026-09-01'
    UNION ALL SELECT '王健', 22, '36062220031102005X', '15080302318', '2026-09-01'
    UNION ALL SELECT '肖诗琪', 19, '350426200711241022', '17350588619', '2026-09-01'
) AS src ON src.name = e.name AND e.active = 1
SET e.age = src.age,
    e.id_card = src.id_card,
    e.mobile = src.mobile,
    e.joined_on = src.joined_on;

-- 导入 16 名匹配员工的健康证到期日；发证日期和图片来源未提供，保持为空。
INSERT INTO employee_health_certificates
    (employee_id, storage_key, file_url, original_name, mime_type, file_size,
     issued_on, expires_on, is_current, uploaded_by)
SELECT e.id,
       '',
       '',
       '表格导入：未上传证件图片',
       'application/octet-stream',
       0,
       NULL,
       src.expires_on,
       1,
       NULL
FROM (
    SELECT '白欣欣' AS name, '2027-08-30' AS expires_on
    UNION ALL SELECT '蒋宇凰', '2027-09-07'
    UNION ALL SELECT '柯梦娜', '2027-09-14'
    UNION ALL SELECT '林剑锋', '2027-08-30'
    UNION ALL SELECT '李龙海', '2026-10-31'
    UNION ALL SELECT '张良有', '2027-08-28'
    UNION ALL SELECT '周茂盈', '2027-08-30'
    UNION ALL SELECT '甘美兰', '2027-08-30'
    UNION ALL SELECT '柏爱珍', '2027-09-12'
    UNION ALL SELECT '徐和妹', '2027-03-01'
    UNION ALL SELECT '甘知香', '2027-08-21'
    UNION ALL SELECT '林碧玉', '2027-06-24'
    UNION ALL SELECT '张雅玲', '2027-08-28'
    UNION ALL SELECT '张中荣', '2027-09-14'
    UNION ALL SELECT '王健', '2027-08-23'
    UNION ALL SELECT '肖诗琪', '2027-05-21'
) AS src
JOIN employees AS e ON e.name = src.name AND e.active = 1;

-- 重建基础台账分类；资金账户不预置，后续按实际账户和备注新增。
INSERT INTO ledger_categories (direction, name, enabled, sort_order) VALUES
    ('income', '服务收入', 1, 10),
    ('expense', '工资薪酬', 1, 10),
    ('expense', '房租', 1, 20),
    ('expense', '采购支出', 1, 30);

COMMIT;

-- 执行后核对：应导入 21 名员工，陈美花的月薪应为 4000.00。
SELECT
    e.employee_no,
    e.name,
    p.name AS position_name,
    e.joined_on,
    e.current_salary,
    e.age,
    e.id_card,
    e.mobile,
    DATE_FORMAT(hc.expires_on, '%Y-%m-%d') AS health_certificate_expires_on,
    e.age AS age_from_source
FROM employees AS e
LEFT JOIN positions AS p ON p.id = e.position_id
LEFT JOIN employee_health_certificates AS hc ON hc.employee_id = e.id AND hc.is_current = 1
ORDER BY e.employee_no;

SELECT COUNT(*) AS imported_employee_count
FROM employees
WHERE employee_no LIKE 'IMPORT-202610-%';
