-- Friends Records 完整表结构（MySQL 8.0+）
-- 本文件不会被程序自动执行，也不会创建数据库。请先自行创建并选择数据库，再执行本文件。
-- 所有金额单位均为元；时长分别使用分钟或天。

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE admin_users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '后台用户ID',
    username VARCHAR(64) NOT NULL COMMENT '登录名',
    password_hash VARCHAR(255) NOT NULL COMMENT 'bcrypt密码摘要',
    display_name VARCHAR(64) NOT NULL COMMENT '显示名称',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_admin_users_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='后台用户';

CREATE TABLE departments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '部门ID',
    department_no VARCHAR(32) NOT NULL COMMENT '部门编号',
    name VARCHAR(64) NOT NULL COMMENT '部门名称',
    parent_id BIGINT UNSIGNED NULL COMMENT '上级部门ID',
    manager_employee_id BIGINT UNSIGNED NULL COMMENT '负责人员工ID',
    sort_order INT NOT NULL DEFAULT 0 COMMENT '排序值',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    remark VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
	active TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1正常，0已删除',
	deleted_at DATETIME NULL COMMENT '删除时间',
	deleted_by BIGINT UNSIGNED NULL COMMENT '执行删除的后台用户ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_departments_no (department_no),
    UNIQUE KEY uk_departments_name (name),
    KEY idx_departments_parent (parent_id),
	KEY idx_departments_active (active),
	KEY idx_departments_deleted_by (deleted_by),
    CONSTRAINT fk_departments_parent FOREIGN KEY (parent_id) REFERENCES departments (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='部门';

CREATE TABLE positions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '岗位ID',
    position_no VARCHAR(32) NOT NULL COMMENT '岗位编号',
    department_id BIGINT UNSIGNED NOT NULL COMMENT '所属部门ID',
    name VARCHAR(64) NOT NULL COMMENT '岗位名称',
    level_name VARCHAR(32) NOT NULL DEFAULT '' COMMENT '职级',
    min_salary DECIMAL(12,2) NULL COMMENT '薪资范围下限',
    max_salary DECIMAL(12,2) NULL COMMENT '薪资范围上限',
    sort_order INT NOT NULL DEFAULT 0 COMMENT '排序值',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    remark VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
	active TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1正常，0已删除',
	deleted_at DATETIME NULL COMMENT '删除时间',
	deleted_by BIGINT UNSIGNED NULL COMMENT '执行删除的后台用户ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_positions_department_name (department_id, name),
    UNIQUE KEY uk_positions_no (position_no),
    KEY idx_positions_department (department_id),
	KEY idx_positions_active (active),
	KEY idx_positions_deleted_by (deleted_by),
    CONSTRAINT fk_positions_department FOREIGN KEY (department_id) REFERENCES departments (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='岗位';

CREATE TABLE employees (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '员工ID',
    employee_no VARCHAR(32) NOT NULL COMMENT '员工编号',
    name VARCHAR(64) NOT NULL COMMENT '姓名',
    gender ENUM('unknown','male','female') NOT NULL DEFAULT 'unknown' COMMENT '性别',
    id_card VARCHAR(32) NOT NULL DEFAULT '' COMMENT '身份证号，建议应用层加密',
    mobile VARCHAR(32) NOT NULL DEFAULT '' COMMENT '手机号',
    department_id BIGINT UNSIGNED NULL COMMENT '当前部门ID',
    position_id BIGINT UNSIGNED NULL COMMENT '当前岗位ID',
    employment_status ENUM('probation','active','left') NOT NULL DEFAULT 'probation' COMMENT '试用期/在职/离职',
    employment_type ENUM('full_time','part_time','intern') NOT NULL DEFAULT 'full_time' COMMENT '正式/兼职/实习',
    joined_on DATE NULL COMMENT '入职日期',
    regularized_on DATE NULL COMMENT '转正日期',
    left_on DATE NULL COMMENT '离职日期',
    current_salary DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT '当前月基本工资',
    education VARCHAR(32) NOT NULL DEFAULT '' COMMENT '学历',
    hometown VARCHAR(128) NOT NULL DEFAULT '' COMMENT '籍贯',
    remark VARCHAR(500) NOT NULL DEFAULT '' COMMENT '备注',
	active TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1正常，0已删除',
	deleted_at DATETIME NULL COMMENT '删除时间',
	deleted_by BIGINT UNSIGNED NULL COMMENT '执行删除的后台用户ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_employees_employee_no (employee_no),
    KEY idx_employees_department (department_id),
    KEY idx_employees_position (position_id),
    KEY idx_employees_status (employment_status),
	KEY idx_employees_active (active),
	KEY idx_employees_deleted_by (deleted_by),
    CONSTRAINT fk_employees_department FOREIGN KEY (department_id) REFERENCES departments (id) ON DELETE SET NULL,
    CONSTRAINT fk_employees_position FOREIGN KEY (position_id) REFERENCES positions (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='员工档案';

ALTER TABLE departments ADD CONSTRAINT fk_departments_manager
    FOREIGN KEY (manager_employee_id) REFERENCES employees (id) ON DELETE SET NULL;

CREATE TABLE employee_health_certificates (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '健康证ID',
    employee_id BIGINT UNSIGNED NOT NULL COMMENT '员工ID',
    storage_key VARCHAR(255) NOT NULL COMMENT '服务器内的相对存储路径',
    file_url VARCHAR(500) NOT NULL COMMENT '页面访问地址',
    original_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '上传时原文件名',
    mime_type VARCHAR(64) NOT NULL COMMENT '图片类型',
    file_size BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '文件字节数',
    issued_on DATE NOT NULL COMMENT '发证日期',
    expires_on DATE NOT NULL COMMENT '到期日期',
    is_current TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否为当前使用的健康证',
    uploaded_by BIGINT UNSIGNED NULL COMMENT '上传管理员ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_health_cert_employee_current (employee_id, is_current, expires_on),
    KEY idx_health_cert_expiry (is_current, expires_on),
    CONSTRAINT fk_health_cert_employee FOREIGN KEY (employee_id) REFERENCES employees (id) ON DELETE CASCADE,
    CONSTRAINT fk_health_cert_uploader FOREIGN KEY (uploaded_by) REFERENCES admin_users (id) ON DELETE SET NULL,
    CONSTRAINT chk_health_cert_dates CHECK (expires_on >= issued_on)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='员工健康证及续证历史';

CREATE TABLE mini_users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '小程序用户ID',
    employee_id BIGINT UNSIGNED NULL COMMENT '关联员工，可为空',
    openid VARCHAR(128) NOT NULL COMMENT '微信OpenID',
    unionid VARCHAR(128) NOT NULL DEFAULT '' COMMENT '微信UnionID',
    display_name VARCHAR(64) NOT NULL DEFAULT '' COMMENT '显示名称',
    enabled TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否启用',
    last_login_at DATETIME NULL COMMENT '最后登录时间',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_mini_users_openid (openid),
    UNIQUE KEY uk_mini_users_employee (employee_id),
    CONSTRAINT fk_mini_users_employee FOREIGN KEY (employee_id) REFERENCES employees (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小程序用户';

CREATE TABLE attendance_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '记录ID',
    employee_id BIGINT UNSIGNED NOT NULL COMMENT '员工ID',
    category ENUM('leave','exception') NOT NULL COMMENT '请假/异常',
    record_type VARCHAR(32) NOT NULL COMMENT '年假、事假、调休、迟到、早退等',
    occurred_on DATE NOT NULL COMMENT '发生日期',
    start_time TIME NULL COMMENT '开始或发生时间',
    end_time TIME NULL COMMENT '结束时间',
    duration_minutes INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '迟到、早退等分钟数',
    duration_days DECIMAL(6,2) NOT NULL DEFAULT 0 COMMENT '请假天数',
    reason VARCHAR(500) NOT NULL DEFAULT '' COMMENT '原因',
    source ENUM('employee','manual','mini_program','import') NOT NULL DEFAULT 'manual' COMMENT '来源',
    status ENUM('pending','approved','rejected','recorded','confirmed','cancelled') NOT NULL DEFAULT 'pending' COMMENT '状态',
    reviewed_by BIGINT UNSIGNED NULL COMMENT '审批人',
    reviewed_at DATETIME NULL COMMENT '审批时间',
    review_remark VARCHAR(255) NOT NULL DEFAULT '' COMMENT '审批备注',
	active TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1正常，0已删除',
	deleted_at DATETIME NULL COMMENT '删除时间',
	deleted_by BIGINT UNSIGNED NULL COMMENT '执行删除的后台用户ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_attendance_employee_date (employee_id, occurred_on),
    KEY idx_attendance_period_type (occurred_on, category, record_type, status),
	KEY idx_attendance_active (active),
	KEY idx_attendance_deleted_by (deleted_by),
    CONSTRAINT fk_attendance_employee FOREIGN KEY (employee_id) REFERENCES employees (id) ON DELETE RESTRICT,
    CONSTRAINT fk_attendance_reviewer FOREIGN KEY (reviewed_by) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='请假与考勤异常';

CREATE TABLE employment_changes (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '异动ID',
    employee_id BIGINT UNSIGNED NOT NULL COMMENT '员工ID',
    change_type ENUM('hire','regularize','transfer','promotion','demotion','leave') NOT NULL COMMENT '异动类型',
    before_department_id BIGINT UNSIGNED NULL,
    after_department_id BIGINT UNSIGNED NULL,
    before_position_id BIGINT UNSIGNED NULL,
    after_position_id BIGINT UNSIGNED NULL,
    before_status ENUM('probation','active','left') NULL,
    after_status ENUM('probation','active','left') NULL,
    effective_on DATE NOT NULL COMMENT '生效日期',
    reason VARCHAR(500) NOT NULL DEFAULT '' COMMENT '原因',
    operator_id BIGINT UNSIGNED NULL COMMENT '操作人',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_employment_changes_employee_date (employee_id, effective_on),
    CONSTRAINT fk_changes_employee FOREIGN KEY (employee_id) REFERENCES employees (id) ON DELETE RESTRICT,
    CONSTRAINT fk_changes_before_department FOREIGN KEY (before_department_id) REFERENCES departments (id) ON DELETE SET NULL,
    CONSTRAINT fk_changes_after_department FOREIGN KEY (after_department_id) REFERENCES departments (id) ON DELETE SET NULL,
    CONSTRAINT fk_changes_before_position FOREIGN KEY (before_position_id) REFERENCES positions (id) ON DELETE SET NULL,
    CONSTRAINT fk_changes_after_position FOREIGN KEY (after_position_id) REFERENCES positions (id) ON DELETE SET NULL,
    CONSTRAINT fk_changes_operator FOREIGN KEY (operator_id) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='人事异动历史';

CREATE TABLE salary_adjustments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '调薪ID',
    employee_id BIGINT UNSIGNED NOT NULL COMMENT '员工ID',
    employment_change_id BIGINT UNSIGNED NULL COMMENT '关联晋升或调岗记录',
    before_salary DECIMAL(12,2) NOT NULL,
    after_salary DECIMAL(12,2) NOT NULL,
    effective_on DATE NOT NULL COMMENT '生效日期',
    reason VARCHAR(500) NOT NULL DEFAULT '',
    operator_id BIGINT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_salary_adjustments_employee_date (employee_id, effective_on),
    CONSTRAINT fk_salary_adjustments_employee FOREIGN KEY (employee_id) REFERENCES employees (id) ON DELETE RESTRICT,
    CONSTRAINT fk_salary_adjustments_change FOREIGN KEY (employment_change_id) REFERENCES employment_changes (id) ON DELETE SET NULL,
    CONSTRAINT fk_salary_adjustments_operator FOREIGN KEY (operator_id) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='调薪历史';

CREATE TABLE payroll_batches (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '工资批次ID',
    payroll_month DATE NOT NULL COMMENT '工资月份，固定存当月1日',
    status ENUM('draft','confirmed','paid','cancelled') NOT NULL DEFAULT 'draft' COMMENT '草稿/已确认/已发放/已取消',
    employee_count INT UNSIGNED NOT NULL DEFAULT 0,
    gross_amount DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '应发合计',
    deduction_amount DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '扣款税费合计',
    net_amount DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '实发合计',
    confirmed_by BIGINT UNSIGNED NULL,
    confirmed_at DATETIME NULL,
    paid_at DATETIME NULL,
    ledger_entry_id BIGINT UNSIGNED NULL COMMENT '发放后关联的台账支出',
    remark VARCHAR(500) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_payroll_batches_month (payroll_month),
    UNIQUE KEY uk_payroll_batches_ledger (ledger_entry_id),
    CONSTRAINT fk_payroll_batches_confirmer FOREIGN KEY (confirmed_by) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='月度工资批次';

CREATE TABLE payroll_items (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '工资条ID',
    payroll_batch_id BIGINT UNSIGNED NOT NULL,
    employee_id BIGINT UNSIGNED NOT NULL,
    base_salary DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT '基本工资快照',
    bonus_amount DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT '奖金补贴',
    attendance_deduction DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT '请假考勤扣款',
    other_deduction DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT '其他扣款',
    social_security DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT '社保',
    tax_amount DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT '个税',
    net_salary DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT '实发工资',
    status ENUM('draft','confirmed','paid') NOT NULL DEFAULT 'draft',
    remark VARCHAR(500) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_payroll_items_batch_employee (payroll_batch_id, employee_id),
    KEY idx_payroll_items_employee (employee_id),
    CONSTRAINT fk_payroll_items_batch FOREIGN KEY (payroll_batch_id) REFERENCES payroll_batches (id) ON DELETE CASCADE,
    CONSTRAINT fk_payroll_items_employee FOREIGN KEY (employee_id) REFERENCES employees (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='员工月度工资条';

CREATE TABLE payroll_adjustments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    payroll_item_id BIGINT UNSIGNED NOT NULL COMMENT '工资条ID',
    attendance_record_id BIGINT UNSIGNED NULL COMMENT '关联请假或异常记录',
    adjustment_type ENUM('bonus','deduction') NOT NULL,
    amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_payroll_adjustment_attendance (payroll_item_id, attendance_record_id),
    CONSTRAINT fk_payroll_adjustments_item FOREIGN KEY (payroll_item_id) REFERENCES payroll_items (id) ON DELETE CASCADE,
    CONSTRAINT fk_payroll_adjustments_attendance FOREIGN KEY (attendance_record_id) REFERENCES attendance_records (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工资奖金扣款明细';

CREATE TABLE ledger_accounts (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '资金账户ID',
    account_no VARCHAR(32) NOT NULL,
    name VARCHAR(64) NOT NULL COMMENT '基本账户、微信账户、现金账户等',
    account_type ENUM('bank','wechat','alipay','cash','other') NOT NULL DEFAULT 'bank',
    opening_balance DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '启用时余额',
    opened_on DATE NULL,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    remark VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_ledger_accounts_no (account_no),
    UNIQUE KEY uk_ledger_accounts_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='台账资金账户';

CREATE TABLE ledger_categories (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    direction ENUM('income','expense') NOT NULL COMMENT '收入/支出',
    name VARCHAR(64) NOT NULL COMMENT '服务收入、工资薪酬、房租等',
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    sort_order INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_ledger_categories_direction_name (direction, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='台账收支分类';

CREATE TABLE ledger_entries (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '流水ID',
    occurred_on DATE NOT NULL COMMENT '发生日期',
    account_id BIGINT UNSIGNED NOT NULL COMMENT '账户ID',
    category_id BIGINT UNSIGNED NULL COMMENT '收支分类ID',
    department_id BIGINT UNSIGNED NULL COMMENT '归属部门ID',
    direction ENUM('income','expense') NOT NULL COMMENT '流入/流出',
    amount DECIMAL(14,2) NOT NULL COMMENT '金额，必须为正数',
    summary VARCHAR(255) NOT NULL COMMENT '摘要',
    counterparty VARCHAR(128) NOT NULL DEFAULT '' COMMENT '对方单位名称',
    voucher_no VARCHAR(64) NOT NULL DEFAULT '' COMMENT '凭证号',
    operator_id BIGINT UNSIGNED NULL,
    remark VARCHAR(500) NOT NULL DEFAULT '',
	active TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1正常，0已删除',
	deleted_at DATETIME NULL COMMENT '删除时间',
	deleted_by BIGINT UNSIGNED NULL COMMENT '执行删除的后台用户ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_ledger_entries_date (occurred_on, id),
    KEY idx_ledger_entries_account_date (account_id, occurred_on, id),
    KEY idx_ledger_entries_department (department_id),
    KEY idx_ledger_entries_category (category_id),
	KEY idx_ledger_entries_active (active),
	KEY idx_ledger_entries_deleted_by (deleted_by),
    CONSTRAINT fk_ledger_entries_account FOREIGN KEY (account_id) REFERENCES ledger_accounts (id) ON DELETE RESTRICT,
    CONSTRAINT fk_ledger_entries_category FOREIGN KEY (category_id) REFERENCES ledger_categories (id) ON DELETE SET NULL,
    CONSTRAINT fk_ledger_entries_department FOREIGN KEY (department_id) REFERENCES departments (id) ON DELETE SET NULL,
    CONSTRAINT fk_ledger_entries_operator FOREIGN KEY (operator_id) REFERENCES admin_users (id) ON DELETE SET NULL,
    CONSTRAINT chk_ledger_entries_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='出纳台账流水';

ALTER TABLE payroll_batches ADD CONSTRAINT fk_payroll_batches_ledger
    FOREIGN KEY (ledger_entry_id) REFERENCES ledger_entries (id) ON DELETE SET NULL;

CREATE TABLE audit_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    admin_user_id BIGINT UNSIGNED NULL,
    module VARCHAR(64) NOT NULL COMMENT '模块',
    action VARCHAR(64) NOT NULL COMMENT '动作',
    target_type VARCHAR(64) NOT NULL DEFAULT '' COMMENT '对象类型',
    target_id BIGINT UNSIGNED NULL COMMENT '对象ID',
    target_text VARCHAR(255) NOT NULL DEFAULT '' COMMENT '对象说明',
    ip_address VARCHAR(64) NOT NULL DEFAULT '',
    result ENUM('success','failure') NOT NULL DEFAULT 'success',
    detail TEXT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_audit_logs_created (created_at),
    KEY idx_audit_logs_user (admin_user_id),
    CONSTRAINT fk_audit_logs_admin FOREIGN KEY (admin_user_id) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='操作日志';

SET FOREIGN_KEY_CHECKS = 1;

-- 首次使用提示：
-- 1. 本文件只建表、不插入演示数据，因此页面和查询接口初始会返回空列表或金额0。
-- 2. 后台登录至少需要一条 admin_users 数据，password_hash 必须使用 bcrypt 生成。
-- 3. 先维护部门、岗位和台账账户，再录入员工及业务数据。
