-- 档口初始资料及 2026 年 9 月营业额，需先执行 sql/migrations/stall_commission.sql 和 stall_commission_v2.sql。
-- 负责人电话按要求留空；康莱德比例暂留0，由页面按月份手动填写。
SET NAMES utf8mb4;
SET @turnover_month = '2026-09-01';

INSERT INTO stall_accounts
    (stall_no,name,responsible_name,responsible_mobile,monthly_rent,default_guaranteed_amount,allocation_mode,profit_jia_rate,profit_yi_rate,guaranteed_rate,base_amount_rate,base_amount_factor,total_amount_rate,remark)
VALUES
    ('S01','每日鲜果','杨亮','',0,0,'custom',0,0,0,0,0,0,''),
    ('S02','益禾堂','王健','',0,0,'profit_share',0.90,0.10,0,0,0,0,''),
    ('S03','小杨哥钢闸肉饭','石','',0,0,'custom',0,0,0,0,0,0,''),
    ('S04','福鼎肉片夹馍','宋晓婉','',0,0,'custom',0,0,0,0,0,0,''),
    ('S05','大众快餐','林建锋','',0,0,'profit_share',0.90,0.10,0,0,0,0,''),
    ('S06','自选餐','白欣欣','',0,0,'profit_share',0.90,0.10,0,0,0,0,''),
    ('S07','郑记面馆','郑光辉','',0,0,'custom',0,0,0,0,0,0,''),
    ('S08','豪式云汉堡','黄增锋','',0,0,'custom',0,0,0,0,0,0,''),
    ('S09','国腿原麻辣烫','梁涛','',0,0,'custom',0,0,0,0,0,0,''),
    ('S10','黄小晨餐江湖饭','黄俊仕','',0,0,'custom',0,0,0,0,0,0,''),
    ('S11','疯狂烧烤','李梦春','',0,0,'custom',0,0,0,0,0,0,''),
    ('S12','羲辣小炒','蒋英慧','',0,0,'custom',0,0,0,0,0,0,''),
    ('S13','戌力行','陈友春','',0,0,'custom',0,0,0,0,0,0,''),
    ('S14','飘味特色美食','马得振','',0,0,'custom',0,0,0,0,0,0,'')
ON DUPLICATE KEY UPDATE
    name=VALUES(name),responsible_name=VALUES(responsible_name),monthly_rent=VALUES(monthly_rent),
    default_guaranteed_amount=VALUES(default_guaranteed_amount),allocation_mode=VALUES(allocation_mode),profit_jia_rate=VALUES(profit_jia_rate),profit_yi_rate=VALUES(profit_yi_rate),
    guaranteed_rate=VALUES(guaranteed_rate),base_amount_rate=VALUES(base_amount_rate),base_amount_factor=VALUES(base_amount_factor),
    total_amount_rate=VALUES(total_amount_rate),remark=VALUES(remark),active=1;

INSERT INTO stall_turnovers
    (stall_id,turnover_month,school_amount,card_amount,huilaimi_amount,cost_amount,guaranteed_amount,remark)
SELECT s.id,@turnover_month,v.card_amount,v.card_amount,v.huilaimi_amount,v.cost_amount,v.guaranteed_amount,''
FROM stall_accounts s
JOIN (
    SELECT 'S01' stall_no,162155.19 card_amount,0 huilaimi_amount,0 cost_amount,0 guaranteed_amount UNION ALL
    SELECT 'S02',1043.26,30552.41,0,0 UNION ALL
    SELECT 'S03',0,0,0,0 UNION ALL
    SELECT 'S04',0,0,0,0 UNION ALL
    SELECT 'S05',73665.38,0,0,0 UNION ALL
    SELECT 'S06',37589.50,0,0,0 UNION ALL
    SELECT 'S07',30542.47,7623.02,0,0 UNION ALL
    SELECT 'S08',11132.40,0,0,0 UNION ALL
    SELECT 'S09',11611.42,0,0,0 UNION ALL
    SELECT 'S10',13066.00,179.01,0,0 UNION ALL
    SELECT 'S11',12135.20,0,0,0 UNION ALL
    SELECT 'S12',11745.00,1116.00,0,0 UNION ALL
    SELECT 'S13',12645.00,0,0,0 UNION ALL
    SELECT 'S14',18591.50,0,0,0
) v ON v.stall_no=s.stall_no
ON DUPLICATE KEY UPDATE
    school_amount=VALUES(school_amount),card_amount=VALUES(card_amount),
    huilaimi_amount=VALUES(huilaimi_amount),cost_amount=VALUES(cost_amount),guaranteed_amount=VALUES(guaranteed_amount),active=1,deleted_at=NULL,deleted_by=NULL;

INSERT INTO stall_commission_settings(turnover_month,canteen_name,conrad_rate,remark)
VALUES(@turnover_month,'厦门理工知行2食堂',0,'康莱德公司比例待填写')
ON DUPLICATE KEY UPDATE canteen_name=VALUES(canteen_name);
