# AI 使用记录

## 工具

| 工具 | 用于 |
|---|---|
| Claude（Cowork） | T0.1、T0.2 初版 |
| Claude（Claude Code 桌面版） | T1.x～T6.x，以及 v1.1 规格返工 |

## 协作方式

先写规格再让 AI 实现。`docs/workflow.md` 定业务规则，`docs/tasks.md` 拆任务、给出层间接口和验收标准。
每次只给 AI 一张任务卡和它引用的规格章节；遇到规格未覆盖的情况先改规格再写代码，不让 AI 自行补充业务规则。
每个任务有可执行的验收，全部通过才进入下一个任务；结果写入 [DEVLOG.md](DEVLOG.md)。

## 人工复核并修正 AI 输出的例子

1. **依赖版本被抬高**：`go get @latest` 把 go.mod 升到 `go 1.25`，违反「Go 1.22+」，人工改为固定版本并删掉 toolchain 行。见 [DEVLOG 的 T0.1](DEVLOG.md#t01-仓库骨架与工具链--2026-11-05)。
2. **`gofmt` 与覆盖率补洞**：`errors.go` 初稿有两处格式不符；`policy.go` 首轮覆盖率 `roleAllowed` 71.4%、`SLADueAt` 75%，补了未知角色、未知优先级的测试后均为 100%。见 DEVLOG 的「环节 2 汇总验证」。
3. **`typecheck` 命令是空操作**：规格要求的 `tsc --noEmit` 在只含 references 的根 tsconfig 上不检查任何文件，故意写错类型时退出码仍为 0。改为 `tsc -b` 并用故意错误验证。见 DEVLOG 的「T5.1～T5.4 前端」。
4. **测试里引用了不存在的函数**：T3.2 测试初稿用了 `svc.Resolve` 和 `itoa`，实际应为 `Transition` 和 `strconv.FormatInt`。见 DEVLOG 的 T3.2。
5. **任务卡与权限规则矛盾**：T5.4 要求并发输家刷新后看到「处理人为先点者」，但 WF§4 当时规定客服无权查看他人工单，实际得到整页 403。AI 没有擅自改权限规则，先如实记入 DEVLOG，由人决定；人决定放宽为「客服只读可见」后，再同步修改规格、代码、测试和 README。见 DEVLOG 的 T5.x「待定 / 遗留」和「规格变更：客服可只读查看任何工单详情」。


## AI对话记录

位于同目录session文件夹中以zip存放
