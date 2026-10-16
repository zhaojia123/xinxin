-- 临时工可按半天固定金额结算；每次工时记录仍统一按小时填写，半天默认4小时。
ALTER TABLE employees MODIFY pay_basis ENUM('monthly','daily','hourly','half_daily') NOT NULL DEFAULT 'monthly' COMMENT '月薪、日薪、时薪或半天工资';
