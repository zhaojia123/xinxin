-- 仅供数据库管理员手工执行；应用当前版本不依赖新增字段即可运行。
--
-- 当前请假/调休已有 attendance_records.record_type、duration_days、salary_effect、source、status，
-- 小程序已有 mini_users 与 mini_user_permissions，均无需再加字段。
-- 本脚本只补一个高频查询索引，降低供货商横向对账保存时定位交叉单元格的耗时。
-- MySQL 5.7 不支持 CREATE INDEX IF NOT EXISTS，因此用 information_schema 保障可重复执行。

SET @schema_name = DATABASE();
SET @index_exists = (
    SELECT COUNT(*)
    FROM information_schema.statistics
    WHERE table_schema = @schema_name
      AND table_name = 'stall_cost_entries'
      AND index_name = 'idx_stall_cost_matrix_lookup'
);
SET @sql = IF(
    @index_exists = 0,
    'ALTER TABLE stall_cost_entries ADD INDEX idx_stall_cost_matrix_lookup (stall_id, cost_month, supplier_name, active, ledger_entry_id)',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 执行后可确认索引已生效：
-- SHOW INDEX FROM stall_cost_entries WHERE Key_name = 'idx_stall_cost_matrix_lookup';
