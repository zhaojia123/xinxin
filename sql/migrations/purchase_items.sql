-- 已有项目执行一次即可；如果从头执行 sql/schema.sql，则不需要再次执行本文件。
CREATE TABLE IF NOT EXISTS purchase_items (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '采购明细ID',
    purchase_date DATE NOT NULL COMMENT '采购日期',
    supplier_name VARCHAR(64) NOT NULL COMMENT '供货商名称',
    category VARCHAR(32) NOT NULL COMMENT '采购品类，如蔬菜、调料、肉类',
    product_name VARCHAR(128) NOT NULL COMMENT '菜品或物料名称',
    quantity DECIMAL(12,3) NOT NULL DEFAULT 1 COMMENT '采购数量',
    unit VARCHAR(16) NOT NULL COMMENT '采购单位',
    unit_price DECIMAL(12,2) NOT NULL DEFAULT 0 COMMENT '单价（元）',
    total_price DECIMAL(14,2) NOT NULL DEFAULT 0 COMMENT '本行总价（元）',
    remark VARCHAR(500) NOT NULL DEFAULT '' COMMENT '特殊需求或备注',
    last_change_detail TEXT NULL COMMENT '最近一次编辑的前后值',
    last_changed_at DATETIME NULL COMMENT '最近一次编辑时间',
    operator_id BIGINT UNSIGNED NULL COMMENT '最后操作用户ID',
    active TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1正常，0已删除',
    deleted_at DATETIME NULL COMMENT '删除时间',
    deleted_by BIGINT UNSIGNED NULL COMMENT '执行删除的用户ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_purchase_items_date (purchase_date, active, id),
    KEY idx_purchase_items_category (category, purchase_date, active),
    KEY idx_purchase_items_deleted_by (deleted_by),
    CONSTRAINT chk_purchase_items_quantity CHECK (quantity > 0),
    CONSTRAINT chk_purchase_items_price CHECK (unit_price >= 0 AND total_price >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='每日供货商采购明细';

SET @db_name = DATABASE();
SET @add_last_change_detail = (SELECT IF(COUNT(*)=0, 'ALTER TABLE purchase_items ADD COLUMN last_change_detail TEXT NULL COMMENT ''最近一次编辑的前后值'' AFTER remark', 'SELECT 1') FROM information_schema.columns WHERE table_schema=@db_name AND table_name='purchase_items' AND column_name='last_change_detail');
PREPARE stmt FROM @add_last_change_detail;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
SET @add_last_changed_at = (SELECT IF(COUNT(*)=0, 'ALTER TABLE purchase_items ADD COLUMN last_changed_at DATETIME NULL COMMENT ''最近一次编辑时间'' AFTER last_change_detail', 'SELECT 1') FROM information_schema.columns WHERE table_schema=@db_name AND table_name='purchase_items' AND column_name='last_changed_at');
PREPARE stmt FROM @add_last_changed_at;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
