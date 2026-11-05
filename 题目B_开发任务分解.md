# 题目B：客服工单协作平台 · 开发任务分解

> 版本 v1.1 · 2026-11-05 · 开发机为 Windows 11，所有命令在 Windows 上可执行，同时保证评审者在 macOS/Linux 上也能复现
> 业务规则以 `题目B_客服工单固定工作流.md`（下称 WF）为准，本文引用其节号，例如 WF§5 指第 5 节。本文只规定"按什么顺序做、每一步做到什么程度算完成"。
> 拆分时发现的补充规则已写入 WF§17，两份文档保持一致。

## 0. 使用方式

1. 按任务编号顺序执行。每个任务对应一次独立的 AI 协作会话和一次 git 提交，提交信息以任务编号开头，例如 `T2.2 状态机与权限纯函数`。
2. 开始任务前，把任务卡和它引用的 WF 章节一并交给 AI。不要让 AI 自行补充业务规则；遇到 WF 未覆盖的情况，先改 WF，再写代码。
3. 任务卡中的"验收"必须全部通过才进入下一个任务。验收不过时，在当前任务内修，不带着问题往后走。
4. 每个任务结束时在 `docs/ai-log/DEVLOG.md` 追加一条记录（模板见第 5 节）。
5. "不做"一栏是范围边界。超出的想法记到 DEVLOG 的"待定"里，不在当前任务实现。

## 0.1 运行环境约定（Windows）

开发机是 Windows，评审者的系统未知。因此项目不依赖 make、bash、curl 等 Linux 工具，所有命令只用 Node（npm scripts）、Go 和 Docker 三样跨平台工具。文档中的命令默认在 PowerShell 中执行。

| 项目 | 约定 | 原因 |
|---|---|---|
| 任务入口 | 根目录 `package.json` 的 npm scripts，替代 Makefile | Windows 默认没有 make；npm scripts 在 cmd、PowerShell、bash 中行为一致 |
| 辅助脚本 | 用 Go（`backend/cmd/*`）或 Node（`scripts/*.mjs`）编写，不写 `.sh`、`.ps1` | 跨平台；避开 PowerShell 5.1 默认非 UTF-8 编码导致的中文乱码 |
| 数据库 | 首选 Docker Desktop（WSL2 后端）运行 `postgres:16`；无法使用 Docker 时，用 PostgreSQL 16 Windows 安装包，见 T0.2 | 两种方式都只需改 `.env` 中的连接串 |
| 数据库端口 | 宿主机端口默认 `5433` | 避免和本机已安装的 PostgreSQL（默认 5432）冲突 |
| 后端监听 | `127.0.0.1:8080`，不监听 `0.0.0.0` | 避免每次 `go run` 生成新 exe 时弹出 Windows 防火墙提示 |
| 前端代理 | Vite 代理目标写 `http://127.0.0.1:8080`，不写 `localhost` | Node 17+ 在 Windows 上常把 `localhost` 解析为 IPv6 `::1`，而后端只监听 IPv4，会导致代理报 `ECONNREFUSED` |
| 环境变量 | 后端启动和测试时自动读取根目录 `.env`（`github.com/joho/godotenv`），已存在的系统环境变量优先 | PowerShell 设置临时环境变量的语法（`$env:X="..."`）和 bash 不同，自动读取可免去这一步 |
| Compose 命令 | npm scripts 一律写独立命令 `docker-compose`，不写 `docker compose` 子命令 | 开发机的 docker CLI 识别不了 `docker compose` 插件（报 `unknown command`），独立的 `docker-compose` v5.5.1 可用；Docker Desktop 通常两者都带，评审者机器上若只有子命令，见 T6.2 |
| 换行符 | 根目录 `.gitattributes` 写 `* text=auto eol=lf` | 防止 Git 在 Windows 上把文件转成 CRLF：容器内的初始化脚本遇到 CRLF 会执行失败，`gofmt -l` 也会把 CRLF 文件全部列为未格式化 |
| 终端编码 | 测试输出出现中文乱码时，先执行 `chcp 65001` | Windows 终端默认代码页为 936（GBK） |
| 路径 | 代码中用 `filepath.Join`，不手写 `/` 或 `\`；仓库放在不含中文和空格的路径下（如 `D:\AGENT\Customer\Customer_service\ticketdesk`） | 避免部分工具在中文路径下出错 |

## 1. 任务总览

| 编号 | 环节 | 任务 | 依赖 | 工时 | 关键产出 |
|---|---|---|---|---|---|
| T0.1 | 环境 | 仓库骨架与工具链 | 无 | 0.3h | 目录结构、go.mod、Vite 工程、.gitignore |
| T0.2 | 环境 | 本地运行环境 | T0.1 | 0.5h | docker-compose.yml、npm scripts、.env.example、Node 辅助脚本 |
| T1.1 | 数据库 | 建表迁移 | T0.2 | 0.3h | `0001_init.up/down.sql` |
| T1.2 | 数据库 | 演示数据 | T1.1 | 0.3h | `0002_seed.up/down.sql` |
| T2.1 | 领域层 | 类型、错误与输入校验 | T0.1 | 0.4h | `domain/types.go`、`errors.go`、`validate.go` |
| T2.2 | 领域层 | 状态机与权限判定 | T2.1 | 0.5h | `domain/policy.go` + U1、U2 |
| T2.3 | 领域层 | SLA 计算 | T2.1 | 0.2h | `domain/sla.go` + U3、U4 |
| T3.1 | 存储层 | 连接池、事务、SQL 查询 | T1.1、T2.1 | 0.6h | `store/*.go` |
| T3.2 | 服务层 | 写操作 | T2.2、T2.3、T3.1 | 0.8h | `service/commands.go` + I1、I2、I3、I7 |
| T3.3 | 服务层 | 读操作 | T3.1、T2.2 | 0.5h | `service/queries.go` + I9 |
| T4.1 | 接口层 | 路由、鉴权中间件、错误映射 | T3.2、T3.3 | 0.4h | `httpapi/server.go`、`middleware.go`、`errors.go` |
| T4.2 | 接口层 | Handler 与 HTTP 测试 | T4.1 | 0.4h | `httpapi/handlers.go` + I4、I5、I6、I8 |
| T5.1 | 前端 | 前端基础设施 | T0.1、T4.2 | 0.4h | API 客户端、类型、用户切换器、通用组件 |
| T5.2 | 前端 | 工单队列页 | T5.1 | 0.5h | `pages/TicketList.tsx` |
| T5.3 | 前端 | 新建工单页 | T5.1 | 0.3h | `pages/TicketNew.tsx` |
| T5.4 | 前端 | 工单详情页 | T5.1 | 0.8h | `pages/TicketDetail.tsx` |
| T6.1 | 交付 | 端到端验收 | 全部 | 0.3h | 验收记录写入 DEVLOG |
| T6.2 | 交付 | README | T6.1 | 0.3h | `README.md` |
| T6.3 | 交付 | AI 记录整理 | T6.1 | 0.2h | `docs/ai-log/` |

合计约 8.0h，处在题面建议 6～8 小时的上限。T0.2 比 v1.0 多 0.2h，用于编写跨平台 Node 辅助脚本和验证 Windows 环境。

依赖关系与关键路径：

```
T0.1 ─┬─ T0.2 ── T1.1 ── T1.2
      │            │
      └─ T2.1 ─┬─ T2.2 ─┐
               └─ T2.3 ─┤
                  T1.1 ─┴─ T3.1 ─┬─ T3.2 ─┬─ T4.1 ── T4.2 ── T5.1 ─┬─ T5.2 ─┐
                                 └─ T3.3 ─┘                        ├─ T5.3 ─┼─ T6.1 ─┬─ T6.2
                                                                   └─ T5.4 ─┘        └─ T6.3
```

单人开发时按编号串行执行，合计 8.0h。依赖图用于两件事：一是判断某个任务出问题会阻塞哪些后续任务；二是多人或多个 AI 会话并行时的分工依据，此时 T1.x 与 T2.x 可并行，T5.2、T5.3、T5.4 可并行，最长链路为 T0.1 → T0.2 → T1.1 → T3.1 → T3.2 → T4.1 → T4.2 → T5.1 → T5.4 → T6.1 → T6.2，约 5.1h。

时间不够时按这个顺序砍：先砍 T5 的视觉细节（保留加载、空状态、错误提示），再砍 I9 中搜索与分页的边界用例。T2.2、T3.2 及其测试（状态机与并发领取）不能砍，它们是题面点名的核心规则。

## 2. 层间契约

各层只通过下面的接口交互。实现任务可以增加私有函数，但不能改公开签名；确需修改时，先改本节并在 DEVLOG 记录原因。

### 2.1 分层与依赖方向

```
httpapi ──▶ service ──▶ domain
               │
               └──▶ store ──▶ domain
```

| 包 | 职责 | 禁止 |
|---|---|---|
| `domain` | 类型、错误码、输入校验、状态机、权限判定、SLA。全部是纯函数 | import 数据库、HTTP、`time.Now()` |
| `store` | SQL、事务、行锁 | 业务判断（不写 if status == ...） |
| `service` | 按 WF§5 模板编排：开事务 → 锁行 → 调 domain 判定 → 写库 → 写事件 | 直接拼 HTTP 响应 |
| `httpapi` | 路由、解析请求、鉴权中间件、domain 错误码 → HTTP 状态码 | 业务判断、直接调 store |

### 2.2 domain 包公开接口

```go
type Role string      // "agent" | "supervisor"
type Status string    // "open" | "in_progress" | "resolved" | "closed"
type Priority string  // "urgent" | "high" | "normal" | "low"
type Action string    // "view" | "claim" | "assign" | "reassign" | "resolve" | "reopen" | "close" | "comment"

type User   struct { ID int64; Name string; Role Role }
type Ticket struct {
    ID int64; Title, Description, CustomerName, CustomerContact string
    Priority Priority; Status Status; AssigneeID *int64; CreatedBy int64
    SLADueAt, CreatedAt, UpdatedAt time.Time
}

type Code string      // 取值与 WF§9 的 code 列一一对应
type Error struct { Code Code; Message string; Details map[string]string }
func (e *Error) Error() string

// 状态机（WF§3）
func CanTransition(from, to Status) bool
func TargetStatus(a Action) (Status, bool)          // resolve→resolved, reopen→in_progress, close→closed

// 权限与前置条件（WF§5 第 5～7 步），按"角色 → 归属 → 状态"顺序返回第一个错误
func Authorize(actor User, t Ticket, a Action) error
// 目标处理人检查（WF§5 第 8 步）；target 为 nil 表示用户不存在
func CheckAssignee(t Ticket, target *User) error
// 详情页按钮来源：对 claim..comment 逐个调用 Authorize，返回无错误的动作
func AllowedActions(actor User, t Ticket) []Action

// SLA（WF§8）
func SLADueAt(createdAt time.Time, p Priority) time.Time
func IsOverdue(t Ticket, now time.Time) bool

// 输入校验（WF§6 字段表）：去首尾空白后校验，失败返回 VALIDATION_ERROR，Details 为 {字段: 原因}
type CreateTicketInput struct { Title, Description, CustomerName, CustomerContact string; Priority Priority }
func (in CreateTicketInput) Normalize() (CreateTicketInput, error)
func NormalizeComment(body string) (string, error)
func NormalizeNote(note string) (string, error)
```

注意：返回值类型是 `error`，失败时返回 `*domain.Error`，成功时必须写 `return nil`，不能返回值为 nil 的 `*domain.Error` 变量（Go 的 typed nil 陷阱会让 `err != nil` 为真）。

`Authorize` 的完整判定表见 WF§17.1，T2.2 的单元测试直接按该表逐格编写。

### 2.3 store 包公开接口

```go
type Store struct { /* *pgxpool.Pool */ }
func Open(ctx context.Context, dsn string) (*Store, error)
func (s *Store) Close()

// 事务：fn 返回 error 或 panic 时回滚，否则提交
func (s *Store) InTx(ctx context.Context, fn func(q Querier) error) error

// Querier 同时由池和事务实现，读操作直接用池，写操作在 InTx 内使用
type Querier interface {
    GetUser(ctx, id int64) (*domain.User, error)                   // 不存在返回 nil, nil
    ListUsers(ctx) ([]domain.User, error)
    LockTicket(ctx, id int64) (*domain.Ticket, error)              // SELECT ... FOR UPDATE；不存在返回 nil, nil
    GetTicket(ctx, id int64) (*domain.Ticket, error)               // 不加锁
    InsertTicket(ctx, t *domain.Ticket) error                      // 写入 t 中的时间字段，回填 ID
    UpdateTicketState(ctx, id int64, s domain.Status, assignee *int64) error  // 同时设 updated_at = now()
    InsertComment(ctx, c *Comment) error                           // 回填 ID、CreatedAt
    InsertEvent(ctx, e *Event) error
    ListTickets(ctx, f TicketFilter) (items []TicketRow, total int, err error)
    ListComments(ctx, ticketID int64) ([]Comment, error)
    ListEvents(ctx, ticketID int64) ([]Event, error)
}
func (s *Store) Q() Querier   // 池上的非事务 Querier

type TicketFilter struct {
    Statuses []domain.Status; Priorities []domain.Priority
    Assignee *AssigneeFilter        // nil 不过滤；Unassigned=true 对应 assignee_id=none
    Q string; Page, PageSize int
    VisibleTo *int64                // 非 nil 时追加 agent 可见范围条件（WF§11）
}
type AssigneeFilter struct { Unassigned bool; UserID int64 }   // 二者互斥
type TicketRow struct { domain.Ticket; AssigneeName *string }
type Comment struct { ID, TicketID, AuthorID int64; Body string; CreatedAt time.Time }
type Event   struct { ID, TicketID, ActorID int64; Type string; Payload json.RawMessage; CreatedAt time.Time }
```

### 2.4 service 包公开接口

```go
// service 依赖接口而非 *store.Store，以便 I7 注入"写事件必定失败"的 Querier 包装
type DB interface {
    InTx(ctx context.Context, fn func(q store.Querier) error) error
    Q() store.Querier
}
type Service struct { db DB; now func() time.Time }
func New(db DB, now func() time.Time) *Service

// 写操作：全部按 WF§5 模板执行，返回更新后的工单
func (s *Service) CreateTicket(ctx, actor domain.User, in domain.CreateTicketInput) (*domain.Ticket, error)
func (s *Service) Claim(ctx, actor domain.User, ticketID int64) (*domain.Ticket, error)
func (s *Service) Assign(ctx, actor domain.User, ticketID, assigneeID int64) (*domain.Ticket, error)
func (s *Service) Reassign(ctx, actor domain.User, ticketID, assigneeID int64) (*domain.Ticket, error)
func (s *Service) Transition(ctx, actor domain.User, ticketID int64, a domain.Action, note string) (*domain.Ticket, error)  // a ∈ resolve/reopen/close
func (s *Service) AddComment(ctx, actor domain.User, ticketID int64, body string) (*store.Comment, error)

// 读操作
func (s *Service) Authenticate(ctx, userID int64) (*domain.User, error)           // 不存在返回 UNAUTHENTICATED
func (s *Service) ListUsers(ctx) ([]domain.User, error)
func (s *Service) ListTickets(ctx, actor domain.User, f store.TicketFilter) (*TicketPage, error)
func (s *Service) GetTicketDetail(ctx, actor domain.User, ticketID int64) (*TicketDetail, error)
```

`TicketPage`、`TicketDetail` 的 JSON 形状与 2.5 节 TypeScript 类型一一对应，`is_overdue` 和 `allowed_actions` 在 service 中用 `s.now()` 计算后填入。

创建工单时，service 取一次 `now := s.now()`，令 `CreatedAt = UpdatedAt = now`、`SLADueAt = domain.SLADueAt(now, priority)`，`InsertTicket` 原样写入这三个值。这样 `sla_due_at - created_at` 恰好等于 SLA 时长，且测试可以注入固定时钟。其余写操作的 `updated_at` 由 SQL 的 `now()` 生成。

### 2.5 HTTP 与前端类型

路由与请求体按 WF§6、WF§11，错误体按 WF§9。成功状态码：创建工单、添加评论返回 201，其余返回 200。写操作统一返回更新后的 `TicketDetail`（评论接口除外，返回新建的 `Comment`），前端拿到后直接替换页面状态，不必再请求一次。

`frontend/src/api/types.ts`：

```ts
export type Role = 'agent' | 'supervisor'
export type Status = 'open' | 'in_progress' | 'resolved' | 'closed'
export type Priority = 'urgent' | 'high' | 'normal' | 'low'
export type Action = 'claim' | 'assign' | 'reassign' | 'resolve' | 'reopen' | 'close' | 'comment'

export interface UserRef { id: number; name: string }
export interface User extends UserRef { role: Role }

export interface TicketSummary {
  id: number; title: string; customer_name: string
  priority: Priority; status: Status; assignee: UserRef | null
  sla_due_at: string; is_overdue: boolean; created_at: string   // RFC3339
}
export interface TicketPage { items: TicketSummary[]; total: number; page: number; page_size: number }

export interface Comment { id: number; author: UserRef; body: string; created_at: string }
export interface TicketEvent {
  id: number; actor: UserRef; created_at: string
  type: 'created' | 'claimed' | 'assigned' | 'reassigned' | 'status_changed' | 'commented'
  payload: Record<string, unknown>
  summary: string            // 后端渲染好的中文说明，见 WF§17.3
}
export interface TicketDetail extends TicketSummary {
  description: string; customer_contact: string; created_by: UserRef; updated_at: string
  comments: Comment[]; events: TicketEvent[]; allowed_actions: Action[]
}

export interface ApiError { code: string; message: string; details?: Record<string, string> }
```

`allowed_actions` 不含 `view`；能拿到详情即代表可查看。

## 3. 任务卡

每张卡的字段含义：**目标**是这个任务要解决的问题；**输入**是开工前必须读的 WF 章节；**产出**是提交里必须出现的文件；**验收**是可执行的检查，全部通过才算完成；**不做**是范围边界。

### 环节 0：环境

#### T0.1 仓库骨架与工具链

- **目标**：建立后续所有任务共用的目录和依赖，保证 `go build` 和 `npm run build` 在空项目上能通过。
- **输入**：WF§14 目录约定与技术选型。
- **产出**：
  - `backend/go.mod`（module `ticketdesk`，Go 1.22+），依赖 `pgx/v5`、`golang-migrate/v4`。
  - `backend/cmd/server/main.go`：启动时用 godotenv 依次尝试读取 `.env` 和 `../.env`（`go -C backend` 会把工作目录切到 `backend/`，所以根目录的 `.env` 在上一级；文件不存在时忽略），读取 `DATABASE_URL`、`ADDR`（默认 `127.0.0.1:8080`），启动一个返回 `{"ok":true}` 的 `GET /api/health`。
  - `backend/internal/{domain,store,service,httpapi}/` 空包（各含 `doc.go`）。
  - `frontend/`：Vite + React + TypeScript 模板，安装 `react-router-dom`；`vite.config.ts` 把 `/api` 代理到 `http://127.0.0.1:8080`（不写 `localhost`，原因见 0.1 节）；`package.json` 增加 `"typecheck": "tsc --noEmit"`。
  - 根目录 `package.json`：只放 npm scripts 和 `concurrently` 一个 devDependency，`engines` 写 `"node": ">=20"`。本任务先加 `setup`、`build`，其余在 T0.2 补齐。
  - 根目录 `.gitignore`（含 `.env`、`node_modules/`、`backend/bin/`、`*.exe`）、`.gitattributes`（`* text=auto eol=lf`）、`docs/ai-log/DEVLOG.md`（写入第 5 节模板）。
  - 前置软件版本写入 DEVLOG：Go、Node、Git、Docker Desktop（`go version`、`node -v`、`docker version` 的输出）。
- **验收**（PowerShell）：
  - `git init` 后执行 `git config core.autocrlf`；若为 `true`，确认 `.gitattributes` 已生效：`git ls-files --eol` 显示的仓库内换行符（`i/` 列）均为 `lf`。
  - `go -C backend build ./...; go -C backend vet ./...` 无输出（`go -C` 需 Go 1.20+，免去 `cd`）。
  - `npm run setup` 后 `npm --prefix frontend run build` 成功。
  - `go -C backend run ./cmd/server` 后，`Invoke-RestMethod http://127.0.0.1:8080/api/health` 返回 `ok : True`；`npm --prefix frontend run dev` 后浏览器访问 `http://127.0.0.1:5173/api/health` 经代理同样返回。
  - 首次运行后端时没有弹出 Windows 防火墙提示（证明只监听了 127.0.0.1）。
- **不做**：数据库连接、任何业务代码。

#### T0.2 本地运行环境

- **目标**：评审者在 Windows、macOS、Linux 上都能用同一组命令起库、迁移、启动和测试，满足"按 README 本地复现"和"推荐提供一条或少量明确的启动命令"。
- **输入**：题面"通用工程要求"；本文 0.1 节；WF§13 测试命令。
- **产出**：
  - `docker-compose.yml`：
    - `db` 服务：`postgres:16`，端口 `5433:5432`，数据卷 `pgdata`，带 `pg_isready` healthcheck。
    - 初始化脚本 `db/init.sql` 挂载到 `/docker-entrypoint-initdb.d/`，内容为 `CREATE DATABASE ticketdesk_test;`。用 `.sql` 而非 `.sh`，因为 `.sh` 文件带 CRLF 换行时容器内无法执行。
    - `gotest` 服务（`profiles: ["test"]`，平时不启动）：`golang:1.22` 镜像，挂载 `./backend`，模块缓存用命名卷，`TEST_DATABASE_URL` 指向 `db:5432`。用于在 Linux 容器中运行 `-race` 测试，原因见 T6.1。
  - `.env.example`：

    ```
    DATABASE_URL=postgres://ticketdesk:ticketdesk@127.0.0.1:5433/ticketdesk?sslmode=disable
    TEST_DATABASE_URL=postgres://ticketdesk:ticketdesk@127.0.0.1:5433/ticketdesk_test?sslmode=disable
    ADDR=127.0.0.1:8080
    ```

  - 迁移内嵌到后端：`main.go` 支持 `-migrate up|down|reset`，用 `embed` 打包 `migrations/`，评审者不必另装 migrate CLI。`reset` 等于 down 到 0 再 up，用于恢复演示数据。
  - `scripts/setup-env.mjs`：`.env` 不存在时从 `.env.example` 复制，已存在时不覆盖。替代 `cp`，在 cmd、PowerShell、bash 下都能用。
  - `scripts/dev-api.mjs`：先 `go build` 到 `backend/bin/`（按 `process.platform` 决定是否加 `.exe`），再启动该程序，收到 Ctrl+C 时结束子进程。不直接用 `go run`，因为在 Windows 上 `go run` 被中断后，编译出的临时 exe 有时仍在运行，导致下次启动报端口 8080 被占用。
  - 根目录 `package.json` 的 scripts：

    | script | 命令 | 用途 |
    |---|---|---|
    | `setup` | `node scripts/setup-env.mjs && npm install && npm --prefix frontend install && go -C backend mod download` | 首次安装 |
    | `db:up` | `docker-compose up -d --wait db` | 启动数据库并等待健康检查通过 |
    | `db:down` | `docker-compose down` | 停止数据库，保留数据 |
    | `db:reset` | `docker-compose down -v && docker-compose up -d --wait db` | 删除数据卷后重建 |
    | `db:psql` | `docker-compose exec db psql -U ticketdesk -d ticketdesk` | 进入 psql，Windows 上无需安装 psql |
    | `migrate` | `go -C backend run ./cmd/server -migrate up` | 执行迁移 |
    | `migrate:down` | `go -C backend run ./cmd/server -migrate down` | 回滚全部迁移 |
    | `seed:reset` | `go -C backend run ./cmd/server -migrate reset` | 恢复演示数据 |
    | `dev:api` | `node scripts/dev-api.mjs` | 只启动后端 |
    | `dev:web` | `npm --prefix frontend run dev` | 只启动前端 |
    | `dev` | `npm run db:up && npm run migrate && concurrently -k -n api,web "npm:dev:api" "npm:dev:web"` | 一条命令启动全部 |
    | `dev:local` | `npm run migrate && concurrently -k -n api,web "npm:dev:api" "npm:dev:web"` | 不用 Docker、使用本机 PostgreSQL 时 |

    以下 scripts 本任务先建好，引用的测试在后续任务中逐步填充：

    | script | 命令 | 用途 |
    |---|---|---|
    | `test` | `go -C backend test ./...` | 单元测试，不需要数据库（T2.1 起有内容） |
    | `test:integration` | `go -C backend test -tags integration -p 1 -count=1 ./...` | 集成测试，需先 `db:up`（T3.1 起有内容，参数说明见 T3.1） |
    | `test:race` | `docker-compose --profile test run --rm gotest go test -race -tags integration -p 1 -count=1 ./...` | 在 Linux 容器中做数据竞争检测（见 T6.1） |
    | `check` | `go -C backend build ./... && go -C backend vet ./... && node scripts/check-fmt.mjs && npm run test` | 每次提交前的检查，见第 4 节；T5.1 起追加 `&& npm --prefix frontend run typecheck` |
    | `smoke` | `node scripts/smoke.mjs` | 接口演示脚本（T4.2） |

    npm 在 Windows 上用 cmd.exe 执行 scripts，cmd 支持 `&&`，所以上表命令在三个平台上通用。但 PowerShell 5.1 不支持 `&&`，本文在 PowerShell 中直接输入的多条命令一律用 `;` 分隔。
  - 不用 Docker 时的备用方案（写入 README）：安装 PostgreSQL 16 Windows 版，用安装时创建的 postgres 账号执行 `db/init-local.sql`（创建 `ticketdesk` 用户、开发库和测试库），把 `.env` 中的端口改为 5432，然后 `npm run dev:local`。
- **验收**（PowerShell，全新克隆的仓库）：
  - `npm run setup; npm run dev` 后浏览器打开 `http://127.0.0.1:5173` 能看到页面，`/api/health` 正常。
  - 在 `npm run dev` 终端按 Ctrl+C 后，`Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue` 无输出（端口已释放），再次 `npm run dev` 正常启动。
  - `npm run db:reset; npm run dev` 可重复执行。
  - 在 Git Bash 或 WSL 中执行同样的 `npm run setup` 与 `npm run dev` 也能成功，以此确认 macOS/Linux 评审者可以复现。
- **不做**：生产部署、CI、Makefile、`.sh` 或 `.ps1` 脚本。

### 环节 1：数据库

#### T1.1 建表迁移

- **目标**：把 WF§10 的数据模型落成可正反执行的迁移，并让数据库成为业务不变量的最后一道防线。
- **输入**：WF§10 建表 SQL；WF§17.4 补充约束。
- **产出**：`backend/migrations/0001_init.up.sql`、`0001_init.down.sql`。内容为 WF§10 的 SQL 加上 WF§17.4 的补充约束。
- **验收**：
  - `npm run migrate` 后，`npm run db:psql` 进入 psql，`\d tickets` 显示全部列、`open_iff_unassigned` 等约束和索引。
  - `npm run migrate:down; npm run migrate` 往返成功。
  - SQL 文件保存为 UTF-8 无 BOM、LF 换行（VS Code 右下角可查看）。带 BOM 的文件会让第一条语句报语法错误。
  - 在 psql 中手工执行以下语句，均被数据库拒绝：插入 `status='open'` 且 `assignee_id` 非空的工单；插入 `status='in_progress'` 且 `assignee_id` 为空的工单；插入标题为全空格的工单；插入 `priority='critical'` 的工单。
- **不做**：触发器、存储过程。业务规则在 Go 中实现，数据库只做兜底。

#### T1.2 演示数据

- **目标**：启动后立即有可演示每条规则的数据，评审者不用自己造数据。
- **输入**：WF§1 演示用户；WF§10 演示工单表。
- **产出**：`0002_seed.up.sql`、`0002_seed.down.sql`（down 删除全部 seed 数据并重置序列）。
  - 4 个用户，ID 固定为 1～4。
  - 6 张工单，`created_at` 用 `now() - interval`，`sla_due_at = created_at + SLA 时长`。创建时间固定为：支付失败无法下单 3h 前，发票抬头修改 1h 前，物流长时间未更新 10h 前，账号无法登录 2h 前，退款进度查询 30h 前，修改收货地址 5 天前。
  - 每张工单的历史事件与其状态一致，例如 closed 工单有 created → claimed → status_changed(resolved) → status_changed(closed) 四条；至少 2 张工单有评论。
  - 插入后用 `setval` 把各表序列推到最大 ID 之后，避免后续插入主键冲突。
- **验收**：
  - `npm run seed:reset` 后，在 `npm run db:psql` 中执行 `SELECT status, count(*) FROM tickets GROUP BY 1;` 覆盖 4 种状态。
  - 中文标题在 psql 和前端中显示正常。若 psql 中乱码，先执行 `chcp 65001` 再进入 psql；数据库本身由 `postgres:16` 镜像默认以 UTF-8 创建，无需修改。
  - 按 WF§8 口径，恰好 2 张工单逾期（"支付失败无法下单"和"物流长时间未更新"）。
  - 每张工单的事件按时间排列后，用 `domain.CanTransition` 回放不出现非法流转（T3.2 完成后补一个测试 `TestSeedConsistency` 自动检查）。
- **不做**：大量随机数据生成。

### 环节 2：领域层

这一环节完全不碰数据库和 HTTP。题面要求测试的状态机规则在这里用纯函数实现、穷举测试，后续各层只调用不重写。

#### T2.1 类型、错误与输入校验

- **目标**：确定全系统共用的类型和错误码，让后面每一层用同一套词汇。
- **输入**：WF§6 字段校验表；WF§9 错误码表；本文 2.2 节。
- **产出**：`domain/types.go`、`domain/errors.go`、`domain/validate.go`、`domain/validate_test.go`（U5）。
  - `errors.go` 为 WF§9 每个 code 定义常量和构造函数，如 `ErrAlreadyClaimed()`；`Message` 用中文。
  - 枚举类型提供 `Valid() bool` 和 `Parse(string)`。
  - 长度按 Unicode 字符数（`utf8.RuneCountInString`）计算，不按字节。
  - 根目录 `package.json` 增加：`"test": "go -C backend test ./..."`（只跑无 `integration` 标签的单元测试，不需要数据库）。
  - 测试源码中的中文字符串直接写在 `.go` 文件里，文件保存为 UTF-8；Go 源码规定为 UTF-8，与 Windows 系统代码页无关。
- **验收**（U5，表驱动）：
  - 标题为 `"   "` → `VALIDATION_ERROR`，`Details["title"]` 非空。
  - 200 个汉字的标题通过，201 个被拒绝。
  - 多个字段同时错误时，`Details` 包含全部出错字段。
  - `priority="critical"` 被拒绝；`" high "` 被拒绝（枚举不做 trim，避免隐式纠错）。
  - 合法输入经 `Normalize` 后首尾空白被去除。
- **不做**：邮箱或电话格式校验（WF§6【决策】）。

#### T2.2 状态机与权限判定

- **目标**：把 WF§3、§4、§5 第 5～8 步变成单一判定函数。服务层、详情页按钮、测试都以它为唯一来源，保证前后端权限一致。
- **输入**：WF§3、§4、§5、§17.1；本文 2.2 节。
- **产出**：`domain/policy.go`、`domain/policy_test.go`（U1、U2）。
- **验收**：
  - U1：对 4×4 共 16 种 `(from, to)` 组合断言 `CanTransition`，恰好 4 个为 true。
  - U2：按 WF§17.1 判定表逐格断言。表中有 11 种有效的（操作者, 状态）组合（open 工单没有处理人，所以不存在"处理人 agent + open"）× 8 种动作，共 88 个用例。每个用例断言具体错误码或 nil，不能只断言"有错误"。
  - 补充用例：open 工单上的 agent 执行 claim 通过；同一 agent 对已被自己领取的工单再 claim 返回 `ALREADY_CLAIMED`。
  - `CheckAssignee`：目标为 nil、目标为 supervisor → `INVALID_ASSIGNEE`；reassign 目标等于当前处理人 → `SAME_ASSIGNEE`；assign 时工单无处理人，不触发 `SAME_ASSIGNEE`。
  - `AllowedActions`：对 seed 中 6 张工单 × 4 个用户各断言一次返回集合。
  - `go -C backend test ./internal/domain -cover` 显示覆盖率 ≥ 95%。
- **不做**：数据库读取。`Authorize` 的入参就是已加载的 `User` 和 `Ticket`。

#### T2.3 SLA 计算

- **目标**：SLA 和逾期只有一处实现，列表、详情、测试共用，避免口径分叉。
- **输入**：WF§8。
- **产出**：`domain/sla.go`、`domain/sla_test.go`（U3、U4）。
- **验收**：
  - U3：4 个优先级的 `SLADueAt(t0, p) - t0` 分别等于 2h、8h、24h、72h。
  - U4：`now = due - 1s` 不逾期；`now = due` 不逾期（WF§8 用严格大于）；`now = due + 1s` 时 open、in_progress 逾期，resolved、closed 不逾期。
  - 测试使用固定时间，不调用 `time.Now()`。
- **不做**：工作日历、暂停计时。

### 环节 3：存储与服务层

#### T3.1 连接池、事务、SQL 查询

- **目标**：提供 2.3 节的数据访问接口，所有 SQL 集中在 store 包，参数化查询，无业务判断。
- **输入**：WF§10、§11；本文 2.3 节。
- **产出**：`store/store.go`（Open、InTx、Q）、`store/tickets.go`、`store/users.go`、`store/activity.go`（评论与事件）、`store/store_test.go`。
  - `InTx` 使用 `pgx.BeginTxFunc` 或等价实现，`defer` 中处理 panic 回滚。
  - `ListTickets` 动态拼接 WHERE 时只拼占位符，值全部走参数；`q` 中的 `%`、`_`、`\` 先转义再包成 `%q%`。
  - `ListTickets` 用一条查询返回分页数据，另一条 `count(*)` 返回 total，两条使用相同 WHERE。
  - 测试辅助：`internal/testutil/db.go` 提供 `NewTestStore(t)`：
    - 从当前目录逐级向上找到含 `go.mod` 的目录，再读取其上一级的 `.env`。`go test` 运行时的工作目录是各包目录（如 `backend/internal/store`），不能用固定相对路径。
    - 读取 `TEST_DATABASE_URL`；未设置时 `t.Fatal` 并提示"先运行 npm run db:up"，不静默跳过，避免集成测试实际没跑却显示通过。
    - 执行迁移，`TRUNCATE ... RESTART IDENTITY CASCADE`，再插入 4 个固定用户。
    - testutil 依赖 store，所以 store 的测试文件用外部测试包 `package store_test`，否则会出现循环导入。
  - 所有连库的测试文件首行加 `//go:build integration`，与单元测试分开运行。
  - 根目录 `package.json` 增加：`"test:integration": "go -C backend test -tags integration -p 1 -count=1 ./..."`。`-p 1` 让各包串行执行，因为它们共用一个测试库并且会清表，并行会互相干扰；`-count=1` 禁用测试缓存，保证每次真实连库。
- **验收**：
  - `LockTicket` 阻塞测试：事务 A 锁住工单后不提交，事务 B 调 `LockTicket` 在 200ms 内不返回；A 提交后 B 返回最新数据。
  - `ListTickets`：`VisibleTo` 生效；`q="50%"` 不会匹配所有工单；`Assignee.Unassigned` 只返回无处理人工单；page_size=2 时翻页结果不重复、不遗漏。
  - `InTx` 中 fn 返回错误后，事务内的写入在库中不存在。
  - 代码中不出现 `fmt.Sprintf` 拼接用户输入到 SQL。
- **不做**：ORM、缓存、软删除。

#### T3.2 写操作

- **目标**：按 WF§5 模板实现 6 个写操作，满足"原子领取""并发只有一人成功""业务变更与历史同事务"三条核心要求。
- **输入**：WF§5、§6、§7、§17.2、§17.3；本文 2.4 节。
- **产出**：`service/service.go`、`service/commands.go`、`service/commands_test.go`（I1、I2、I3、I7）、`service/seed_test.go`（T1.2 补充的 `TestSeedConsistency`）。
  - 每个方法的结构固定为：校验输入 → `InTx { LockTicket → nil 则 404 → Authorize → (CheckAssignee) → UpdateTicketState / InsertComment → InsertEvent }` → 返回。actor 已由中间件从数据库加载，service 不再查询。
  - Assign、Reassign 在事务内用 `GetUser` 读取目标用户后再调 `CheckAssignee`。
  - 事件 payload 按 WF§6 表构造，用结构体 + `json.Marshal`，不手写 JSON 字符串。
- **验收**：
  - I1：20 个 goroutine 用 3 个 agent 轮流 claim 同一工单，起跑闸门对齐；恰好 1 个成功，19 个返回 `ALREADY_CLAIMED`；库中 `claimed` 事件恰好 1 条，assignee 等于成功者。用 `go -C backend test -tags integration -run TestConcurrentClaim -count=20 ./internal/service` 连续通过。
  - I2：agent claim 与 supervisor assign 并发，同上断言只有一方成功。
  - I3：create → claim → comment → resolve → reopen → reassign → resolve → close，每步后断言 status、assignee、事件总数和最后一条事件的 type 与 payload。
  - I7：用包装 Querier 让 `InsertEvent` 返回错误，分别对 Claim 和 AddComment 调用；断言工单状态、处理人、评论数均未变化。
  - 所有写操作在无权限、非法状态时数据库无任何变化（I3 中穿插几次失败调用并断言事件数不变）。
- **不做**：乐观锁、重试、通知。

#### T3.3 读操作

- **目标**：实现列表与详情，保证客服可见范围、逾期标记、可执行动作都由后端计算。
- **输入**：WF§4 查看权限、§11、§17.3；本文 2.4、2.5 节。
- **产出**：`service/queries.go`、`service/queries_test.go`（I9）、`service/summary.go`（事件说明渲染）。
  - `ListTickets`：actor 为 agent 时强制设置 `VisibleTo`，覆盖调用方传入的任何值。
  - `GetTicketDetail`：工单不存在 → 404；否则对任何角色返回评论、事件（按 `created_at, id` 升序）、`allowed_actions`（WF§4：客服只读可见他人工单，此时 `allowed_actions` 为空）。
  - 事件 `summary` 按 WF§17.3 模板渲染，用户名一次性批量查出，不在循环中逐条查询。
- **验收**（I9）：
  - 李雷看列表时看不到韩梅的工单，能看到全部 open 未分配工单；李雷传 `assignee_id=3` 得到空列表而非韩梅的工单。
  - 王敏看列表能看到全部；`status=open,in_progress` 多选正确；`assignee_id=none` 正确；`q` 同时匹配标题和客户名，大小写不敏感。
  - 用 `now` 注入使某张工单刚过截止时间，`is_overdue` 为 true；将其 resolve 后为 false。
  - 李雷访问韩梅的工单详情 → 成功且 `allowed_actions` 为空；访问不存在的 ID → `TICKET_NOT_FOUND`。
  - 详情中 `allowed_actions` 与 `domain.AllowedActions` 结果一致。
- **不做**：排序参数（WF§11 固定排序）、全文索引。

### 环节 4：接口层

#### T4.1 路由、鉴权中间件、错误映射

- **目标**：建立 HTTP 外壳，使 WF§2 的身份规则和 WF§9 的错误格式在所有接口上统一生效。
- **输入**：WF§2、§9、§11；本文 2.5 节。
- **产出**：`httpapi/server.go`（`NewServer(svc) http.Handler`，用 Go 1.22 `http.ServeMux` 的 `METHOD /path/{id}` 模式）、`httpapi/middleware.go`、`httpapi/errors.go`、`httpapi/json.go`。
  - 中间件：`/api/health` 以外的 `/api/*` 都要求 `X-User-ID`，调用 `svc.Authenticate`，把 `domain.User` 放入 `context`。
  - 错误映射：`*domain.Error` 按 code 映射 HTTP 状态；其他错误记日志后返回 500 `INTERNAL`，响应中不含原始错误文本。
  - JSON 解析：`DisallowUnknownFields` 关闭（以便忽略客户端多传的 `role`），请求体上限 64KB，解析失败返回 400 `VALIDATION_ERROR`。
  - 路径参数 `{id}` 非正整数 → 400；`{id}` 合法但不存在 → 404。
  - 恢复中间件：handler panic 时返回 500 并记录堆栈。
  - 请求日志：方法、路径、用户 ID、状态码、耗时。
- **验收**：
  - 映射表单测：WF§9 每个 code 映射到对应 HTTP 状态码，无遗漏（遍历 `domain` 中全部 code 常量）。
  - 无 `X-User-ID`、`X-User-ID: abc`、`X-User-ID: 999` 均返回 401，错误体格式符合 WF§9。
- **不做**：CORS（开发期走 Vite 代理，同源）、限流。

#### T4.2 Handler 与 HTTP 测试

- **目标**：把 2.4 节的 service 方法逐一暴露为 WF§6、§11 的接口，并在 HTTP 层验证错误码，证明"后端校验每项操作，不依赖前端"。
- **输入**：WF§6、§9、§11、§13；本文 2.5 节。
- **产出**：`httpapi/handlers.go`、`httpapi/dto.go`（domain → JSON 结构转换）、`httpapi/api_test.go`（I4、I5、I6、I8，用 `httptest.NewServer` + 真实测试库）。
  - 列表参数解析：`status`、`priority` 逗号分隔且逐个校验枚举；`page`、`page_size` 非法或越界返回 400。
- **验收**：
  - I4（以 supervisor 身份调用，排除权限干扰）：open 上 resolve、in_progress 上 close、closed 上 reopen → 409 `INVALID_TRANSITION`；closed 上 assign → 409 `ALREADY_CLAIMED`；closed 上 reassign、comment → 409 `INVALID_STATE`；resolved 上 reassign → 409 `INVALID_STATE`。具体码以 WF§17.1 判定表为准。
  - I5：agent 调 assign、reassign → 403 `FORBIDDEN`；agent 对他人工单 resolve、comment → 403 `NOT_ASSIGNEE`，查看他人工单详情 → 200 且 `allowed_actions` 为空；supervisor 调 claim → 403 `FORBIDDEN`。
  - I6：agent 请求体带 `{"assignee_id":2,"role":"supervisor"}` 调 assign → 403；请求头加 `X-User-Role: supervisor` 无效果。
  - I8：创建工单缺标题 → 400 且 `details.title` 存在；JSON 格式错误 → 400；不存在的工单 → 404；assign 给 supervisor 或不存在的用户 → 422 `INVALID_ASSIGNEE`；无身份 → 401。
  - 每个写接口成功时返回的 JSON 能被 2.5 节的 TypeScript 类型描述（测试中解码到对应 Go 结构体并检查关键字段非零）。
  - 附带 `scripts/smoke.mjs`（Node 20 内置 `fetch`，无需安装依赖），按 I3 流程调用一遍接口，每步打印状态和处理人，最后再演示一次权限拒绝（403）和非法流转（409）。根目录增加 `"smoke": "node scripts/smoke.mjs"`，作为 README 的接口演示脚本。不用 curl：PowerShell 5.1 中的 `curl` 是 `Invoke-WebRequest` 的别名，参数与真正的 curl 不同。
- **不做**：OpenAPI 文档生成（README 中列接口表即可）。

### 环节 5：前端

前端只负责展示和收集输入。按钮显示与否完全按 `allowed_actions`，筛选结果完全按后端返回，不在前端重算权限、逾期或可见范围。

#### T5.1 前端基础设施

- **目标**：提供各页面共用的 API 调用、当前用户状态和反馈组件，后续页面只写业务。
- **输入**：WF§2、§9、§12；本文 2.5 节。
- **产出**：
  - `src/api/types.ts`（2.5 节原样）、`src/api/client.ts`：`request<T>()` 统一加 `X-User-ID`、解析 `ApiError`、网络错误转为 `{code:'NETWORK_ERROR'}`；每个接口一个函数。
  - `src/api/errorText.ts`：WF§9 每个 code → 中文提示。
  - `src/context/CurrentUser.tsx`：启动时 `GET /api/users`，当前用户保存在 React state 与 URL 查询参数 `?as=<id>`（便于开两个窗口模拟两名客服），默认 ID 1。
  - `src/components/`：`UserSwitcher`、`Toast`、`StatusBadge`、`PriorityBadge`、`OverdueBadge`、`Loading`、`EmptyState`、`ErrorBanner`、`Pagination`。
  - `src/App.tsx`：路由与顶栏布局。
- **验收**：
  - `npm --prefix frontend run build` 与 `npm --prefix frontend run typecheck` 无错误。
  - 切换用户后，顶栏显示新用户名与角色，当前页面重新请求数据（在浏览器网络面板可见请求头 `X-User-ID` 变化）。
  - 停掉后端时页面显示网络错误提示，不白屏。
- **不做**：localStorage 持久化当前用户（用 URL 参数代替）、全局状态库。

#### T5.2 工单队列页

- **目标**：满足题面列表要求：8 个展示字段、3 种筛选、搜索、分页。
- **输入**：WF§11、§12；题面"创建与查看工单"。
- **产出**：`src/pages/TicketList.tsx`、`src/components/TicketFilters.tsx`。
  - 表格列：标题（链接到详情）、客户、优先级、状态、处理人（无则显示"未分配"）、SLA 截止时间（本地时区）、是否逾期、创建时间。
  - 筛选状态写入 URL 查询参数，刷新和返回后保持；改变筛选条件时页码回到 1。
  - 搜索框输入防抖 300ms。
  - 处理人下拉：全部 / 未分配 / 每个 agent。客服身份下仍显示该下拉，结果由后端限定。
  - 客服身份下在页面顶部提示"仅显示待领取和分配给你的工单"。
- **验收**：
  - 用主管和客服分别打开，列表内容与 T3.3 的 I9 结论一致。
  - 每个筛选条件单独与组合使用时，地址栏参数与后端请求参数一致。
  - 加载中显示加载状态；无结果显示空状态和"清除筛选"按钮；请求失败显示错误提示。
  - 逾期行有醒目标识（颜色 + "已逾期"文字，不只依赖颜色）。
- **不做**：列排序、批量操作、自动刷新。

#### T5.3 新建工单页

- **目标**：提供创建表单，前端校验规则与 WF§6 一致，并正确显示后端返回的字段错误。
- **输入**：WF§6 字段校验表、§8。
- **产出**：`src/pages/TicketNew.tsx`、`src/validation.ts`（与 WF§6 同规则）。
  - 优先级下拉显示 SLA 时长，例如"紧急（2 小时）"，默认 normal。
  - 提交中按钮禁用；成功后跳转到新工单详情并 toast "工单已创建"。
- **验收**：
  - 空标题提交时前端拦截，不发请求。
  - 后端 400 响应的 `details` 能显示在对应字段下。验证方法：临时让 `validation.ts` 返回空结果，提交空标题，确认标题下出现后端的错误信息，然后恢复。
  - 字数计数器在接近上限时提示。
- **不做**：附件、富文本、草稿保存。

#### T5.4 工单详情页

- **目标**：在一个页面完成工单的全部操作，并让评审者直观看到权限和状态机的效果。
- **输入**：WF§3、§6、§9、§12、§17.3；本文 2.5 节。
- **产出**：`src/pages/TicketDetail.tsx`，以及 `ActionPanel`、`AssigneePicker`、`CommentList`、`CommentForm`、`EventTimeline` 组件。
  - 信息区：标题、描述、客户与联系方式、优先级、状态、处理人、创建人、创建/更新时间、SLA 截止时间与剩余时间（逾期时显示"已逾期 X 小时"）。
  - 操作区：只渲染 `allowed_actions` 中的按钮。assign、reassign 弹出 agent 下拉（reassign 时排除当前处理人）；resolve、reopen、close 可填写备注；close 需二次确认。
  - 评论区：`comment` 不在 `allowed_actions` 时隐藏输入框，并显示原因（"仅处理人或主管可评论"或"工单已关闭"）。
  - 历史区：按时间升序展示 `actor.name`、类型中文名、时间、`summary`。
  - 写操作成功：用响应中的 `TicketDetail` 替换页面状态，toast 成功提示。
  - 写操作返回 409：toast 显示错误文案，并自动重新拉取详情。返回 403：显示错误文案并重新拉取。
  - 详情请求返回 403、404：显示对应的整页提示和"返回列表"链接。
- **验收**：
  - 用 seed 数据按下表逐行操作，按钮集合与结果符合预期：

    | 当前用户 | 工单 | 应显示的按钮 |
    |---|---|---|
    | 李雷 | 发票抬头修改（open） | 领取 |
    | 王敏 | 发票抬头修改（open） | 分配、评论 |
    | 王敏 | 账号无法登录（in_progress，韩梅） | 改派、解决、评论 |
    | 韩梅 | 账号无法登录 | 解决、评论 |
    | 李雷 | 账号无法登录 | 只读可见，无按钮，评论框隐藏 |
    | 李雷 | 退款进度查询（resolved，李雷） | 重开、关闭、评论 |
    | 王敏 | 修改收货地址（closed） | 无按钮，评论框隐藏 |

  - 并发演示：两个浏览器窗口分别用 `?as=2` 和 `?as=3` 打开同一 open 工单，先后点"领取"；后点的窗口显示"工单已被其他客服领取"，并刷新为 in_progress，处理人为先点者，按钮消失。
  - 每次操作后历史区立即出现新事件。
- **不做**：实时推送、乐观更新（等后端响应后再更新页面）。

### 环节 6：交付

#### T6.1 端到端验收

- **目标**：模拟评审者从零复现，确认核心路径正确，再写 README。
- **输入**：WF§16 需求追溯表；本文第 4 节完成定义。
- **产出**：DEVLOG 中的一条验收记录，列出每项检查的结果与发现的问题。
- **验收**：
  - 在另一个目录全新克隆仓库，执行 `npm run db:reset`（删除旧数据卷），再按 README 草稿的启动命令操作，记录实际耗时。
  - `npm run check`、`npm run test:integration` 全部通过。
  - `npm run test:race` 无数据竞争报告。该命令在 `gotest` 容器（Linux）中执行 `go test -race -tags integration -p 1 -count=1 ./...`。原因：Go 的 `-race` 需要 cgo 和 C 编译器，Windows 上默认没有 gcc，直接运行会报错；容器方案不需要在 Windows 上安装 MinGW。容器内由环境变量 `TEST_DATABASE_URL=...@db:5432/...` 覆盖 `.env` 中的 `127.0.0.1:5433`（系统环境变量优先，见 0.1 节）。
  - `npm run smoke` 输出完整走通。
  - 浏览器验证分别在 Edge 和 Chrome 中进行（Windows 自带 Edge，评审者常用 Chrome）。
  - 按 T5.4 的按钮表和并发演示走一遍浏览器流程。
  - WF§16 每一行都能指出对应的代码文件和测试名。
  - 发现的问题在本任务内修复，或写入 README 的"已知限制"。
- **不做**：新功能。

#### T6.2 README

- **目标**：满足题面对 README 的全部要求，让评审者 10 分钟内读懂设计并跑起来。
- **输入**：WF§15；全部【决策】条目。
- **产出**：根目录 `README.md`，章节顺序：
  1. 快速开始：前置依赖及版本（Go 1.22+、Node 20+、Docker Desktop；Windows 上 Docker Desktop 需启用 WSL2）、`npm run setup` 和 `npm run dev` 两条命令、访问地址、演示用户表。说明这些命令在 Windows（PowerShell/cmd）、macOS、Linux 上相同。另附"不使用 Docker"的步骤（T0.2）和常见问题：端口 5433 或 8080 被占用时如何修改 `.env`、终端中文乱码时执行 `chcp 65001`、Docker Desktop 未启动时 `db:up` 的报错信息、提示找不到 `docker-compose` 命令时的处理（脚本使用独立的 `docker-compose`；若本机只有 `docker compose` 子命令，把 `package.json` 中这几个脚本的 `docker-compose` 改成 `docker compose`）。
  2. 功能演示路径：按 seed 数据说明如何演示领取、并发冲突、改派、状态流转、权限拒绝。
  3. 角色模型与身份：WF§1、§2、§4 摘要。
  4. 状态机：图 + 流转表 + 非法流转的错误码。
  5. 领取并发策略：行锁方案、时序图、为什么不用条件 UPDATE、对应测试名。
  6. SLA 口径：时长表、逾期判定、时区、reopen 后的处理。
  7. 接口列表：方法、路径、权限、成功与主要错误码。
  8. 测试：如何运行，每类测试覆盖什么。
  9. 设计取舍：全部【决策】条目，每条一句理由。
  10. 已完成、未完成、已知限制。
  11. AI 使用说明：工具、协作方式，指向 `docs/ai-log/`。
- **验收**：题面"工程要求"中 README 应说明的 6 项（角色模型、状态机、领取并发策略、SLA 口径、启动方式、已完成项和未完成项）在目录中都有独立小节；README 中每条命令都在 T6.1 中实际执行过。
- **不做**：复制 WF 全文。README 写结论，细节指向 `docs/workflow.md`（WF 的副本）。

#### T6.3 AI 记录整理

- **目标**：满足题面"提交 AI 对话记录、操作日志或其他相关信息"，并让评审者看出 AI 是在规格约束下使用的。
- **输入**：开发过程中的 DEVLOG 与各会话导出。
- **产出**：`docs/ai-log/` 下：
  - `DEVLOG.md`：按任务编号排列的开发日志（第 5 节模板）。
  - `sessions/`：各 AI 会话的原始导出，文件名以任务编号开头。
  - `README.md`：说明使用了哪些 AI 工具、每个工具用于哪些任务，以及人工复核和修正 AI 输出的典型例子（至少 3 例，各附 DEVLOG 链接）。
  - 将 WF 复制为 `docs/workflow.md`、本文复制为 `docs/tasks.md`，作为规格与计划的存档。
- **验收**：导出内容中没有密钥、个人隐私信息；每个任务编号在 DEVLOG 中都有记录。
- **不做**：编辑或美化原始对话内容。

## 4. 通用完成定义

每个任务除自身验收外，提交前还要满足：

| 检查项 | 命令或方法 |
|---|---|
| 一键检查 | `npm run check`，依次执行下面四项，任一失败即退出 |
| 后端编译与静态检查 | `go -C backend build ./...`、`go -C backend vet ./...` |
| 后端格式 | `node scripts/check-fmt.mjs`：运行 `gofmt -l backend`，有输出时打印文件列表并以退出码 1 结束（`gofmt -l` 自身总是返回 0，不能直接用作检查） |
| 单元测试 | `npm run test` |
| 前端类型检查 | `npm --prefix frontend run typecheck`（T5.x 起） |
| 集成测试 | 涉及 store 及以上的任务另跑 `npm run test:integration` |
| 换行符 | `git ls-files --eol` 中没有 `i/crlf` |
| 无硬编码密钥 | 连接串只来自环境变量，`.env` 不提交 |
| 规格一致 | 新增或修改的业务规则已写回 WF，并在 DEVLOG 注明 |
| 日志 | DEVLOG 已追加本任务记录 |

## 5. DEVLOG 记录模板

```markdown
## T3.2 写操作 · 2026-11-0X HH:MM～HH:MM

- AI 工具 / 会话：<工具名>，sessions/T3.2-xxx.md
- 给 AI 的输入：任务卡 T3.2 + WF§5、§6、§7、§17.2
- AI 产出：commands.go 初稿、commands_test.go
- 人工修正：
  - AI 在 Claim 中先 UPDATE 后检查状态，改为先 LockTicket 再 Authorize（WF§5 顺序）
- 验收结果：I1 -count=20 通过；I7 通过
- 待定 / 遗留：无
```

## 6. 测试编号与任务对照

| 测试 | 内容（WF§13） | 所在任务 | 文件 |
|---|---|---|---|
| U1 | 状态流转穷举 | T2.2 | `domain/policy_test.go` |
| U2 | 权限判定表 | T2.2 | `domain/policy_test.go` |
| U3、U4 | SLA 与逾期 | T2.3 | `domain/sla_test.go` |
| U5 | 输入校验 | T2.1 | `domain/validate_test.go` |
| I1、I2 | 并发领取 | T3.2 | `service/commands_test.go` |
| I3 | 完整生命周期 | T3.2 | `service/commands_test.go` |
| I7 | 原子性 | T3.2 | `service/commands_test.go` |
| I9 | 列表与可见范围 | T3.3 | `service/queries_test.go` |
| I4、I5、I6、I8 | HTTP 层状态、权限、身份、错误码 | T4.2 | `httpapi/api_test.go` |
| — | 行锁阻塞、ILIKE 转义、分页 | T3.1 | `store/store_test.go` |
| — | seed 数据一致性 | T3.2 | `service/seed_test.go` |

