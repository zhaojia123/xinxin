-- 合并 2026-09 历史对账回填造成的供货商别名；先归并成本，再停用重复基础资料。
START TRANSACTION;

UPDATE stall_cost_entries
SET supplier_name = CASE supplier_name
    WHEN '秀澜鑫欢猪肉店' THEN '秀源鑫欢猪肉店'
    WHEN '群利鑫 （调味品）' THEN '群利鑫（调味品）'
    WHEN '豫集翔 （易耗品）' THEN '豫集翔（易耗品）'
    WHEN '阿湖水产' THEN '阿淘水产'
    ELSE supplier_name
END
WHERE supplier_name IN ('秀澜鑫欢猪肉店', '群利鑫 （调味品）', '豫集翔 （易耗品）', '阿湖水产');

UPDATE supplier_accounts
SET active = 0
WHERE name IN ('秀澜鑫欢猪肉店', '群利鑫 （调味品）', '豫集翔 （易耗品）', '阿湖水产');

COMMIT;
