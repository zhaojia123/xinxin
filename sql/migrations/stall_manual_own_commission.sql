-- 档口当月手动抽成：留空表示沿用档口规则，填写后仅覆盖当前档口当前月份的我方抽成。
SET @schema_name = DATABASE();

SELECT IF(COUNT(*)=0,'ALTER TABLE stall_turnovers ADD COLUMN manual_own_commission DECIMAL(14,2) NULL DEFAULT NULL COMMENT ''当月手动我方抽成，空表示按规则计算'' AFTER guaranteed_amount','SELECT 1') INTO @sql FROM information_schema.columns WHERE table_schema=@schema_name AND table_name='stall_turnovers' AND column_name='manual_own_commission'; PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
