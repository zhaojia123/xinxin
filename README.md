# Friends Records

一个 Go 单体项目：后台 HTML、后台 JSON API、微信小程序 API 和 MySQL 数据访问由同一个程序提供。

微信小程序工程位于 [`miniprogram`](./miniprogram)，可直接用微信开发者工具导入；使用说明见 [`miniprogram/README.md`](./miniprogram/README.md)。

## 目录结构

结构参考 `go-mp-api-tide`，保留适合小项目的必要分层：

```text
friends-records/
├── main.go                       # 唯一启动入口
├── config.toml                   # 本地配置
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

程序不会自动建库、建表或插入演示数据。执行 `sql/schema.sql` 后所有页面最初都是空列表，统计金额为 `¥0.00`。

## 启动

1. 自行创建 MySQL 数据库。
2. 在数据库中手动执行 `sql/schema.sql`。
3. 修改根目录 `config.toml` 中的 `mysql.dsn`。
4. 启动：

```bash
cd /Users/mrzhao/go/friends-records
make run
```

当前配置监听 `:8999`，后台地址：<http://127.0.0.1:8999/admin/login>。

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
- `/admin/audit-logs`：操作日志

## 查询与联动接口

```text
GET/POST /admin/login
GET/POST/PUT/DELETE /api/admin/employees
PUT  /api/admin/employees/leave
GET/POST /api/admin/options
POST /api/admin/upload/health-certificate
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
GET  /api/admin/audit-logs
POST /api/mini/login
GET/POST/PUT /api/mini/ledger
GET/POST/PUT /api/mini/employees
GET/POST /api/mini/options
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
- 健康证照片默认保存到 `data/uploads/health-certificates`，可通过 `config.toml` 的 `[upload]` 修改目录和大小上限。
- 服务器部署时可将 `upload.dir` 写成绝对路径，例如 `/data/friends-records/uploads`，文件只会保存在你的服务器目录中，MySQL只保存路径和证件信息。
- `config.toml` 中的微信 AppSecret 已经在聊天和本地文件中出现，正式上线前建议在微信公众平台重置。
