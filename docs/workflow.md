# 题目B：客服工单协作平台 · 固定工作流规格

> 版本 v3.1 · 2026-11-05 · 依据：`AI_Coding_五类候选人测试题.md` 题目B原文
> v3.1 变更：客服可只读查看任何工单详情（第 0、4、9、15 节，17.1 判定表 view 列），列表范围与写操作权限不变
> 本文是后续开发的唯一业务基准。标注【决策】的条目是题面未规定、由本文确定的规则，前端、后端、测试、README 必须一致。

## 0. 设计要点

| 主题 | 规则 | 详见 |
|---|---|---|
| 创建者 | agent 和 supervisor 都可创建工单 | 4 |
| 关闭 | 只由用户手动关闭，没有系统自动关闭 | 3 |
| 处理权限 | agent 只能操作分配给自己的工单，supervisor 可操作任意工单 | 3、4 |
| 并发 | 所有写操作统一为"事务内 `SELECT ... FOR UPDATE` 锁行 → Go 规则校验 → UPDATE → 写历史 → 提交" | 5、7 |
| 错误区分 | 先锁行再判断，按顺序返回 404、403、409 | 5、9 |
| 客服可见范围 | 列表只显示待领取池 + 分配给自己的工单；详情对所有客服只读可见，写操作仍限处理人 | 4、11 |
| 身份传递 | `X-User-ID` 请求头，角色只从数据库读取 | 2 |

## 1. 角色与身份

| 角色 | 枚举值 | 一句话职责 |
|---|---|---|
| 客服 | `agent` | 领取未分配工单，处理分配给自己的工单 |
| 主管 | `supervisor` | 分配、改派，查看并操作全部工单 |

演示用户（迁移脚本预置）：

| id | 名称 | 角色 |
|---|---|---|
| 1 | 主管 王敏 | supervisor |
| 2 | 客服 李雷 | agent |
| 3 | 客服 韩梅 | agent |
| 4 | 客服 张伟 | agent |

## 2. 身份传递与鉴权规则

1. 前端在每个 `/api` 请求上带 `X-User-ID: <id>`，值来自页面顶部的"当前用户"切换器。
2. 后端中间件按该 ID 查 `users` 表，把 `{id, role}` 放入请求上下文。角色只取自数据库。
3. 请求体或 Query 中出现的 `role`、`actor_id` 等字段一律忽略。
4. 缺少该头或 ID 不是正整数 → 401；用户不存在 → 401。
5. 每个写操作在服务层重新做权限判断，前端隐藏按钮只是体验优化。
6. 防冒用不在范围内，README 中如实说明。

## 3. 状态机

```
            claim / assign                resolve
  open ─────────────────────▶ in_progress ─────────▶ resolved ──close──▶ closed（终态）
                                   ▲                    │
                                   └──── reopen ────────┘
```

合法流转只有 4 条，其余全部拒绝（409 `INVALID_TRANSITION`）：

| # | 流转 | 操作 | 允许的操作者 | 处理人变化 |
|---|---|---|---|---|
| T1 | open → in_progress | claim | 任意 agent，工单须未分配 | 设为自己 |
| T1 | open → in_progress | assign | supervisor，工单须未分配 | 设为指定 agent |
| T2 | in_progress → resolved | resolve | 当前处理人 agent，或 supervisor | 不变 |
| T3 | resolved → in_progress | reopen | 当前处理人 agent，或 supervisor | 不变 |
| T4 | resolved → closed | close | 当前处理人 agent，或 supervisor | 不变 |

不改变状态、但受状态约束的操作：

| 操作 | 允许状态 | 操作者 | 说明 |
|---|---|---|---|
| reassign | 仅 in_progress | supervisor | 目标须是 agent 且不同于当前处理人；resolved 须先 reopen |
| comment | 除 closed 外【决策】 | 处理人 agent，或 supervisor | closed 是终态，冻结一切写入 |

【决策】处理人规则：工单离开 open 后始终有处理人；reopen 回到原处理人；没有"取消分配"操作（题面未要求）。
【决策】Supervisor 可执行 resolve/reopen/close，因为题面只限制了客服的处理范围，主管拥有全部工单的处理权。

数据库保留一道兜底约束：`CHECK ((status = 'open') = (assignee_id IS NULL))`，即 open 必无处理人，非 open 必有处理人。

## 4. 权限矩阵

"自己的"指 `assignee_id = 当前用户`。✅ 允许，❌ 403。

| 操作 | agent | supervisor |
|---|---|---|
| 创建工单 | ✅ | ✅ |
| 查看列表 | 仅 open 未分配 + 自己的 | 全部 |
| 查看详情 | ✅ 全部，只读【决策】 | 全部 |
| claim | ✅（open 未分配） | ❌（主管走 assign） |
| assign | ❌ | ✅ |
| reassign | ❌ | ✅ |
| resolve / reopen / close | 仅自己的 | ✅ |
| comment | 仅自己的 | ✅ |
| 查看用户列表（切换器、分配下拉框） | ✅ | ✅ |

【决策】客服可以只读查看任何工单的详情（含他人工单），但列表只显示待领取池和自己的工单，写操作仍要求是处理人，他人工单的 `allowed_actions` 为空。原因：并发领取的失败方刷新后应看到工单已被谁领取，而不是整页 403。代价是客服可通过工单 ID 读到他人工单的描述、联系方式和评论；README 的已知限制中说明，生产环境如需保密可改回按归属限制查看。

## 5. 每个写操作的固定处理流程

所有写操作走同一个模板，顺序固定，测试按此顺序断言错误码：

```
1. 鉴权中间件：X-User-ID → users 表 → actor{id, role}；失败 401
2. 解析并校验请求体；失败 400 VALIDATION_ERROR（带字段级错误）
3. BEGIN
4. SELECT * FROM tickets WHERE id = $1 FOR UPDATE；无行 → 404 TICKET_NOT_FOUND
5. 角色检查（该角色能否执行此操作）；失败 403 FORBIDDEN
6. 归属检查（agent 是否为处理人，claim 除外）；失败 403 NOT_ASSIGNEE
7. 状态检查（状态机 + 未分配等前置条件）；失败 409（具体码见第 9 节）
8. 目标用户检查（assign/reassign：存在且为 agent，且不等于当前处理人）；不存在 422 / 非 agent 422 / 相同 409
9. UPDATE tickets SET ..., updated_at = now()
10. INSERT INTO ticket_events (...)
11. COMMIT；任一步出错 ROLLBACK
```

【决策】第 5、6 步早于第 7 步：无权限的用户先得到 403，不能通过 409 探测工单状态。
【决策】第 8 步中目标用户不存在或不是 agent 属于请求内容错误，返回 422 `INVALID_ASSIGNEE`；与当前处理人相同属于业务冲突，返回 409 `SAME_ASSIGNEE`。

状态机规则写成纯函数，便于单测，服务层第 5～7 步直接调用：

```go
func CanTransition(from, to Status) bool
func Authorize(actor User, t Ticket, a Action) error   // 完整判定表见 17.1
```

## 6. 操作定义

| 操作 | 接口 | 请求体 | 前置条件（第 7 步） | 写入 | 事件 |
|---|---|---|---|---|---|
| 创建 | `POST /api/tickets` | title, description, customer_name, customer_contact, priority | 无 | status=open, assignee=NULL, sla_due_at | `created` |
| 领取 | `POST /api/tickets/{id}/claim` | 无 | status=open 且 assignee IS NULL，否则 409 `ALREADY_CLAIMED` | assignee=actor, status=in_progress | `claimed` |
| 分配 | `POST /api/tickets/{id}/assign` | assignee_id | 同上 | assignee=目标, status=in_progress | `assigned` |
| 改派 | `POST /api/tickets/{id}/reassign` | assignee_id | status=in_progress，否则 409 `INVALID_STATE` | assignee=目标 | `reassigned` |
| 解决 | `POST /api/tickets/{id}/resolve` | note（可选） | status=in_progress | status=resolved | `status_changed` |
| 重开 | `POST /api/tickets/{id}/reopen` | note（可选） | status=resolved | status=in_progress | `status_changed` |
| 关闭 | `POST /api/tickets/{id}/close` | note（可选） | status=resolved | status=closed | `status_changed` |
| 评论 | `POST /api/tickets/{id}/comments` | body | status≠closed | 插入 comments | `commented` |

【决策】状态流转用"动作型"接口（claim/resolve/...），不提供通用 `PATCH status`。每个动作的权限和前置条件不同，分开写更容易校验和测试。

字段校验（第 2 步）：

| 字段 | 规则 |
|---|---|
| title | 去首尾空白后 1～200 字符 |
| description | 去首尾空白后 1～5000 字符 |
| customer_name | 1～100 字符 |
| customer_contact | 1～200 字符；【决策】不校验邮箱/电话格式，只要求非空 |
| priority | urgent / high / normal / low 之一 |
| assignee_id | 正整数 |
| comment body / note | body 1～2000 字符；note 0～500 字符 |

事件 `payload`（JSONB）内容：

| 事件 | payload |
|---|---|
| created | `{priority}` |
| claimed | `{assignee_id}` |
| assigned | `{assignee_id}` |
| reassigned | `{from_assignee_id, to_assignee_id}` |
| status_changed | `{from, to, note}` |
| commented | `{comment_id, excerpt}`（前 100 字符） |

历史展示时把 payload 渲染为一句说明，例如"王敏 将工单从 李雷 改派给 韩梅"。

## 7. 并发策略

采用悲观行锁：每个写操作在事务内先 `SELECT ... FOR UPDATE` 锁住工单行（第 5 节第 4 步），再做所有校验和写入。

两名客服同时领取同一工单时的过程（PostgreSQL 默认 READ COMMITTED）：

```
客服A: BEGIN; SELECT ... FOR UPDATE  → 拿到锁，status=open
客服B: BEGIN; SELECT ... FOR UPDATE  → 阻塞等待
客服A: UPDATE ... ; INSERT event ; COMMIT → 释放锁
客服B: 读到 A 提交后的最新行，status=in_progress → 409 ALREADY_CLAIMED ; ROLLBACK
```

选择行锁而非单条条件 UPDATE（`UPDATE ... WHERE status='open' AND assignee_id IS NULL`）的理由：

| 维度 | 条件 UPDATE | FOR UPDATE 行锁（采用） |
|---|---|---|
| 领取正确性 | 正确 | 正确 |
| 区分 404/403/409 | 0 行时需再查一次，且两次查询之间状态可能又变 | 锁定后读到的就是最终判断依据 |
| 复用到其他操作 | 每个操作要单独写 WHERE 条件 | 所有操作共用一个模板，规则只写在 Go 纯函数里 |
| 代价 | 无等待 | 同一工单上的写操作串行；工单粒度的锁冲突极少，可接受 |

评论也锁工单行，防止"客服刚被改派走，同时又在原工单上评论成功"。

【决策】不实现乐观锁（version 字段）。它属于可选扩展；行锁已经防止了并发覆盖。代价是"用户 A 看着旧页面点按钮"时，后端按最新状态判断，可能返回 409，前端提示后刷新详情。

## 8. SLA 口径

- `sla_due_at = created_at + 时长`，在创建时计算并存库（便于按截止时间排序和索引）。
- 时长：urgent 2h，high 8h，normal 24h，low 72h。
- 【决策】优先级创建后不可修改（题面未要求编辑工单），所以 `sla_due_at` 不会失效。
- 逾期 `is_overdue = status IN (open, in_progress) AND now > sla_due_at`，不存库。后端读出工单后在 Go 中统一计算并随响应返回，前端不自行计算，避免两处口径不一致。
- 【决策】resolved 后 reopen 回到 in_progress，按原截止时间重新参与逾期判断，SLA 不暂停也不重置。
- 时间统一存 `timestamptz`（UTC），API 返回 RFC3339，前端按浏览器本地时区显示。
- SLA 计算写成 `func SLADueAt(createdAt time.Time, p Priority) time.Time` 和 `func IsOverdue(t Ticket, now time.Time) bool`，`now` 作为参数传入以便测试。

## 9. 错误响应

统一格式：

```json
{ "error": { "code": "ALREADY_CLAIMED", "message": "工单已被其他客服领取", "details": {} } }
```

| HTTP | code | 场景 |
|---|---|---|
| 400 | VALIDATION_ERROR | 字段缺失、超长、枚举非法、JSON 格式错误；`details` 为 `{字段: 原因}` |
| 401 | UNAUTHENTICATED | 缺少或无效 `X-User-ID`，用户不存在 |
| 403 | FORBIDDEN | 角色不允许该操作（如 agent 调用 assign，supervisor 调用 claim） |
| 403 | NOT_ASSIGNEE | agent 操作或评论不属于自己的工单 |
| 404 | TICKET_NOT_FOUND | 工单 ID 不存在 |
| 409 | ALREADY_CLAIMED | claim/assign 时工单已有处理人（含并发失败方） |
| 409 | INVALID_TRANSITION | 不在 4 条合法流转内的状态变化 |
| 409 | INVALID_STATE | reassign 时非 in_progress；在 closed 工单上评论 |
| 409 | SAME_ASSIGNEE | 改派目标与当前处理人相同 |
| 422 | INVALID_ASSIGNEE | 目标用户不存在或不是 agent |
| 500 | INTERNAL | 未预期错误；日志记录细节，响应不暴露 |

前端按 `code` 显示中文提示；409 时自动重新拉取工单详情，让用户看到最新状态。

## 10. 数据模型（PostgreSQL）

```sql
CREATE TYPE user_role       AS ENUM ('agent', 'supervisor');
CREATE TYPE ticket_status   AS ENUM ('open', 'in_progress', 'resolved', 'closed');
CREATE TYPE ticket_priority AS ENUM ('urgent', 'high', 'normal', 'low');

CREATE TABLE users (
  id         BIGSERIAL PRIMARY KEY,
  name       TEXT NOT NULL,
  role       user_role NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tickets (
  id               BIGSERIAL PRIMARY KEY,
  title            TEXT NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  description      TEXT NOT NULL CHECK (length(btrim(description)) BETWEEN 1 AND 5000),
  customer_name    TEXT NOT NULL CHECK (length(btrim(customer_name)) BETWEEN 1 AND 100),
  customer_contact TEXT NOT NULL CHECK (length(btrim(customer_contact)) BETWEEN 1 AND 200),
  priority         ticket_priority NOT NULL,
  status           ticket_status NOT NULL DEFAULT 'open',
  assignee_id      BIGINT REFERENCES users(id),
  created_by       BIGINT NOT NULL REFERENCES users(id),
  sla_due_at       TIMESTAMPTZ NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT open_iff_unassigned CHECK ((status = 'open') = (assignee_id IS NULL))
);

CREATE TABLE ticket_comments (
  id         BIGSERIAL PRIMARY KEY,
  ticket_id  BIGINT NOT NULL REFERENCES tickets(id),
  author_id  BIGINT NOT NULL REFERENCES users(id),
  body       TEXT NOT NULL CHECK (length(btrim(body)) BETWEEN 1 AND 2000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ticket_events (
  id         BIGSERIAL PRIMARY KEY,
  ticket_id  BIGINT NOT NULL REFERENCES tickets(id),
  actor_id   BIGINT NOT NULL REFERENCES users(id),
  type       TEXT NOT NULL CHECK (type IN
             ('created','claimed','assigned','reassigned','status_changed','commented')),
  payload    JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tickets_status_created   ON tickets (status, created_at DESC);
CREATE INDEX idx_tickets_assignee         ON tickets (assignee_id);
CREATE INDEX idx_tickets_priority         ON tickets (priority);
CREATE INDEX idx_comments_ticket          ON ticket_comments (ticket_id, created_at);
CREATE INDEX idx_events_ticket            ON ticket_events (ticket_id, created_at);
```

【决策】评论和历史只追加、不可编辑删除，满足审计用途。外键不加 `ON DELETE CASCADE`，因为没有删除工单的操作。
【决策】搜索用 `ILIKE '%关键词%'`，演示数据量下无需索引；README 说明数据增大后可加 `pg_trgm` GIN 索引。

迁移工具：`golang-migrate`，目录 `backend/migrations/`，`0001_init.up.sql` 建表，`0002_seed.up.sql` 插入演示数据。

演示工单（seed）需覆盖每种状态和逾期情况，`created_at` 用 `now() - interval` 构造：

| 标题示例 | 优先级 | 状态 | 处理人 | 构造目的 |
|---|---|---|---|---|
| 支付失败无法下单 | urgent | open | 无 | 已逾期（创建于 3 小时前），可演示领取 |
| 发票抬头修改 | normal | open | 无 | 未逾期，可演示并发领取 |
| 物流长时间未更新 | high | in_progress | 李雷 | 已逾期（10 小时前） |
| 账号无法登录 | high | in_progress | 韩梅 | 可演示改派 |
| 退款进度查询 | normal | resolved | 李雷 | 可演示重开、关闭；逾期标记不显示 |
| 修改收货地址 | low | closed | 张伟 | 演示终态，所有按钮禁用 |

每条 seed 工单同时插入对应的历史事件，保证详情页历史完整。

## 11. 读接口

| 接口 | 说明 |
|---|---|
| `GET /api/users` | 返回全部用户 `{id, name, role}`，供切换器和分配下拉框 |
| `GET /api/me` | 返回当前 `X-User-ID` 对应的用户 |
| `GET /api/tickets` | 列表，参数见下表 |
| `GET /api/tickets/{id}` | 详情，含 comments 与 events（按时间升序）及当前用户可执行的 `allowed_actions` |

列表参数：

| 参数 | 取值 | 说明 |
|---|---|---|
| status | 枚举，可多选（逗号分隔） | |
| priority | 枚举，可多选 | |
| assignee_id | 用户 ID 或 `none` | `none` 表示未分配 |
| q | 字符串 | 标题或客户名称 `ILIKE` 匹配 |
| page / page_size | 默认 1 / 20，page_size 上限 100 | 超出范围返回 400 |

返回 `{items, total, page, page_size}`。每个 item 含：id, title, customer_name, priority, status, assignee{id,name}, sla_due_at, is_overdue, created_at。
【决策】默认排序 `created_at DESC, id DESC`（加 id 保证分页稳定）。
【决策】agent 调用列表时，后端在 SQL 中强制追加 `(status='open' AND assignee_id IS NULL) OR assignee_id = 当前用户`，再叠加用户给的筛选条件。

`allowed_actions` 由后端用与写操作相同的 `Authorize` 函数计算，前端只按它显示按钮，不自己推导权限。

## 12. 页面

| 页面 | 路由 | 内容 |
|---|---|---|
| 顶栏 | 全局 | 当前用户下拉切换（显示角色），切换后刷新当前页数据 |
| 工单队列 | `/` | 筛选栏（状态、优先级、处理人、关键词）、表格（第 11 节字段，逾期标红）、分页；空状态、加载状态 |
| 新建工单 | `/tickets/new` | 表单 + 前端校验（与后端规则一致）；提交成功跳转详情 |
| 工单详情 | `/tickets/:id` | 基本信息、SLA 倒计时/逾期标记；操作区按 `allowed_actions` 渲染（领取、分配/改派下拉、解决、重开、关闭）；评论列表与输入框；历史时间线 |

前端反馈：所有写操作按钮提交时禁用并显示加载；成功 toast；失败按错误码显示中文信息，400 时在对应字段下显示错误。

## 13. 自动化测试清单

单元测试（纯函数，无数据库）：

| 编号 | 测试内容 |
|---|---|
| U1 | `CanTransition`：4 条合法流转返回 true；其余 12 种组合（含 closed→任意、自身→自身）返回 false，表驱动覆盖全部 16 种 |
| U2 | `Authorize`：按第 4 节矩阵，对每个操作 × 角色 × 是否处理人 × 状态的组合断言返回的错误类型 |
| U3 | `SLADueAt`：4 个优先级的时长 |
| U4 | `IsOverdue`：截止前、恰好等于截止、截止后；open/in_progress 逾期，resolved/closed 永不逾期 |
| U5 | 输入校验：空白标题、超长、非法优先级 |

集成测试（连真实 PostgreSQL，`TEST_DATABASE_URL`，每个用例前清表）：

| 编号 | 测试内容 |
|---|---|
| I1 | **并发领取**：1 张 open 工单，N=20 个 goroutine 以不同 agent 同时 claim（用 `sync.WaitGroup` + 起跑闸门 channel 对齐）；断言恰好 1 个成功、19 个 `ALREADY_CLAIMED`；数据库中 assignee 等于成功者；`claimed` 事件恰好 1 条 |
| I2 | 并发 claim 与 assign：agent 领取和主管分配同时进行，只有一方成功 |
| I3 | 完整生命周期：create → claim → resolve → reopen → reassign → resolve → close，每一步检查状态、处理人、事件条数与类型 |
| I4 | 非法流转经 HTTP 返回 409：open 直接 resolve、in_progress 直接 close、closed 上 reopen/reassign/assign/comment |
| I5 | 权限：agent 调 assign/reassign → 403；agent 对他人工单 resolve/comment → 403；supervisor 调 claim → 403 |
| I6 | 角色来自数据库：agent 的请求体里夹带 `"role":"supervisor"` 调 assign，仍返回 403 |
| I7 | 原子性：注入一个写事件必定失败的 store（或在事务内让事件插入违反约束），断言工单状态未改变 |
| I8 | 404、401、400、422 各一例 |
| I9 | 列表：agent 看不到他人工单；筛选、搜索、分页的 total 正确；逾期字段正确 |

命令（Windows、macOS、Linux 通用）：`npm run test` 跑单元测试；`npm run db:up` 启动测试库后，`npm run test:integration` 跑集成测试。集成测试文件带 `//go:build integration` 标签。

## 14. 开发顺序（6～8 小时预算）

细化的任务卡、层间接口和验收标准见 `题目B_开发任务分解.md`，以该文档为执行依据。下表只是阶段概览。

| 阶段 | 产出 | 预计 | 完成标准 |
|---|---|---|---|
| 0～1 | 仓库骨架、npm scripts、docker-compose（postgres）、迁移与 seed | 1.4h | `npm run setup; npm run dev` 后能看到演示数据 |
| 2 | domain 包：状态机、权限、SLA 纯函数 + U1～U5 | 1.1h | `npm run test` 通过 |
| 3 | store + service：写操作模板、事务、行锁 + I1、I2、I3、I7、I9 | 1.9h | 并发领取测试 `-count=20` 稳定通过 |
| 4 | HTTP 层：路由、鉴权中间件、错误映射 + I4、I5、I6、I8 | 0.8h | `npm run smoke` 走通第 13 节 I3 流程 |
| 5 | React 前端：顶栏切换、列表、新建、详情 | 2.0h | 浏览器手工走通全部操作 |
| 6 | 端到端验收、README、AI 记录整理 | 0.8h | 第 16 节清单全部勾选 |

目录约定：

```
backend/   cmd/server, internal/{domain,service,store,httpapi}, migrations/
frontend/  src/{api,pages,components}
docs/ai-log/   AI 对话导出与操作日志
db/       init.sql（Docker 初始化）, init-local.sql（本机 PostgreSQL）
scripts/  setup-env.mjs, dev-api.mjs, check-fmt.mjs, smoke.mjs
README.md  package.json（npm scripts 任务入口）  docker-compose.yml  .env.example  .gitattributes
```

【决策】开发机为 Windows，任务入口用 npm scripts 而非 Makefile，辅助脚本用 Node 或 Go 编写，不写 shell 脚本，保证 Windows、macOS、Linux 使用同一组命令。详见任务分解文档 0.1 节。

技术选型【决策】：Go 1.22+ 标准库 `net/http` 路由 + `pgx/v5`；golang-migrate；前端 Vite + React + TypeScript，不引入大型组件库。

## 15. README 必须包含的章节

角色模型（第 1、2、4 节摘要）、状态机图与流转表（第 3 节）、领取并发策略（第 7 节）、SLA 口径（第 8 节）、启动方式（一条命令起库、迁移、后端、前端）、测试运行方式、设计取舍（所有【决策】条目）、已完成项、未完成项、已知限制（身份可冒用、客服可读他人工单详情、ILIKE 搜索性能、无乐观锁）。

## 16. 需求追溯（交付前自查）

| 题面要求 | 本文位置 |
|---|---|
| 两种角色、前端切换用户、后端从库读角色、忽略客户端角色 | 1、2、I6 |
| 工单字段；新工单 open 且无处理人 | 6、10 |
| 列表字段、按状态/优先级/处理人筛选、搜索、分页 | 11、12、I9 |
| 客服领取 open 未分配工单，原子设置处理人并进入 in_progress | 3、6、7 |
| 并发领取只有一人成功 | 7、I1、I2 |
| 客服只能处理自己的工单 | 4、I5 |
| 主管分配（原子进入 in_progress）、改派（状态不变） | 3、6 |
| resolved 不能直接改派；closed 不能分配/改派/重开 | 3、9、I4 |
| 业务冲突返回 409 | 9 |
| 4 条合法流转，其余后端拒绝 | 3、U1、I4 |
| SLA 时长、逾期口径、查询时计算 | 8、U3、U4 |
| 评论权限 | 4、I5 |
| 记录创建、领取、分配、改派、状态变化、评论事件；展示操作者、类型、时间、说明 | 6、12 |
| 业务变更与历史写入原子一致 | 5、I7 |
| 迁移、演示用户与工单 | 10 |
| 状态机和并发领取的自动化测试 | 13 |
| 验证、权限、不存在、冲突的清晰反馈 | 9、12 |
| README 指定章节 | 15 |
| AI 对话记录/操作日志 | 14（docs/ai-log） |

## 17. 补充规则（任务分解时确定）

### 17.1 `Authorize` 完整判定表

按第 5 节顺序（角色 → 归属 → 状态）得出的结果。✓ 表示允许；F = 403 `FORBIDDEN`，N = 403 `NOT_ASSIGNEE`，AC = 409 `ALREADY_CLAIMED`，IT = 409 `INVALID_TRANSITION`，IS = 409 `INVALID_STATE`。open 工单没有处理人，所以只有"任意 agent"一行。

| 操作者 / 状态 | view | claim | assign | reassign | resolve | reopen | close | comment |
|---|---|---|---|---|---|---|---|---|
| supervisor / open | ✓ | F | ✓ | IS | IT | IT | IT | ✓ |
| supervisor / in_progress | ✓ | F | AC | ✓ | ✓ | IT | IT | ✓ |
| supervisor / resolved | ✓ | F | AC | IS | IT | ✓ | ✓ | ✓ |
| supervisor / closed | ✓ | F | AC | IS | IT | IT | IT | IS |
| 任意 agent / open | ✓ | ✓ | F | F | N | N | N | N |
| 处理人 agent / in_progress | ✓ | AC | F | F | ✓ | IT | IT | ✓ |
| 处理人 agent / resolved | ✓ | AC | F | F | IT | ✓ | ✓ | ✓ |
| 处理人 agent / closed | ✓ | AC | F | F | IT | IT | IT | IS |
| 非处理人 agent / in_progress | ✓ | AC | F | F | N | N | N | N |
| 非处理人 agent / resolved | ✓ | AC | F | F | N | N | N | N |
| 非处理人 agent / closed | ✓ | AC | F | F | N | N | N | N |

实现注意：状态检查按"动作要求的起始状态"判断，不能只调 `CanTransition(当前状态, 目标状态)`。例如 reopen 的目标是 in_progress，而 open → in_progress 本身合法，若只查 `CanTransition` 会错误地允许在 open 工单上 reopen。每个流转动作的起始状态固定为：resolve 要求 in_progress，reopen、close 要求 resolved；不满足时返回 IT。

推导依据：claim 只允许 agent，assign、reassign 只允许 supervisor（角色）；view 和 claim 始终豁免归属检查，其余动作要求是处理人（归属）；状态检查见第 3、6 节。assign、reassign 通过后还要执行第 5 节第 8 步的目标用户检查。

### 17.2 其他补充规则

| 主题 | 规则 |
|---|---|
| 时间来源 | service 注入时钟 `now func() time.Time`。创建工单时用同一个 `now` 写入 created_at、updated_at 和 sla_due_at，逾期判断也用它，测试可固定时间 |
| 成功响应 | 创建工单、添加评论返回 201，其余写操作返回 200；写操作返回更新后的工单详情（评论接口返回新评论） |
| 未知字段 | 请求体中的未知字段（如 `role`）静默忽略；请求体上限 64KB |
| 长度计算 | 按 Unicode 字符数，不按字节；枚举值不做 trim |
| 搜索转义 | `q` 中的 `%`、`_`、`\` 先转义，再作为 ILIKE 子串匹配 |
| agent 筛选他人 | agent 传 `assignee_id=其他客服` 时返回空列表，不报错 |
| 前端当前用户 | 存于 URL 参数 `?as=<id>`，默认 1；便于两个窗口模拟两名客服并发领取 |
| 前端按钮 | 只按详情接口返回的 `allowed_actions` 渲染，`view` 不出现在其中 |

### 17.3 历史事件说明模板

后端渲染为 `summary` 字段，前端直接显示。`{actor}` 为操作者姓名。

| 事件 | 模板 |
|---|---|
| created | `{actor} 创建了工单，优先级 {priority 中文}` |
| claimed | `{actor} 领取了工单` |
| assigned | `{actor} 将工单分配给 {assignee}` |
| reassigned | `{actor} 将工单从 {from} 改派给 {to}` |
| status_changed | `{actor} 将状态从 {from 中文} 改为 {to 中文}`；有备注时追加 `：{note}` |
| commented | `{actor} 发表评论：{excerpt}` |

中文名：urgent 紧急、high 高、normal 普通、low 低；open 待处理、in_progress 处理中、resolved 已解决、closed 已关闭。

### 17.4 数据库补充约束

在第 10 节建表 SQL 的 `tickets` 上追加：

```sql
CONSTRAINT sla_after_created CHECK (sla_due_at > created_at),
CONSTRAINT updated_not_before_created CHECK (updated_at >= created_at)
```

"处理人必须是 agent"需要跨表判断，CHECK 无法表达。本项目只在 Go 中校验（第 5 节第 8 步），不加触发器，README 的已知限制中说明。
