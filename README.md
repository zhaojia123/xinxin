# Friends Records

一个 Go 单体项目：后台 HTML、后台 JSON API、微信小程序 API 和 MySQL 数据访问由同一个程序提供。

微信小程序工程位于 [`miniprogram`](./miniprogram)，可直接用微信开发者工具导入；使用说明见 [`miniprogram/README.md`](./miniprogram/README.md)。

## 目录结构

结构参考 `go-mp-api-tide`，保留适合小项目的必要分层：

```text
friends-records/
├── main.go                       # 唯一启动入口
├── config.local.toml             # 本地真实配置（不提交）
├── config.production.toml        # 线上真实配置（不提交）
├── config.*.example.toml         # 可提交的配置模板
├── api/
│   ├── request/                  # API 入参
│   └── response/                 # 页面与 API 出参
├── config/                       # TOML 配置加载
├── internal/
│   ├── handler/                  # 页面和接口控制器
│   ├── models/mysql/             # MySQL 查询与事务
│   ├── service/                  # 登录、工资等业务逻辑
│   ├── apperror/                 # 带文件和行号的中文错误
│   ├── httpx/                    # HTTP 公共响应
│   └── token/                    # 小程序 Token
├── router/
│   ├── admin.go                 # 后台页面与 /api/admin 路由
│   ├── mini.go                  # 微信小程序 /api/mini 路由
│   ├── upload.go                # 文件上传预留入口
│   └── router.go                # 公共路由组装
├── sql/schema.sql                # 手动执行的完整表结构
├── sql/migrations/               # 已有数据库的增量表结构
└── webassets/
    ├── templates/                # 后台 HTML 页面
    └── static/                   # CSS 和 JavaScript
```

## 数据关系

- 部门 `departments` 一对多岗位 `positions`，员工通过 ID 关联部门和岗位。
- 员工 `employees` 一对多人事异动、调薪、请假异常和月度工资条。
- 员工健康证单独保存在 `employee_health_certificates`，续证时保留历史照片和日期；员工列表提醒20天内到期及已过期证件。
- 月度工资由 `payroll_batches` 和 `payroll_items` 保存历史快照。
- 工资奖金扣款明细 `payroll_adjustments` 可以关联具体请假或异常记录。
- 工资发放接口在同一个数据库事务中创建台账支出，并通过 `payroll_batches.ledger_entry_id` 精确关联。
- 台账账户保存期初余额，每笔流水按所属账户计算结存。
- 台账转账凭证保存在 `ledger_attachments`，员工身份证/健康证图片保存在 `employee_attachments`，均支持同一对象多张图片和假删除。
- 每日供货采购明细保存在 `purchase_items`，按日期、供货商、品类、数量、单价和备注记录，支持后台与小程序导出 Excel。
- 小程序权限使用 `mini_user_permissions` 控制底部导航和对应接口；`mini_users.can_*` 仅作为旧数据兼容字段。
- 权限表同时支持查看、新增、编辑、删除四种操作权限，可只开放查看而禁止写入。
- 以后新增模块只需增加对应的 `module_key` 权限记录，不必修改用户表结构。

程序不会自动建库、建表或插入演示数据。执行 `sql/schema.sql` 后所有页面最初都是空列表，统计金额为 `¥0.00`。

## 启动

1. 自行创建 MySQL 数据库。
2. 在数据库中手动执行 `sql/schema.sql`。
   已有数据库升级时，另外执行 `sql/migrations/ledger_attachments.sql`、`sql/migrations/employee_attachments.sql`、`sql/migrations/purchase_items.sql`、`sql/migrations/mini_user_permissions.sql`、`sql/migrations/employee_payroll_rules.sql`、`sql/migrations/20260918_payroll_accounts.sql`、`sql/migrations/stall_monthly_expenses.sql`、`sql/migrations/stall_commission_v3.sql` 和 `sql/migrations/payroll_group_settlement.sql`。
3. 如需导入员工资料，执行 `sql/import/employee_import_202609.sql`。该文件只导入主表人员，当前全部按月薪处理；空工资和空日期会保留为空，不会导入“钟点工”工作表。
   生成 2026-09 工资后，如需录入表格里的两笔已结算金额，再执行 `sql/import/payroll_settlements_202609.sql`。
4. 复制 `config.local.example.toml` 为根目录 `config.local.toml`，再修改其中的 `mysql.dsn`。
   月薪统一按30天计薪，默认公休3天即按27个计薪日计算；公休天数默认由 `app.monthly_rest_days` 配置，员工档案可单独填写公休天数覆盖默认值（单休可填4/5，双休可填8/9），修改配置后重启服务即可生效。
5. 启动：

```bash
cd /Users/mrzhao/go/friends-records
make run
```

当前配置监听 `:8999`，后台地址：<http://127.0.0.1:8999/admin/login>。

### 线上部署与 502 排查

本地与线上无需来回修改同一份配置。程序会按 `config.local.toml`、`config.production.toml`、旧版 `config.toml` 的顺序自动读取第一份存在的配置：本地保留 `config.local.toml`，服务器只放置 `config.production.toml` 即可。也可以通过 `FRIENDS_RECORDS_CONFIG=/绝对路径/配置文件.toml` 明确指定配置文件。

首次线上部署时，将 `config.production.example.toml` 复制为服务器根目录的 `config.production.toml`，仅在这份服务器文件中填写数据库连接、`app.token_secret` 和微信密钥。线上配置必须保持 `app.env = "production"`、`wechat.mock_login = false` 和绝对上传目录；这些真实配置文件已加入 `.gitignore`。

本项目的 HTML 和 CSS 会嵌入 Go 二进制文件。服务器拉取最新代码后，不能只重启旧进程，必须先重新编译，再启动新二进制：

```bash
cd /root/xinxin
git pull
GOCACHE=/tmp/friends-records-go-cache make build
pkill -f '/root/xinxin/bin/friends-records' || true
nohup ./bin/friends-records > /tmp/friends-records.log 2>&1 &
```

随后先在服务器本机检查 Go 服务，不要直接从浏览器判断：

```bash
ss -lntp | grep 8999
curl -i http://127.0.0.1:8999/healthz
tail -n 100 /tmp/friends-records.log
```

正常时 `/healthz` 会返回 `200`。如果没有监听 8999，优先查看日志中的配置或 MySQL 连接错误；如果 Go 服务正常监听而域名仍返回 502，检查 Nginx 的 `proxy_pass` 是否指向 `http://127.0.0.1:8999`，修改后执行 `nginx -t && systemctl reload nginx`。

也可以直接运行：

```bash
GO111MODULE=on go run .
```

需要在 SQL 执行失败时显示准确行号，可以使用批处理方式执行：

```bash
mysql --show-warnings --verbose -u root -p friends_records < /Users/mrzhao/go/friends-records/sql/schema.sql
```

MySQL 会输出类似 `ERROR 1064 ... at line 90`。Go 服务的所有 HTTP 错误响应也会包含项目相对文件和行号；数据库错误还会附带 SQL 查询所在的 Go 文件、行号和 MySQL 原因。

## 页面

- `/admin/employees`：员工档案
- `/admin/organization`：部门与岗位
- `/admin/leaves`：请假、迟到和早退统计
- `/admin/changes`：人事异动
- `/admin/salary-adjustments`：调薪记录
- `/admin/payroll`：月度工资，可实时编辑并重新计算
- `/admin/ledger`：台账流水与年度统计
- `/admin/purchases`：每日供货采购清单，可按日期导出 Excel
- `/admin/mini-users`：小程序用户启用状态和模块权限管理
- `/admin/audit-logs`：操作日志

## 查询与联动接口

```text
GET/POST /admin/login
GET/POST/PUT/DELETE /api/admin/employees
PUT  /api/admin/employees/leave
GET/POST /api/admin/options
POST /api/admin/upload/health-certificate
POST /api/admin/upload/ledger-proof
POST /api/admin/upload/employee-document
GET  /api/admin/organization
GET/POST/PUT/DELETE /api/admin/departments
GET/POST/PUT/DELETE /api/admin/positions
GET/POST/PUT/DELETE /api/admin/attendance?period=month|quarter|half|year
PUT  /api/admin/attendance/status
GET/POST /api/admin/employment-changes
GET/POST /api/admin/salary-adjustments
GET  /api/admin/payroll?month=2026-09
PUT  /api/admin/payroll?id=工资明细ID
POST /api/admin/payroll/generate
POST /api/admin/payroll/confirm
POST /api/admin/payroll/pay
GET/POST/PUT/DELETE /api/admin/ledger?month=2026-09
GET/POST/PUT/DELETE /api/admin/purchases?start_date=2026-09-01&end_date=2026-09-01
GET /admin/purchases/export?start_date=2026-09-01&end_date=2026-09-01
GET  /api/admin/audit-logs
POST /api/mini/login
GET/POST/PUT /api/mini/ledger
GET/POST/PUT/DELETE /api/mini/purchases
GET /api/mini/purchases/export?start_date=2026-09-01&end_date=2026-09-01
GET/POST/PUT /api/mini/employees
GET/POST /api/mini/options
POST /api/mini/ledger-proof
POST /api/mini/employee-document
```

生成工资：

```json
{"month":"2026-09"}
```

编辑工资明细时，前端只提交组成项，服务端重新计算实发工资：

```json
{
  "base_salary": 8500,
  "bonus": 500,
  "attendance_deduction": 80,
  "other_deduction": 0,
  "social_security": 892,
  "tax": 126
}
```

工资发放并写入台账：

```json
{"batch_id":1,"account_id":1,"occurred_on":"2026-09-10"}
```

## 重要说明

- 后台目前按照此前要求不保存 Cookie，因此页面和后台 API 尚未做访问拦截。正式上线前必须增加后台 Session 或 Token 鉴权。
- MySQL 的 3306 端口不要暴露到公网。
- 健康证照片默认保存到 `data/uploads/health-certificates`，可通过当前环境配置文件的 `[upload]` 修改目录和大小上限。
- 台账凭证图片保存到 `data/uploads/ledger-proofs`，员工证件图片保存到 `data/uploads/employee-documents`；后台和小程序详情页会展示这些图片，列表页不增加图片内容。
- 服务器部署时可将 `upload.dir` 写成绝对路径，例如 `/data/friends-records/uploads`，文件只会保存在你的服务器目录中，MySQL只保存路径和证件信息。
- 请勿把微信 AppSecret、数据库密码或 token 密钥写入模板与代码库；如密钥曾泄露，正式上线前请在微信公众平台重置。
