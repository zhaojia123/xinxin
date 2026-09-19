-- 员工基础资料导入：来自“副本入职时间统计(1).xlsx”的主表。
-- 当前没有小时工，所以只导入主表 24 人；“钟点工”工作表中的人员不导入。
-- 未填写的工资、入职日期、离职日期均保留为空，统一按月薪处理。
-- 执行前请先执行 sql/migrations/employee_payroll_rules.sql。
SET NAMES utf8mb4;
START TRANSACTION;

INSERT INTO departments (department_no,name,enabled,active,remark)
VALUES ('ZXX-001','知行二餐厅',1,1,'Excel员工资料导入')
ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id),enabled=1,active=1;

INSERT IGNORE INTO positions (position_no,name,department_id,enabled,active,remark)
SELECT src.position_no,src.name,d.id,1,1,'Excel员工资料导入'
FROM (
    SELECT 'ZXX-P01' position_no,'经理' name UNION ALL
    SELECT 'ZXX-P02','文员' UNION ALL
    SELECT 'ZXX-P03','仓管' UNION ALL
    SELECT 'ZXX-P04','厨师' UNION ALL
    SELECT 'ZXX-P05','帮厨' UNION ALL
    SELECT 'ZXX-P06','切配' UNION ALL
    SELECT 'ZXX-P07','洗消' UNION ALL
    SELECT 'ZXX-P08','益禾堂'
) src JOIN departments d ON d.name='知行二餐厅';

INSERT IGNORE INTO employees
    (employee_no,name,gender,id_card,mobile,department_id,position_id,employment_status,employment_type,
     joined_on,regularized_on,left_on,entry_salary,current_salary,pay_basis,education,hometown,remark,active)
SELECT src.employee_no,src.name,'unknown','', '',d.id,p.id,src.employment_status,'full_time',
       src.joined_on,NULL,src.left_on,src.current_salary,src.current_salary,'monthly','','','Excel导入：部门为知行二餐厅',1
FROM (
    SELECT 'IMPORT-202609-001' employee_no,'白欣欣' name,'经理' position_name,'active' employment_status,'2026-08-24' joined_on,NULL left_on,NULL current_salary UNION ALL
    SELECT 'IMPORT-202609-002','蒋宇凰','文员','active','2026-09-02',NULL,5500 UNION ALL
    SELECT 'IMPORT-202609-003','柯梦娜','仓管','active','2026-09-11',NULL,5500 UNION ALL
    SELECT 'IMPORT-202609-004','林剑锋','厨师','active','2026-09-01',NULL,10000 UNION ALL
    SELECT 'IMPORT-202609-005','李龙海','厨师','active','2026-09-01',NULL,7500 UNION ALL
    SELECT 'IMPORT-202609-006','黄铭福','厨师','active',NULL,NULL,NULL UNION ALL
    SELECT 'IMPORT-202609-007','吕振兴','厨师','active','2026-09-14',NULL,7500 UNION ALL
    SELECT 'IMPORT-202609-008','张良有','帮厨','active','2026-09-01',NULL,NULL UNION ALL
    SELECT 'IMPORT-202609-009','周茂盈','帮厨','active','2026-09-01',NULL,5000 UNION ALL
    SELECT 'IMPORT-202609-010','王建华','帮厨','active','2026-09-01',NULL,4000 UNION ALL
    SELECT 'IMPORT-202609-011','甘华珍','切配','active','2026-09-01',NULL,5000 UNION ALL
    SELECT 'IMPORT-202609-012','甘美兰','切配','active','2026-09-01',NULL,5000 UNION ALL
    SELECT 'IMPORT-202609-013','柏爱珍','切配','active','2026-09-01',NULL,5000 UNION ALL
    SELECT 'IMPORT-202609-014','徐和妹','切配','active','2026-09-01',NULL,5000 UNION ALL
    SELECT 'IMPORT-202609-015','甘知香','切配','active','2026-09-01',NULL,5000 UNION ALL
    SELECT 'IMPORT-202609-016','张霞','切配','active','2026-09-01',NULL,4000 UNION ALL
    SELECT 'IMPORT-202609-017','杨海秀','切配','active','2026-09-01',NULL,4000 UNION ALL
    SELECT 'IMPORT-202609-018','林碧玉','切配','active','2026-09-01',NULL,5000 UNION ALL
    SELECT 'IMPORT-202609-019','张雅玲','切配','active','2026-09-01',NULL,4000 UNION ALL
    SELECT 'IMPORT-202609-020','张中荣','洗消','active','2026-09-08',NULL,4000 UNION ALL
    SELECT 'IMPORT-202609-021','华丕合','洗消','left','2026-09-01','2026-09-13',4000 UNION ALL
    SELECT 'IMPORT-202609-022','邢爱莲','洗消','left','2026-09-08','2026-09-13',4000 UNION ALL
    SELECT 'IMPORT-202609-023','王健','益禾堂','active',NULL,NULL,NULL UNION ALL
    SELECT 'IMPORT-202609-024','肖诗琪','益禾堂','active',NULL,NULL,NULL
) src
JOIN departments d ON d.name='知行二餐厅'
JOIN positions p ON p.department_id=d.id AND p.name=src.position_name;

-- 将已有入职/离职日期写入人事异动历史，便于人事异动页面查看。
INSERT INTO employment_changes
    (employee_id,change_type,before_department_id,after_department_id,before_position_id,after_position_id,
     before_status,after_status,effective_on,reason)
SELECT e.id,'hire',NULL,e.department_id,NULL,e.position_id,NULL,'active',e.joined_on,'Excel导入：入职记录'
FROM employees e
WHERE e.employee_no LIKE 'IMPORT-202609-%' AND e.joined_on IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM employment_changes c WHERE c.employee_id=e.id AND c.change_type='hire' AND c.effective_on=e.joined_on);

INSERT INTO employment_changes
    (employee_id,change_type,before_department_id,after_department_id,before_position_id,after_position_id,
     before_status,after_status,effective_on,reason)
SELECT e.id,'leave',e.department_id,e.department_id,e.position_id,e.position_id,'active','left',e.left_on,'Excel导入：离职记录，已结算'
FROM employees e
WHERE e.employee_no LIKE 'IMPORT-202609-%' AND e.left_on IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM employment_changes c WHERE c.employee_id=e.id AND c.change_type='leave' AND c.effective_on=e.left_on);

-- 特殊休息：不计正常工资，同时按每天 50 元补助；重复执行不会重复插入。
DROP TEMPORARY TABLE IF EXISTS _import_special_rest;
CREATE TEMPORARY TABLE _import_special_rest (employee_no VARCHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL, occurred_on DATE NOT NULL,
    PRIMARY KEY (employee_no,occurred_on));
INSERT INTO _import_special_rest VALUES
('IMPORT-202609-010','2026-09-04'),('IMPORT-202609-010','2026-09-05'),('IMPORT-202609-010','2026-09-06'),('IMPORT-202609-010','2026-09-07'),('IMPORT-202609-010','2026-09-12'),
('IMPORT-202609-011','2026-09-04'),('IMPORT-202609-011','2026-09-05'),('IMPORT-202609-011','2026-09-06'),('IMPORT-202609-011','2026-09-07'),('IMPORT-202609-011','2026-09-12'),
('IMPORT-202609-012','2026-09-04'),('IMPORT-202609-012','2026-09-05'),('IMPORT-202609-012','2026-09-06'),('IMPORT-202609-012','2026-09-07'),('IMPORT-202609-012','2026-09-12'),
('IMPORT-202609-013','2026-09-04'),('IMPORT-202609-013','2026-09-05'),('IMPORT-202609-013','2026-09-06'),('IMPORT-202609-013','2026-09-07'),('IMPORT-202609-013','2026-09-12'),
('IMPORT-202609-014','2026-09-04'),('IMPORT-202609-014','2026-09-05'),('IMPORT-202609-014','2026-09-06'),('IMPORT-202609-014','2026-09-07'),('IMPORT-202609-014','2026-09-12'),
('IMPORT-202609-015','2026-09-04'),('IMPORT-202609-015','2026-09-05'),('IMPORT-202609-015','2026-09-06'),('IMPORT-202609-015','2026-09-07'),('IMPORT-202609-015','2026-09-12'),
('IMPORT-202609-016','2026-09-04'),('IMPORT-202609-016','2026-09-05'),('IMPORT-202609-016','2026-09-06'),('IMPORT-202609-016','2026-09-07'),('IMPORT-202609-016','2026-09-12'),
('IMPORT-202609-017','2026-09-04'),('IMPORT-202609-017','2026-09-05'),('IMPORT-202609-017','2026-09-06'),('IMPORT-202609-017','2026-09-07'),('IMPORT-202609-017','2026-09-12'),
('IMPORT-202609-018','2026-09-04'),('IMPORT-202609-018','2026-09-05'),('IMPORT-202609-018','2026-09-06'),('IMPORT-202609-018','2026-09-07'),('IMPORT-202609-018','2026-09-12'),
('IMPORT-202609-019','2026-09-04'),('IMPORT-202609-019','2026-09-05'),('IMPORT-202609-019','2026-09-06'),('IMPORT-202609-019','2026-09-07'),('IMPORT-202609-019','2026-09-12'),
('IMPORT-202609-021','2026-09-04'),('IMPORT-202609-021','2026-09-05'),('IMPORT-202609-021','2026-09-06'),('IMPORT-202609-021','2026-09-07'),('IMPORT-202609-021','2026-09-12'),
('IMPORT-202609-020','2026-09-12'),('IMPORT-202609-022','2026-09-12');
INSERT INTO attendance_records
    (employee_id,category,record_type,occurred_on,duration_minutes,duration_days,salary_effect,subsidy_amount,reason,source,status,active)
SELECT e.id,'exception','特殊休息',r.occurred_on,0,1,'deduct_and_subsidy',50,
       '特殊情况休息：不计正常工资，按天补助50元','import','confirmed',1
FROM _import_special_rest r JOIN employees e ON e.employee_no=r.employee_no
WHERE NOT EXISTS (SELECT 1 FROM attendance_records a WHERE a.employee_id=e.id AND a.occurred_on=r.occurred_on AND a.record_type='特殊休息' AND a.active=1);
DROP TEMPORARY TABLE _import_special_rest;

COMMIT;

-- 华丞合、邢爱莲的实际结算金额（1236、748）请在生成 2026-09 工资后执行 sql/import/payroll_settlements_202609.sql。
-- UPDATE payroll_items i JOIN payroll_batches b ON b.id=i.payroll_batch_id JOIN employees e ON e.id=i.employee_id
-- SET i.manual_net_salary=1236.00,i.net_salary=1236.00,i.status='paid',i.paid_at=COALESCE(i.paid_at,NOW()),i.remark='线下已发放1236元（固定实发）'
-- WHERE b.payroll_month='2026-09-01' AND e.employee_no='IMPORT-202609-021';
-- UPDATE payroll_items i JOIN payroll_batches b ON b.id=i.payroll_batch_id JOIN employees e ON e.id=i.employee_id
-- SET i.manual_net_salary=748.00,i.net_salary=748.00,i.status='paid',i.paid_at=COALESCE(i.paid_at,NOW()),i.remark='线下已发放748元（固定实发）'
-- WHERE b.payroll_month='2026-09-01' AND e.employee_no='IMPORT-202609-022';
