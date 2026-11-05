# 客服工单协作平台

多客服协作处理工单：两种角色（客服、主管）、固定状态机、原子领取、SLA 逾期标记、完整操作历史。
后端 Go + PostgreSQL，前端 React + TypeScript。业务规则的唯一基准是 [docs/workflow.md](docs/workflow.md)，执行计划见 [docs/tasks.md](docs/tasks.md)。

## 1. 快速开始

前置依赖：Go 1.22+、Node 20+、Docker Desktop（Windows 需启用 WSL2 后端）。

```bash
npm run setup
npm run dev
```

这两条命令在 Windows（PowerShell、cmd）、macOS、Linux 上相同。`setup` 安装依赖并生成 `.env`；`dev` 启动数据库、执行迁移（含演示数据），再同时启动后端和前端。

- 前端：http://127.0.0.1:5173
- 后端：http://127.0.0.1:8080（只监听回环地址，Windows 不会弹防火墙提示）

演示用户（右上角切换；也可用 URL 参数 `?as=<id>` 在两个窗口里模拟两名客服）：

| id | 名称 | 角色 |
|---|---|---|
| 1 | 主管 王敏 | supervisor |
| 2 | 客服 李雷 | agent |
| 3 | 客服 韩梅 | agent |
| 4 | 客服 张伟 | agent |

常用命令：

| 命令 | 作用 |
|---|---|
| `npm run db:up` / `db:down` / `db:reset` | 启动、停止、删除数据卷后重建数据库 |
| `npm run seed:reset` | 把数据恢复成初始演示数据 |
| `npm run db:psql` | 进入 psql（无需本机安装） |
| `npm run check` | 编译、vet、gofmt、单元测试、前端类型检查 |
| `npm run test:integration` | 集成测试（需先 `db:up`） |
| `npm run test:race` | 在 Linux 容器里做数据竞争检测 |
| `npm run smoke` | 接口演示脚本（需先 `dev` 或 `dev:api`） |

### 不使用 Docker

安装 PostgreSQL 16，用安装时创建的 postgres 账号执行 `db/init-local.sql`（创建用户、开发库和测试库），
把 `.env` 里两个连接串的端口改为 5432，然后运行 `npm run dev:local`。

### 常见问题

- **端口 5433 或 8080 被占用**：修改 `.env` 中的 `DATABASE_URL`、`TEST_DATABASE_URL` 端口，或 `ADDR`；改数据库端口时同步改 `docker-compose.yml` 的端口映射。
- **终端中文乱码**：先执行 `chcp 65001`。
- **Docker Desktop 未启动**：`db:up` 会报无法连接 Docker daemon，先启动 Docker Desktop。
- **找不到 `docker-compose` 命令**：脚本使用独立的 `docker-compose`。若本机只有 `docker compose` 子命令，把 `package.json` 里 `db:*` 和 `test:race` 脚本中的 `docker-compose` 改成 `docker compose`。

## 2. 功能演示路径

数据来自迁移 `0002_seed`，`npm run seed:reset` 可随时恢复。

| 想演示什么 | 怎么做 |
|---|---|
| 领取 | 切到李雷，打开「发票抬头修改」（待处理，未分配），点「领取」 |
| 并发冲突 | 开两个窗口，分别用 `/?as=2` 和 `/?as=3` 打开同一张未分配工单，先后点「领取」。后点的窗口弹出「工单已被其他客服领取」，随后重新拉取详情，页面变为「处理中」、处理人是先点的客服，按钮消失（客服对详情只读可见，见第 3 节） |
| 分配 / 改派 | 切到王敏：未分配工单选客服后「分配」；「账号无法登录」（处理中）可「改派」，下拉里不含当前处理人 |
| 状态流转 | 李雷打开「退款进度查询」（已解决）可「重开」或「关闭」；「修改收货地址」（已关闭）没有任何按钮，评论框隐藏，只剩历史 |
| 权限拒绝 | 李雷打开「账号无法登录」（韩梅的）只能只读查看：没有操作按钮，评论框隐藏，直接调 resolve 接口得到 403 `NOT_ASSIGNEE`；李雷没有「分配」「改派」按钮，直接调接口得到 403 `FORBIDDEN` |
| 逾期标记 | 「支付失败无法下单」「物流长时间未更新」带红色「已逾期」；已解决、已关闭的工单永不逾期 |
| 命令行演示 | `npm run smoke` 走完整生命周期，再演示一次 403 和 409 |

## 3. 角色模型与身份

| 角色 | 能做什么 |
|---|---|
| 客服 `agent` | 创建工单；领取未分配工单；对**自己的**工单解决、重开、关闭、评论；列表只看到未分配工单和自己的工单；详情可只读查看任何工单（他人工单没有操作按钮） |
| 主管 `supervisor` | 创建工单；分配、改派；对任意工单解决、重开、关闭、评论；查看全部工单 |

前端每个请求带 `X-User-ID`，后端按它查 `users` 表取角色。请求体或请求头里自称的 `role`、`actor_id` 一律忽略（测试 I6）。
每个写操作都在服务层重新判断权限，前端隐藏按钮只是体验优化；详情页的按钮完全来自后端返回的 `allowed_actions`。

## 4. 状态机

```
            claim / assign                resolve
  open ─────────────────────▶ in_progress ─────────▶ resolved ──close──▶ closed（终态）
                                   ▲                    │
                                   └──── reopen ────────┘
```

合法流转只有 4 条，其余都被后端拒绝：

| 流转 | 操作 | 谁可以 |
|---|---|---|
| open → in_progress | claim | 任意客服（工单须未分配） |
| open → in_progress | assign | 主管（工单须未分配） |
| in_progress → resolved | resolve | 处理人或主管 |
| resolved → in_progress | reopen | 处理人或主管 |
| resolved → closed | close | 处理人或主管 |

另有两个不改变状态但受状态约束的操作：改派只允许 `in_progress`；评论允许除 `closed` 外的所有状态。

| 非法情形 | HTTP | code |
|---|---|---|
| 不在 4 条合法流转内（如 open 上 resolve、closed 上 reopen） | 409 | `INVALID_TRANSITION` |
| 工单已有处理人时 claim / assign（含并发失败方） | 409 | `ALREADY_CLAIMED` |
| 非 in_progress 改派；closed 上评论 | 409 | `INVALID_STATE` |
| 改派给当前处理人 | 409 | `SAME_ASSIGNEE` |

状态检查按「动作要求的起始状态」判断，而不是只调 `CanTransition(当前, 目标)`：reopen 的目标是 in_progress，而 open → in_progress 本身合法，只查后者会错误放行 open 上的 reopen（测试 `TestAuthorize_ReopenOnOpenRejected`）。
数据库另有兜底约束 `CHECK ((status = 'open') = (assignee_id IS NULL))`。

## 5. 领取并发策略

所有写操作共用一个模板：**事务内 `SELECT ... FOR UPDATE` 锁住工单行 → 在 Go 里判定 → UPDATE → 写事件 → 提交**。

```
客服A: BEGIN; SELECT ... FOR UPDATE  → 拿到锁，status=open
客服B: BEGIN; SELECT ... FOR UPDATE  → 阻塞等待
客服A: UPDATE; INSERT event; COMMIT   → 释放锁
客服B: 读到 A 提交后的最新行，status=in_progress → 409 ALREADY_CLAIMED; ROLLBACK
```

不用单条条件 `UPDATE ... WHERE status='open' AND assignee_id IS NULL`，原因：

- 0 行受影响时无法区分 404、403、409，需要再查一次，而两次查询之间状态可能又变；先锁再判断，读到的就是最终依据。
- 其他操作（分配、改派、解决、评论）要共用同一套规则；规则只写在 `domain` 的纯函数里，不在每条 SQL 的 WHERE 里重复。
- 代价是同一张工单上的写操作串行，工单粒度的冲突极少，可以接受。

评论也锁工单行，防止「客服刚被改派走，同时又在原工单上评论成功」。

对应测试：`TestConcurrentClaim`（20 个 goroutine、3 个客服同时领取同一工单，恰好 1 个成功、19 个 `ALREADY_CLAIMED`，库中 `claimed` 事件 1 条）、`TestConcurrentClaimVsAssign`（领取与分配并发只有一方成功）、`TestLockTicketBlocks`（行锁确实阻塞）。

## 6. SLA 口径

| 优先级 | urgent | high | normal | low |
|---|---|---|---|---|
| 时长 | 2 小时 | 8 小时 | 24 小时 | 72 小时 |

- `sla_due_at = created_at + 时长`，创建时计算并存库。创建时 service 只取一次时钟，同时写入 `created_at`、`updated_at` 和 `sla_due_at`。
- 逾期 `is_overdue = status IN (open, in_progress) AND now > sla_due_at`，**严格大于**：恰好等于截止时间不算逾期。不存库，由后端读出工单后用 `domain.IsOverdue` 统一计算，前端只显示。
- `resolved`、`closed` 永不逾期。reopen 后回到 `in_progress`，按**原截止时间**重新参与判断，SLA 不暂停也不重置。
- 优先级创建后不可修改，所以 `sla_due_at` 不会失效。
- 时间存 `timestamptz`（UTC），接口返回 RFC3339，前端按浏览器本地时区显示。

## 7. 接口列表

所有 `/api/*`（`/api/health` 除外）都要求 `X-User-ID`，缺失或无效返回 401。错误体统一为 `{"error":{"code","message","details"}}`。
写接口成功后返回更新后的工单详情，评论接口返回新建的评论。

| 方法 | 路径 | 权限 | 成功 | 主要错误 |
|---|---|---|---|---|
| GET | `/api/health` | 无需身份 | 200 | |
| GET | `/api/me` | 任意 | 200 | 401 |
| GET | `/api/users` | 任意 | 200 | 401 |
| GET | `/api/tickets` | 客服只见未分配+自己的；主管全部 | 200 | 400 `VALIDATION_ERROR` |
| POST | `/api/tickets` | 任意 | 201 | 400 `VALIDATION_ERROR` |
| GET | `/api/tickets/{id}` | 任意（客服看他人工单时 `allowed_actions` 为空） | 200 | 404 |
| POST | `/api/tickets/{id}/claim` | 客服 | 200 | 403 `FORBIDDEN`、409 `ALREADY_CLAIMED` |
| POST | `/api/tickets/{id}/assign` | 主管 | 200 | 403、409 `ALREADY_CLAIMED`、422 `INVALID_ASSIGNEE` |
| POST | `/api/tickets/{id}/reassign` | 主管 | 200 | 409 `INVALID_STATE` / `SAME_ASSIGNEE`、422 |
| POST | `/api/tickets/{id}/resolve` | 处理人或主管 | 200 | 403 `NOT_ASSIGNEE`、409 `INVALID_TRANSITION` |
| POST | `/api/tickets/{id}/reopen` | 处理人或主管 | 200 | 同上 |
| POST | `/api/tickets/{id}/close` | 处理人或主管 | 200 | 同上 |
| POST | `/api/tickets/{id}/comments` | 处理人或主管 | 201 | 403 `NOT_ASSIGNEE`、409 `INVALID_STATE` |

列表参数：`status`、`priority`（逗号分隔，逐个校验）、`assignee_id`（用户 ID 或 `none`）、`q`（标题或客户名，大小写不敏感，`%`/`_`/`\` 按字面匹配）、`page`（默认 1）、`page_size`（默认 20，上限 100，越界返回 400）。排序固定为 `created_at DESC, id DESC`。
所有写操作的错误顺序固定为：401 → 400 → 404 → 403 `FORBIDDEN` → 403 `NOT_ASSIGNEE` → 409 → 422，无权限的用户拿不到状态信息。

## 8. 测试

```bash
npm run check              # 编译、vet、gofmt、单元测试、前端类型检查
npm run test:integration   # 集成测试，需先 npm run db:up
npm run test:race          # Linux 容器里 go test -race（Windows 默认没有 cgo 编译器）
```

| 编号 | 覆盖内容 | 位置 |
|---|---|---|
| U1、U2 | 4×4 状态流转穷举；权限判定表 88 个用例逐格断言错误码 | `domain/policy_test.go` |
| U3、U4 | SLA 时长；逾期边界（截止前 1 秒、恰好、后 1 秒） | `domain/sla_test.go` |
| U5 | 输入校验（空白、200/201 个汉字、多字段、枚举不 trim） | `domain/validate_test.go` |
| I1、I2 | 并发领取、领取与分配并发 | `service/concurrency_test.go` |
| I3 | 完整生命周期，每步断言状态/处理人/事件，穿插失败调用 | `service/commands_test.go` |
| I7 | 事件写入失败时业务变更全部回滚 | `service/atomic_test.go` |
| I9 | 列表可见范围、筛选、搜索、逾期 | `service/queries_test.go`、`detail_test.go` |
| I4、I5、I6、I8 | HTTP 层状态码、权限、角色来源、错误格式 | `httpapi/api_test.go`、`api_errors_test.go` |
| — | 行锁阻塞、LIKE 转义、分页、事务回滚 | `store/store_test.go`、`list_test.go` |
| — | seed 数据用 `CanTransition` 回放一致 | `service/seed_test.go` |
| — | 错误码映射表无遗漏 | `httpapi/errors_test.go` |

集成测试读取 `TEST_DATABASE_URL`，未设置时直接失败而不是跳过，避免「没跑却显示通过」。各包共用一个测试库并会清表，所以用 `-p 1` 串行。

## 9. 设计取舍

| 决策 | 理由 |
|---|---|
| 主管也能 resolve / reopen / close / comment | 题面只限制了客服的处理范围，主管拥有全部工单的处理权 |
| 客服可只读查看任何工单详情，列表仍只显示未分配和自己的 | 并发领取的失败方刷新后能看到工单已被谁领取，而不是整页 403；写操作权限不变 |
| 先锁行再判断，错误顺序 404 → 403 → 409 | 无权限的用户先得到 403，不能通过 409 探测状态 |
| 目标用户不存在或非客服 → 422，与当前处理人相同 → 409 | 前者是请求内容错误，后者是业务冲突 |
| 动作型接口，不提供通用 `PATCH status` | 每个动作权限和前置条件不同，分开更好校验和测试 |
| 工单离开 open 后始终有处理人，没有「取消分配」 | 题面未要求；由数据库 CHECK 兜底 |
| 评论冻结于 closed | closed 是终态 |
| 不实现乐观锁 | 行锁已防止并发覆盖；旧页面上点按钮会得到 409 并自动刷新 |
| 优先级创建后不可改 | 题面未要求编辑，SLA 截止时间因此不会失效 |
| 联系方式只要求非空 | 不校验邮箱/电话格式 |
| 搜索用 `ILIKE '%关键词%'` | 演示数据量无需索引；数据增大后可加 `pg_trgm` GIN 索引 |
| 当前用户存于 URL 参数 `?as=` | 便于两个窗口模拟并发，不做 localStorage 持久化 |
| 任务入口用 npm scripts，不用 Makefile | 开发机是 Windows，npm scripts 在三个平台行为一致 |

## 10. 已完成、未完成、已知限制

**已完成**：两种角色与身份传递；创建、列表（8 个字段、状态/优先级/处理人筛选、搜索、分页）、详情；领取、分配、改派、解决、重开、关闭、评论；原子领取与并发测试；SLA 与逾期标记；操作历史（后端渲染中文说明）；迁移与演示数据；单元、集成、竞争检测测试。

**未做**（题面可选项）：乐观锁、实时推送、附件、工单编辑、通知、OpenAPI 文档。

**已知限制**：

- 身份可冒用：`X-User-ID` 没有口令或签名，任何人可以填别人的 ID。题面明确不要求完整登录，这里只保证角色取自数据库。
- 客服可通过工单 ID 读到他人工单的描述、联系方式和评论。列表范围受限只是不主动展示，不是保密措施；生产环境如需保密，应改回按归属限制查看。
- 「处理人必须是客服」只在 Go 中校验，数据库没有跨表约束，也没加触发器。
- `ILIKE` 搜索在数据量大时较慢。
- 无乐观锁：旧页面上的操作以后端最新状态为准，可能返回 409。

## 11. AI 使用说明

开发用 Claude（Claude Code 桌面版）完成。做法是先写规格再让 AI 实现：`docs/workflow.md` 定业务规则，`docs/tasks.md` 拆任务并给出层间接口和验收，AI 每次只拿一张任务卡和它引用的规格章节，不让它自行补充业务规则。
每个任务结束追加 DEVLOG 记录，包含给 AI 的输入、产出、人工修正和验收结果。详见 [docs/ai-log/](docs/ai-log/)。
