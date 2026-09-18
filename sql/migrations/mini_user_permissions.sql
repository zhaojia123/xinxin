-- 已有项目执行一次即可；从头执行 sql/schema.sql 则不需要再次执行。
-- 三个字段均为 1 表示允许，0 表示禁止。执行后可按用户ID分配模块权限。
-- MySQL 5.7 不支持 ADD COLUMN IF NOT EXISTS，下面用信息_schema判断后再执行，重复运行也安全。
SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE mini_users ADD COLUMN can_ledger TINYINT(1) NOT NULL DEFAULT 1 AFTER enabled',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'mini_users' AND column_name = 'can_ledger');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE mini_users ADD COLUMN can_purchases TINYINT(1) NOT NULL DEFAULT 1 AFTER can_ledger',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'mini_users' AND column_name = 'can_purchases');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE mini_users ADD COLUMN can_employees TINYINT(1) NOT NULL DEFAULT 1 AFTER can_purchases',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'mini_users' AND column_name = 'can_employees');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS mini_user_permissions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '小程序权限ID',
    mini_user_id BIGINT UNSIGNED NOT NULL COMMENT '小程序用户ID',
    module_key VARCHAR(64) NOT NULL COMMENT '模块标识，如ledger、purchases、employees',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '权限记录是否启用',
    can_view TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否允许查看',
    can_create TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否允许新增',
    can_edit TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否允许编辑',
    can_delete TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否允许删除',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_mini_user_permission (mini_user_id, module_key),
    KEY idx_mini_permission_module (module_key, enabled),
    CONSTRAINT fk_mini_permission_user FOREIGN KEY (mini_user_id) REFERENCES mini_users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小程序用户模块权限';

-- 兼容已经创建过旧版权限表的数据库。
SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE mini_user_permissions ADD COLUMN can_view TINYINT(1) NOT NULL DEFAULT 1 AFTER enabled',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'mini_user_permissions' AND column_name = 'can_view');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE mini_user_permissions ADD COLUMN can_create TINYINT(1) NOT NULL DEFAULT 1 AFTER can_view',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'mini_user_permissions' AND column_name = 'can_create');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE mini_user_permissions ADD COLUMN can_edit TINYINT(1) NOT NULL DEFAULT 1 AFTER can_create',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'mini_user_permissions' AND column_name = 'can_edit');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql = (SELECT IF(COUNT(*) = 0,
    'ALTER TABLE mini_user_permissions ADD COLUMN can_delete TINYINT(1) NOT NULL DEFAULT 1 AFTER can_edit',
    'SELECT 1') FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'mini_user_permissions' AND column_name = 'can_delete');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 将现有三个固定字段迁移为可扩展的权限记录；重复执行不会覆盖已有权限。
INSERT IGNORE INTO mini_user_permissions (mini_user_id, module_key, enabled, can_view, can_create, can_edit, can_delete)
SELECT id, 'ledger', can_ledger, can_ledger, can_ledger, can_ledger, can_ledger FROM mini_users;
INSERT IGNORE INTO mini_user_permissions (mini_user_id, module_key, enabled, can_view, can_create, can_edit, can_delete)
SELECT id, 'purchases', can_purchases, can_purchases, can_purchases, can_purchases, can_purchases FROM mini_users;
INSERT IGNORE INTO mini_user_permissions (mini_user_id, module_key, enabled, can_view, can_create, can_edit, can_delete)
SELECT id, 'employees', can_employees, can_employees, can_employees, can_employees, can_employees FROM mini_users;

-- 例如：仅允许用户ID为 2 的用户访问采购模块
-- UPDATE mini_users SET can_ledger=0, can_purchases=1, can_employees=0 WHERE id=2;
-- 推荐改用可扩展权限表：
-- INSERT INTO mini_user_permissions (mini_user_id,module_key,enabled) VALUES (2,'attendance',1)
-- ON DUPLICATE KEY UPDATE enabled=VALUES(enabled);
