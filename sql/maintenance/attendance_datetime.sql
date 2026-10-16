-- 请假记录支持跨日期、半天及秒级时间。已有数据库执行一次即可。
ALTER TABLE attendance_records
    ADD COLUMN start_at DATETIME NULL COMMENT '开始或发生日期时间' AFTER occurred_on,
    ADD COLUMN end_at DATETIME NULL COMMENT '结束日期时间' AFTER start_at;

UPDATE attendance_records
SET start_at = CASE WHEN start_time IS NULL THEN NULL ELSE TIMESTAMP(occurred_on, start_time) END,
    end_at = CASE WHEN end_time IS NULL THEN NULL ELSE TIMESTAMP(occurred_on, end_time) END
WHERE start_at IS NULL AND end_at IS NULL;
