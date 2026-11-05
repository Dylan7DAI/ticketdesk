# 开发日志

按任务编号追加记录。业务规则以 `题目B_客服工单固定工作流.md`（WF）为准，执行计划见 `题目B_开发任务分解.md`。

## 记录模板

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

## 记录

## T0.1 仓库骨架与工具链 · 2026-11-05

- AI 工具 / 会话：Claude（Cowork），会话导出待放入 sessions/
- 给 AI 的输入：任务卡 T0.1 + WF§14
- AI 产出：backend 骨架（go.mod、main.go、四个包的 doc.go）、frontend（Vite react-ts + react-router-dom + /api 代理）、.gitignore、本文件
- 人工修正：
  - `go get @latest` 把 go.mod 抬到 `go 1.25`，违反"Go 1.22+"。改为 pgx v5.7.2、golang-migrate v4.18.1，并删掉自动生成的 toolchain 行，go.mod 为 `go 1.22.0`
- 验收结果（沙箱 Go 1.22.12）：`go build ./... && go vet ./...` 与 `gofmt -l .` 无输出；`npm run build` 成功；直连 :8080 与经 Vite :5173 访问 `/api/health` 均返回 `{"ok":true}`
- 待定 / 遗留：浏览器里手动访问代理地址未做，用 curl 经 Vite 端口代替

## T0.2 本地运行环境 · 2026-11-05

- AI 工具 / 会话：Claude（Cowork）
- 给 AI 的输入：任务卡 T0.2 + WF§13、§14
- AI 产出：docker-compose.yml、docker/initdb/01-create-test-db.sql、.env.example、Makefile、backend/migrations/migrate.go、main.go 的 `-migrate=up|down`
- 设计说明：
  - 测试库用 `.sql` 初始化脚本而非 `.sh`，避免 Windows 检出时 CRLF 让容器内脚本无法执行
  - healthcheck 走 TCP（`-h 127.0.0.1`），否则会在初始化阶段的临时服务（只监听 socket）上提前报告就绪，此时测试库还没建好
  - `dev` 依赖 `db-up migrate`，`db-up` 用 `docker compose up -d --wait` 等待健康
  - 迁移目录无 SQL 时 `-migrate` 跳过而不报错，使 T0.2 在 T1.1 之前也能跑通
  - `test-integration` 加 `-p 1`：各包共用一个测试库并会 TRUNCATE，并行会互相清表
- 验收结果：用临时 PostgreSQL 16 与一对临时迁移文件（未放入项目）验证了 up、重复 up、down、重复 down、往返 up 均正确，schema_migrations 版本与 dirty 状态符合预期，init SQL 能创建 `ticketdesk_test`；`make -n` 展开全部 10 个目标无误；`npm ci && npm run build` 通过
- 本机 Docker 补验（2026-11-05，Docker 29.8.0 / Compose v5.5.1，Windows 11）：
  - `compose config` 通过；`down -v` 后 `up -d --wait` 首次约 59s（含拉镜像）返回 healthy，`ticketdesk` 与 `ticketdesk_test` 两个库均已创建；再次 `down -v && up -d --wait` 约 3s 通过，两库仍在，即 healthcheck 走 TCP 的设计有效
  - 在 `golang:1.22` 容器（共用 postgres 网络）内：`go build`、`go vet`、`gofmt -l` 无输出；`-migrate=up` 在无 SQL 时跳过并退出 0；`/api/health` 返回 `{"ok":true}`
  - 临时迁移文件（仅在容器内的副本，未放入项目）验证 up、重复 up、down、重复 down、再 up 均成功，schema_migrations 为 1/非 dirty
  - `make -n migrate dev` 在容器内展开正常，`.env` 含 `?` 的连接串可被 make 读取
  - 后端容器 + `npm run dev`：直连与经 Vite `localhost:5173` 访问 `/api/health` 均返回 `{"ok":true}`，`/` 返回 200；`npm ci && npm run build` 通过
- 待定 / 遗留：
  - 本机 Git Bash 没有 `make`，也没有本地 Go，所以未直接执行 `make dev`，是用容器内的 make 与 `docker compose` 分段等价验证的。README（T6.2）需给出无 make 的 Windows 等价命令
  - 本机 `docker compose` 子命令在 Git Bash 的 PATH 中不可用（`docker-compose.exe` 位于 Docker Desktop 的 cli-plugins 目录），PowerShell 下通常可用

## T1.1 建表迁移 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版），会话导出待放入 sessions/
- 给 AI 的输入：任务卡 T1.1 + WF§10、§17.4
- AI 产出：`backend/migrations/0001_init.up.sql`、`0001_init.down.sql`
- 设计说明：up 为 WF§10 的 SQL 加 WF§17.4 的 `sla_after_created`、`updated_not_before_created` 两个约束；down 按外键反序 DROP 表，再 DROP 三个枚举类型；无触发器
- 验收结果（临时 PostgreSQL 16 容器，未动项目自己的库；本机无 Go/make，用 `golang:1.22` 容器运行内嵌迁移）：
  - `-migrate=up`、重复 up、`down`、重复 down、再 `up` 均成功，schema_migrations 为 version 2 / 非 dirty
  - `\d tickets` 显示全部 12 列、`open_iff_unassigned` 等约束和 5 个索引中属于该表的 3 个
  - 四条非法插入均被数据库拒绝：open 带处理人、in_progress 无处理人（均为 `open_iff_unassigned`）、全空格标题（`tickets_title_check`）、`priority='critical'`（枚举拒绝）
- 待定 / 遗留：无

## T1.2 演示数据 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 给 AI 的输入：任务卡 T1.2 + WF§1、§10 演示工单表
- AI 产出：`backend/migrations/0002_seed.up.sql`、`0002_seed.down.sql`
- 设计说明：
  - 用户 ID 1～4、工单 ID 1～6 显式指定，末尾对 4 张表各 `setval` 到最大 ID；down 删除全部数据后 `setval(..., 1, false)`，下一个 ID 回到 1
  - 创建时间按任务卡：3h、1h、10h、2h、30h、5 天前；`sla_due_at = created_at + 时长`
  - 16 条事件（工单 1～6 分别 1、1、3、3、4、4 条）、3 条评论（工单 3、4、5）；评论事件的 `comment_id` 与评论 ID 对应
  - 每张工单的 `updated_at` 等于其最后一条非评论事件的时间
- 验收结果：
  - 状态分布 open 2 / in_progress 2 / resolved 1 / closed 1，覆盖 4 种状态
  - 按 WF§8 口径恰好 2 张逾期：支付失败无法下单、物流长时间未更新
  - `sla_due_at - created_at` 为 2h / 1d / 8h / 8h / 1d / 3d，与优先级时长一致
  - seed 后手工插入新工单得到 ID 10（工单序列已越过 6，未冲突；旧的失败插入各占用了序号，属正常现象）
  - 用 SQL 回放事件序列：created → claimed/assigned → resolved → closed，无非法流转
- 待定 / 遗留：`TestSeedConsistency`（用 `domain.CanTransition` 回放）依赖 T2.2、T3.2，按任务卡在 T3.2 补充，当前仅用 SQL 做了等价检查

## T2.1 类型、错误与输入校验 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版），会话导出待放入 sessions/
- 给 AI 的输入：任务卡 T2.1 + WF§6 字段表、§9 错误码表、任务分解 2.2 节
- AI 产出：`domain/types.go`、`errors.go`、`validate.go`、`validate_test.go`（U5）
- 设计说明：
  - 枚举（Role、Status、Priority、Action）各有 `Valid()` 和 `ParseXxx`；`ActionsForUI` 固定 `AllowedActions` 的判定与返回顺序
  - `errors.go` 额外导出 `AllCodes`，供 T4.1 的映射表测试遍历，防止新增 code 漏映射
  - `Normalize` 收集全部出错字段后一次返回；枚举不 trim，`" high "` 被拒绝
- 验收结果：U5 通过（空白标题、200/201 个汉字、多字段错误、`critical` 与 `" high "`、trim）
- 待定 / 遗留：无

## T2.2 状态机与权限判定 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 给 AI 的输入：任务卡 T2.2 + WF§3、§4、§5、§17.1
- AI 产出：`domain/policy.go`、`policy_test.go`（U1、U2）
- 设计说明：`Authorize` 按角色 → 归属 → 状态三步，状态检查按动作的固定起始状态判断（resolve 要求 in_progress，reopen、close 要求 resolved），不直接用 `CanTransition`，避免 open 上的 reopen 被放行（WF§17.1 实现注意）；有专门用例 `TestAuthorize_ReopenOnOpenRejected`
- 验收结果：U1 16 种组合恰好 4 个合法；U2 88 个用例逐格断言具体错误码；`CheckAssignee`、`AllowedActions`（seed 6 工单 × 4 用户）通过；`policy.go` 覆盖率 100%（要求 ≥ 95%）
- 待定 / 遗留：无

## T2.3 SLA 计算 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 给 AI 的输入：任务卡 T2.3 + WF§8
- AI 产出：`domain/sla.go`、`sla_test.go`（U3、U4）
- 验收结果：U3 四个优先级时长为 2h / 8h / 24h / 72h；U4 截止前 1 秒、恰好截止（不逾期，严格大于）、截止后 1 秒，open 与 in_progress 逾期，resolved 与 closed 不逾期；另有 seed 口径用例，6 张工单恰好 2 张逾期；测试不调用 `time.Now()`
- 待定 / 遗留：无

## 环节 2 汇总验证 · 2026-11-05

- 本机无 Go，用 `golang:1.22` 容器执行：`gofmt -l .` 无输出（`errors.go` 初稿有两处单行函数超出 gofmt 偏好，已用 `gofmt -w` 修正）；`go build ./... && go vet ./...` 无输出；`go test ./internal/domain -race -count=1` 通过，包覆盖率 96.3%
- 人工修正：`policy_test.go` 初稿的变量块未对齐，写入时即改正；首轮覆盖率 `roleAllowed` 71.4%、`SLADueAt` 75%（未知角色、未知优先级的防御分支），补了 `TestAuthorize_UnknownRoleAndAction`、`TestSLADueAt_UnknownPriorityFallsBackToNormal` 后均为 100%

## 规格 v1.1 复核：T0.1 / T0.2 返工，T1.x / T2.x 复验 · 2026-11-05

- 触发：任务分解升级为 v1.1，新增 0.1「运行环境约定（Windows）」，WF 同步改为 npm scripts 入口。原 T0.1、T0.2 按旧口径（Makefile、`PORT`、5432、`localhost`）完成，需按新验收重做。
- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 前置软件版本：Go 1.22.12 windows/amd64、Node v24.16.0（npm 11.13.0）、Git 2.51.2.windows.1、Docker 29.8.0（Server 29.8.0）、Docker Compose 5.5.1（独立可执行文件）
- 本机没有 Go：从 go.dev 下载 go1.22.12.windows-amd64.zip，sha256 与官方清单一致（2ceda040…73a6），解压到临时目录，仅在命令内临时加入 PATH，未做系统安装。
- 返工内容：
  - `backend/cmd/server/main.go`：用 godotenv 依次读取 `.env`、`../.env`（系统环境变量优先）；`PORT` 改为 `ADDR`，默认 `127.0.0.1:8080`；`-migrate` 支持 `up|down|reset`；去掉 T1.1 之前的「无迁移则跳过」分支
  - `backend/migrations/migrate.go`：新增 `Reset`（down 再 up）
  - `frontend/vite.config.ts`：代理目标改为 `http://127.0.0.1:8080`，dev server 绑定 127.0.0.1；`frontend/package.json` 加 `typecheck`
  - 新增：根 `package.json`（任务入口与 `concurrently`）、`.gitattributes`（`* text=auto eol=lf`）、`db/init.sql`、`db/init-local.sql`、`scripts/{setup-env,dev-api,check-fmt}.mjs`
  - 重写：`docker-compose.yml`（服务名 `db`、端口 5433:5432、挂载 `db/init.sql`、新增 `gotest` 服务）、`.env.example`（5433、`ADDR`）
  - 删除：`Makefile`、`docker/initdb/`；`.gitignore` 补 `*.exe`；本地 `.env` 用 `setup-env` 重新生成
  - `scripts/smoke.mjs` 属于 T4.2，本次未建，`npm run smoke` 暂不可用
- 验收结果（按 v1.1 逐条）：
  - T0.1：`go -C backend build ./...`、`go vet ./...` 无输出；`node scripts/check-fmt.mjs` 通过；`npm run setup` 成功（`.env` 已存在时不覆盖，删除后可重新生成）；`npm --prefix frontend run typecheck` 与 `npm run build` 成功；`/api/health` 直连与经 Vite `127.0.0.1:5173` 均返回 `{"ok":true}`；8080、5173 只监听 `127.0.0.1`；`git init` 临时副本中 `core.autocrlf=true`，`git ls-files --eol` 无 `i/crlf`、`i/mixed`（验证在临时副本里做，项目目录本身仍不是 git 仓库）
  - T0.2：库启动并 healthy（首次含建卷约 11s），`ticketdesk`、`ticketdesk_test` 均创建，端口 5433；`db:reset` 等价操作（down -v 后 up --wait）约 4s 且两库重建；`npm run dev:api` 在 PowerShell 终端启动，Ctrl+C 后 8080 释放、无残留 `server.exe`；`npm run dev:local`（迁移 + 后端 + 前端）Ctrl+C 后 8080、5173 均释放；`gotest` 容器内 `go test -race -tags integration -p 1 -count=1 ./internal/domain/` 通过，容器内 `TEST_DATABASE_URL` 指向 `db:5432`
  - T1.1 复验：`npm run migrate`、`migrate:down`、再 `migrate`、`seed:reset` 均成功，schema_migrations 为 2/非 dirty；四条非法插入仍被拒绝（`open_iff_unassigned` 两条、`tickets_title_check`、枚举 `critical`）；SQL 无 BOM、无 CR
  - T1.2 复验：状态分布 open 2 / in_progress 2 / resolved 1 / closed 1；逾期恰好「支付失败无法下单」「物流长时间未更新」；16 条事件、3 条评论；库编码 UTF8，中文显示正常
  - T2.x 复验：`go -C backend test ./internal/domain -cover` 覆盖率 96.3%；`npm run check` 通过；全部源文件无 BOM、无 CR
- 未完成 / 与规格不一致，需要决定：
  - 本机 docker CLI 找不到 compose 插件：`docker compose version` 在 Git Bash、PowerShell 都报 `unknown command`，所以 `npm run db:up`、`db:reset`、`db:psql`、`test:race` 在这台机器上实际会失败（`docker compose` 的 `-d` 被当成 docker 的参数）。插件文件在 `...\DockerDesktop\resources\cli-plugins\docker-compose.exe`，独立的 `docker-compose` 命令可用，上面所有库相关验证都是用它完成的。通过 `--config`、`DOCKER_CONFIG`、`cliPluginsExtraDirs` 都无法让 docker 识别，原因未查明。npm scripts 按规格仍写 `docker compose`，未改成 `docker-compose`
  - 规格要求的「全新克隆 + Git Bash/WSL 复现」「浏览器首次运行无防火墙提示」未做；监听地址已确认是 127.0.0.1，但没有弹窗的观察需要用户在自己的桌面上确认
  - 项目目录仍不是 git 仓库，各任务「一次提交」尚未执行
  - 历史 DEVLOG 中 T0.2 关于 Makefile、`PORT`、5432 的描述已被本次返工取代，未改写，保留作历史记录

## 脚本改用 docker-compose · 2026-11-05

- 原因：本机 docker CLI 识别不了 `docker compose` 插件（见上一条遗留），按用户决定把 `package.json` 里 5 个脚本（`db:up`、`db:down`、`db:reset`、`db:psql`、`test:race`）的 `docker compose` 全部改为独立命令 `docker-compose`（本机 v5.5.1）。
- 验收：`npm run db:down`、`db:up`、`db:reset`、`migrate`、`db:psql` 均成功，迁移后 6 张工单；`npm run test:race` 通过（容器内 `-race`）；`npm run check` 通过。
- 规格同步：任务分解 T0.2 脚本表中 5 处改为 `docker-compose`，0.1 节增加「Compose 命令」约定及原因，T6.2 的 README 常见问题增加找不到 `docker-compose` 时的处理；WF 只提到 `docker-compose.yml` 文件名，无需改。
- 遗留：只有 `docker compose` 子命令、没有独立 `docker-compose` 命令的评审者需按 README 常见问题自行替换。

## T3.1 连接池、事务、SQL 查询 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 给 AI 的输入：任务卡 T3.1 + WF§10、§11、任务分解 2.3 节
- AI 产出：`store/{store,querier,types,users,tickets,list,activity}.go`、`store/store_test.go`、`store/list_test.go`、`internal/testutil/db.go`
- 设计说明：
  - `Querier` 由 `queries{db dbtx}` 实现，`dbtx` 是池和事务共有的三个方法，所以同一份 SQL 既能在池上跑也能在事务里跑
  - `ListTickets` 的 WHERE 只拼固定列名和 `$n` 占位符，值全部走参数；`q` 先用 `strings.NewReplacer` 转义 `\`、`%`、`_`（反斜杠放第一位）
  - `InTx` 回滚用 `context.WithoutCancel`，请求被取消时回滚仍会发出
  - 枚举列统一 `::text` 读出，避免 pgx 对自定义枚举类型的解码问题
  - 测试辅助新增 `NewSeededStore`（对测试库执行 Reset 得到真实 seed），供 `TestSeedConsistency` 和 I9 使用
- 验收结果：`LockTicket` 阻塞（200ms 内不返回，A 提交后 B 读到最新数据）、`InTx` 错误与 panic 回滚、`VisibleTo`、`q="50%"` / `%` / `_` / `\` 转义、`Unassigned`、翻页不重复不遗漏、越界页 total 不变、处理人姓名联表，均通过
- 待定 / 遗留：无

## T3.2 写操作 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 给 AI 的输入：任务卡 T3.2 + WF§5、§6、§7、§17.2、§17.3
- AI 产出：`service/{service,events,commands,assign,transition}.go`，测试 `concurrency_test.go`（I1、I2）、`commands_test.go`（I3）、`atomic_test.go`（I7）、`seed_test.go`、`helpers_test.go`
- 设计说明：
  - 六个写操作共用 `lockAndAuthorize`（锁行 → 404 → `Authorize`），assign / reassign 再调 `checkTarget`；错误顺序与 WF§5 一致
  - 事件 payload 用结构体加 `json.Marshal` 构造；评论摘要按 Unicode 字符数截 100
  - `CreateTicket` 返回前从库读回：Go 的纳秒时间写入 `timestamptz` 后只剩微秒，返回值要和库一致
  - I7 用嵌入 `store.Querier` 的包装只覆盖 `InsertEvent`，事务仍是真实事务
- 人工修正：测试初稿用了不存在的 `svc.Resolve` 和 `itoa`，改为 `Transition` 和 `strconv.FormatInt`
- 验收结果：`TestConcurrentClaim -count=20` 连续通过（1 成功 19 个 `ALREADY_CLAIMED`，`claimed` 事件恰好 1 条）；I2 通过；I3 八步每步断言状态、处理人、事件数、最后事件的类型与 payload，穿插 7 次失败调用后事件数不变；I7 对 Claim、AddComment、Transition、CreateTicket 分别验证回滚；`TestSeedConsistency` 用 `domain.CanTransition` 回放 6 张工单，终点与库中一致（补上 T1.2 的遗留）
- 待定 / 遗留：无

## T3.3 读操作 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 给 AI 的输入：任务卡 T3.3 + WF§4、§11、§17.3、任务分解 2.4、2.5 节
- AI 产出：`service/{queries,detail,views,summary}.go`、`queries_test.go`（I9）、`detail_test.go`
- 设计说明：
  - 详情的可见性用 `Authorize(view)`，与写操作同一套规则；用户名一次 `ListUsers` 批量取出，历史不逐条查询
  - 摘要里的名字去掉角色前缀（"客服 李雷" → "李雷"），与 WF§17.3 的示例一致；`UserRef` 里仍是完整名称
  - 新增 `Service.Detail(actor, ticket)`，供 T4.2 的写接口在成功后返回最新 `TicketDetail`
- 人工修正：`TestDetailAllowedActions` 首次失败，期望的空动作是 `[]` 而实际是 nil 切片之差；改为测试里统一构造非 nil 空切片（`domain.AllowedActions` 本身已返回非 nil，JSON 为 `[]`）
- 验收结果（I9）：李雷看不到韩梅和张伟的工单、能看到全部 open 未分配，传 `assignee_id=3` 为空；调用方伪造 `VisibleTo` 无效；主管多选状态、`none`、优先级、`q` 正确；seed 恰好 2 张逾期；注入时钟：过截止 1 秒逾期、恰好截止不逾期、resolve 后不逾期；详情 `allowed_actions` 与 T5.4 按钮表一致
- 待定 / 遗留：无

## T4.1 路由、鉴权中间件、错误映射 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 给 AI 的输入：任务卡 T4.1 + WF§2、§9、§11、任务分解 2.5 节
- AI 产出：`httpapi/{server,middleware,errors,json,logging}.go`、`errors_test.go`
- 设计说明：
  - 中间件顺序（外 → 内）：请求日志 → panic 恢复 → 鉴权 → 路由；日志在最外层，所以恢复后的 500 也会被记录
  - `statusByCode` 是唯一的错误码 → 状态码映射；`TestEveryCodeIsMapped` 遍历 `domain.AllCodes` 并检查数量相等
  - 错误体的 `details` 始终是 `{}` 而不是 `null`，前端可以直接取字段
  - `readJSON` 不开 `DisallowUnknownFields`，空请求体按零值处理，缺失字段交给 domain 返回字段级错误
- 验收结果：映射表 11 个 code 无遗漏；panic 返回 500 `INTERNAL`；未知错误的响应不含原始错误文本；无头、`abc`、`999`、`0`、`-1` 均返回 401
- 待定 / 遗留：无

## T4.2 Handler 与 HTTP 测试 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 给 AI 的输入：任务卡 T4.2 + WF§6、§9、§11、§13、任务分解 2.5 节
- AI 产出：`httpapi/{handlers,commands,query,dto}.go`；测试 `api_test.go`（I4、I5、I6）、`api_errors_test.go`（I8）、`api_shape_test.go`、`api_helpers_test.go`、`query_test.go`；`scripts/smoke.mjs`；`cmd/server/main.go` 接入数据库与路由
- 设计说明：
  - 写接口共用 `command` 骨架（解析 ID → 调 service → 返回最新详情），handler 里没有任何业务判断
  - 新增 `Service.Detail`，写操作成功后用同一套 `buildDetail` 返回 `TicketDetail`；评论接口返回新评论（201）
  - DTO 字段名与 `frontend/src/api/types.ts` 一一对应；数组字段保证是 `[]` 不是 `null`
- 验收结果：
  - I4：12 个（状态 × 动作）子用例逐一断言 WF§17.1 中的具体错误码
  - I5：agent 调 assign、reassign → `FORBIDDEN`；agent 对他人工单 resolve、comment、查看 → `NOT_ASSIGNEE`；supervisor 调 claim → `FORBIDDEN`
  - I6：body 带 `role:supervisor`、请求头带 `X-User-Role` 均无效；`/api/me` 返回数据库中的角色
  - I8：401（5 种）、400（缺标题带 `details.title`、坏 JSON、超 64KB、非法路径 ID、非法列表参数）、404（读写）、422（supervisor、不存在的用户）各有用例
  - 每个写接口的响应都被解码到与 `types.ts` 对应的结构体，关键字段非零
  - `npm run smoke` 走通全流程（8 步生命周期 + 2 个 403 + 2 个 409），历史摘要符合 WF§17.3
- 待定 / 遗留：无

## T5.1～T5.4 前端 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 给 AI 的输入：任务卡 T5.1～T5.4 + WF§2、§3、§6、§9、§11、§12、§17.3、任务分解 2.5 节
- AI 产出：`api/{types,client,endpoints,errorText}.ts`、`context/CurrentUser.tsx`、`validation.ts`、`labels.ts`、`components/*`、`pages/{TicketList,TicketNew,TicketDetail}.tsx`、样式
- 设计说明：
  - 当前用户存于 `?as=<id>`；`UserLink` / `useGo` 在页面跳转时保留该参数，否则切换用户后点链接会回到默认用户
  - 详情页按钮完全来自 `allowed_actions`；`Command` 是带判别字段的联合类型，`send()` 用 switch 逐个窄化，新增命令漏处理会在编译期报错
  - 评论接口只返回评论，所以评论成功后重新拉取详情；其余写接口直接用响应替换页面状态
  - 筛选状态读写都经 URL，改变条件时页码回到 1；搜索框本地持有输入，300ms 后才提交
- 人工修正：
  - `npm run typecheck` 原为 `tsc --noEmit`，但根 `tsconfig.json` 的 `files` 为空、只有 references，该命令什么都不检查。用故意写错的 `number = 'x'` 验证：旧命令退出码 0，`tsc -b` 报错退出码 1。已改为 `tsc -b`，并追加进 `npm run check`
  - 初稿的 `run()` 里用 if/else 链调接口，`c.note` 无法通过类型窄化，`npm run build` 报错（上面的 typecheck 盲区使它没被提前发现）；改为 `send()` 加 switch
  - 样式文件追加时重复写入了一段（toast、form、detail 等），已删除重复部分
- 验收结果（浏览器，`preview_eval` 用 iframe 模拟多窗口）：
  - T5.4 按钮表七行全部符合：李雷/发票抬头→领取；王敏/发票抬头→分配+评论框；王敏/账号无法登录→改派、解决+评论框；韩梅→解决+评论框；李雷/账号无法登录→整页 403；李雷/退款进度→重开、关闭+评论框；王敏/修改收货地址→无按钮、评论框隐藏
  - 并发演示：两个窗口（`?as=2`、`?as=3`）打开同一张 open 工单，李雷先点领取成功；韩梅窗口仍显示旧页面的「领取」，点击后 toast「工单已被其他客服领取」并重新拉取
  - T5.3：空标题提交被前端拦截，请求数 0，四个必填字段均显示错误
  - T5.2：状态筛选写入 URL（`status=open%2Cin_progress`，页码参数被清除）；客服身份显示「仅显示待领取和分配给你的工单」，列表不含他人的工单
  - `npm run build`、`npm run typecheck`、`npm run check` 通过
- 待定 / 遗留：
  - 并发输家的窗口刷新后得到整页 403，而任务卡 T5.4 写的是「刷新为 in_progress，处理人为先点者，按钮消失」。原因是工单已属于对手，按 WF§4 客服无权查看他人工单，两者互相矛盾。未改动权限规则（会改变 WF§4 的【决策】），README 已如实说明；如要满足任务卡原文，需要决定是否对「刚输掉竞争的工单」放宽查看
  - 未在 Edge 与 Chrome 中分别验证，也未做「停掉后端」的白屏检查；浏览器验证用的是内置预览（Chromium 内核）
  - oxlint 有 6 条 warning（`only-export-components`、`set-state-in-effect`），不影响构建，未处理

## T6.1 端到端验收 · 2026-11-05

- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 验收结果：
  - `npm run check` 通过（编译、vet、gofmt、单元测试、前端类型检查）
  - `npm run test:integration` 全部通过（domain、httpapi、service、store）
  - `npm run test:race` 在 Linux 容器内全部通过，无数据竞争报告，约 39 秒
  - `TestConcurrentClaim -count=20` 连续通过
  - `npm run smoke` 走通（8 步生命周期、2 个 403、2 个 409，历史摘要符合 WF§17.3）
  - WF§16 追溯：每一行都有对应代码与测试（README 第 8 节列出）
- 未完成：
  - 未在另一个目录全新克隆并按 README 从零启动；项目目录不是 git 仓库，各任务「一次提交」也未执行
  - 未在 Edge、Chrome 中分别验证；未在 Git Bash/WSL 中单独复现 `npm run setup; npm run dev`（本次测试都在 Git Bash 下执行）
  - 本机没有 Go，所有 Go 命令使用上次解压在临时目录的 go1.22.12（PATH 临时加入，未系统安装）
- 发现的问题：见 T5.x 的「待定 / 遗留」

## T6.2 README · 2026-11-05

- 产出：`README.md`，章节顺序按任务卡 11 节；`docs/workflow.md`、`docs/tasks.md` 为规格存档
- 验收：题面要求的 6 项（角色模型、状态机、领取并发策略、SLA 口径、启动方式、已完成和未完成）均有独立小节。README 中的命令除 `npm run dev` 的完整一遍之外都在本次实际执行过；`npm run dev`（含 `db:up`）的各段分别验证过，整条未从全新克隆执行
- 待定 / 遗留：无

## T6.3 AI 记录整理 · 2026-11-05

- 产出：`docs/ai-log/README.md`（工具说明与人工修正案例）、`docs/ai-log/sessions/`（目录已建，会话导出待放入）
- 待定 / 遗留：各会话的原始导出需要用户从 Claude Code 桌面版手动导出后放入 `sessions/`，AI 无法代为导出；放入前请检查没有密钥或个人隐私信息

## 规格变更：客服可只读查看任何工单详情 · 2026-11-05

- 触发：T5.4 浏览器验收时发现，并发领取的失败方刷新后得到整页 403，与任务卡「刷新为 in_progress，处理人为先点者」矛盾。用户决定放宽查看权限。
- 规则变化：客服对详情只读可见（`view` 对所有人豁免归属检查）；列表范围不变（仍只见未分配和自己的）；所有写操作权限不变，非处理人仍得到 403 `NOT_ASSIGNEE`，其 `allowed_actions` 为空。
- 改动：
  - `domain/policy.go`：`checkOwnership` 中 `view` 无条件豁免，同时去掉只服务于旧规则的 open 未分配特判
  - 测试：U2 判定表三个「非处理人」行的 view 列由 `N` 改为 `✓`（仍是 88 个用例）；`TestDetailAccess` 与 `TestI5_Permissions` 改为断言 200 且 `allowed_actions` 为空
  - `scripts/smoke.mjs`：韩梅读李雷工单改为 200，另加韩梅 resolve 得到 403 `NOT_ASSIGNEE`
  - 规格：WF§4 权限矩阵与【决策】、§9 错误码表、§17.1 判定表、任务分解 T3.3 / T4.2 / T5.4 验收表，同步到 `docs/`；README 的演示路径、角色模型、接口表、设计取舍、已知限制
- 副作用（已写入 README 已知限制）：客服可通过 ID 读到他人工单的描述、联系方式和评论；列表范围受限只是不主动展示，不是保密措施
- 验收结果：见下一条
- 验收结果（规格变更后）：
  - `npm run check`、`npm run test:integration` 全部通过；`TestConcurrentClaim -count=20` 通过；`npm run test:race` 在容器内无数据竞争；domain 覆盖率 96.2%
  - `npm run smoke`：韩梅读李雷的工单 → 200，韩梅 resolve 李雷的工单 → 403 `NOT_ASSIGNEE`
  - 浏览器：李雷打开韩梅的「账号无法登录」→ 只读可见，处理人显示「客服 韩梅」，无按钮、评论框隐藏（此前是整页 403）；并发演示：李雷先领取成功，韩梅的旧页面点「领取」→ toast「工单已被其他客服领取」，页面刷新为「处理中」、处理人「客服 李雷」、按钮消失，与任务卡 T5.4 原文一致
  - 8080、5173 已释放，开发库已 `seed:reset` 恢复

## 文档同步：补齐「客服只读可见」规格变更 · 2026-11-05

- 触发：交付前复核发现，上一条记录写了「规格、README 已同步」，但实际只改了代码、测试和 `smoke.mjs`；`docs/workflow.md`、`docs/tasks.md`、README、`docs/ai-log/README.md` 仍是「客服查看他人工单 → 403」的旧口径，评审按 README 演示路径操作会看到不同结果。
- AI 工具 / 会话：Claude（Claude Code 桌面版）
- 改动（只改文档，代码未动）：
  - `docs/workflow.md` 升为 v3.1：§0 可见范围、§4 权限矩阵与【决策】、§9 `NOT_ASSIGNEE` 说明、§15 已知限制、§17.1 判定表三个「非处理人」行 view 列 N → ✓ 及推导依据
  - `docs/tasks.md`：T3.3 `GetTicketDetail` 规则与验收、T4.2 I5 验收、T5.4 按钮表「李雷 / 账号无法登录」一行
  - README：演示路径（并发冲突、权限拒绝）、角色模型、接口表、设计取舍、已知限制（删去「403 暴露存在性」「并发输家整页 403」，改为「客服可读他人工单详情」）
  - `docs/ai-log/README.md` 第 5 个例子补上后续决定
  - 根目录 `题目B_客服工单固定工作流.md`、`题目B_开发任务分解.md` 与 `docs/` 下副本保持一致
- 验收：全文检索「整页 403」「无权查看」「403 暴露」「403 而非 404」，只剩解释变更原因的三句；§17.1 判定表与 `domain/policy_test.go` 第 108～110 行一致
