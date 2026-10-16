-- 确保台账筛选和工资发放使用的内置收支分类存在。
-- 可重复执行，不会覆盖已有同名分类的其他字段。
INSERT INTO ledger_categories (direction, name, enabled, sort_order)
VALUES
    ('income', '服务收入', 1, 10),
    ('expense', '工资薪酬', 1, 10),
    ('expense', '房租', 1, 20),
    ('expense', '采购支出', 1, 30)
ON DUPLICATE KEY UPDATE enabled = 1;
