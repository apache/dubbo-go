# dubbo-go 代码审查指南（Code Review Standard & Process）

> 适用对象：`dubbo.apache.org/dubbo-go/v3`（本仓库）
> 维护者：Committer / Maintainer 团队
> 版本：v1.0 · 2026-07-30
> 配套文档： [`CONTRIBUTING.md`](./CONTRIBUTING.md) · [`security-review-2026-07-30.md`](./security-review-2026-07-30.md)

---

## 0. 为什么需要这份指南

dubbo-go 是一个**面向网络的 Go RPC 框架**，被上游业务作为基础库消费。它的代码质量直接影响成百上千个微服务的稳定性、安全性和可维护性。与业务代码不同，这里有两个特殊性：

1. **并发是默认状态**：连接池（getty）、注册中心监听、配置中心订阅、每请求 goroutine，几乎每一行都在并发环境里跑。
2. **信任边界复杂**：配置中心（ZK/Nacos/Apollo/File）、反序列化（Hessian2）、脚本路由（goja JS）、metrics/probe 端点、TLS 都是典型的攻击面。

我们在 [`security-review-2026-07-30.md`](./security-review-2026-07-30.md) 中已经发现若干真实风险（脚本路由无执行超时、metrics 端点默认无认证、TLS 未设 `MinVersion`、`exec` 拼 `$USER`、File 配置中心路径可穿越）。**自动化工具（golangci / CodeQL）无法替代人对这些语义风险的判断**，因此必须建立一套可执行的人工审查标准。

本指南目标：让任何一次 PR 的审查结果**可预期、可对齐、可复盘**，而不是依赖 reviewer 的个人经验。

---

## 1. 角色与职责

| 角色 | 职责 |
|------|------|
| **Author（作者）** | 保证 PR 自测通过、描述清晰、关联 issue、不夹带无关改动；对 reviewer 的疑问及时响应。 |
| **Reviewer（审查者）** | 至少 1 名，对改动的正确性、安全性、可维护性负责；按本指南清单逐项确认；给出明确的 `Approve` / `Request changes`。 |
| **Maintainer / Approver** | 拥有合并权限的 Committer；负责最终把关**兼容性破坏**、跨模块影响、与发行版本（release 分支）的一致性；可要求额外批准。 |
| **CI（自动门禁）** | 跑 fmt / test / lint / CodeQL / codecov，作为**合并的最低门槛**，不等于人工审查。 |

> ⚠️ 当前仓库**没有 `CODEOWNERS` 文件**。强烈建议建立（见 §6.3），否则 reviewer 分配完全靠手动，容易出现「无人审」或「不懂的人审」。

---

## 2. 审查流程

```
┌────────────┐   ┌──────────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────┐
│ 1.作者自检 │ → │ 2.提交 PR(门禁) │ → │ 3.自动 CI    │ → │ 4.人工审查   │ → │ 5.合并   │
└────────────┘   └──────────────────┘   └──────────────┘   └──────────────┘   └──────────┘
  本地 pre-push   标题/描述/issue/签名     fmt+test+lint     ≥1 Approve +     squash/merge
  跑全量清单       见 CONTRIBUTING          +CodeQL+codecov   无未解决 Blocker    回退策略
```

### 2.1 作者自检（提交前）
- 本地执行（不要等 CI 报错）：
  ```bash
  make check-fmt && make test && make lint
  go test -race ./...          # ⚠️ 当前 CI 未开启 -race，框架级改动务必本地加
  ```
- 对照 §4 清单做一遍自我审查，尤其是并发安全与安全性维度。
- 一个 PR **只解决一件事**（一个 issue）；重构与功能修复分开提。

### 2.2 提交 PR
遵循现有 [`CONTRIBUTING.md`](./CONTRIBUTING.md)：
- 标题：`<type>: 一句话摘要`，backport 加 `[3.0]` 前缀。
- 描述必须关联 issue：`Fixes: #xxxx`。
- 提交需 `Signed-off-by`（提交时 `git commit -s`）。
- PR 描述用本指南 §7 的模板，明确：改动动机、影响面、测试方式、兼容性说明。

### 2.3 自动门禁（CI）
满足以下条件**才允许进入人工审查**：
- License header 检查通过（skywalking-eyes）。
- `make check-fmt` 通过（gofmt / goimports）。
- `make test` 全绿，codecov 不下降（或下降有明确理由并经 Maintainer 同意）。
- `make lint`（golangci-lint）零告警。
- CodeQL 扫描无新增 high/critical。

> 现状提示：CI 当前**未跑 `-race`、未启用 `gosec`、未接 `govulncheck`**。这些由人工审查在 §4.2 / §6 中补足。

### 2.4 人工审查
- **最少批准数**：普通 PR ≥ 1 名 Reviewer；涉及 `protocol/`、`cluster/`、`registry/`、`config_center/`、`common/`、`global/` 等核心模块的改动需 ≥ 2 名（其中至少 1 名为对应模块 Maintainer）。
- **分配**：有 `CODEOWNERS` 时按文件路由；没有时由作者 @ 对应模块 owner。
- **SLA（建议）**：Reviewer 在 **1 个工作日内**给出首轮反馈；作者 **2 个工作日内**响应；阻塞性问题 24h 内处理。跨时区可放宽，但需在 PR 内说明。

### 2.5 评论分级与处理
所有评论必须带严重级别标记（与本指南 §5 一致）：

| 标记 | 含义 | 作者是否必须处理 |
|------|------|------------------|
| 🔴 Blocker | 合并前必须修复（安全/正确性/数据风险/破坏性变更） | 必须，否则不可合并 |
| 🟡 Suggestion | 应当修复（可维护性/性能/测试缺口） | 应修复；若坚持不改，需在 PR 内说明理由并经 Reviewer 认可 |
| 💭 Nit | 锦上添花（命名/文档/风格细节） | 可选择性处理 |

### 2.6 合并与回退
- 优先 **squash merge**，保持 `main` 线性、commit message 干净。
- 合并前确认所有 🔴 已清零、🟡 已达成一致。
- 若合并后发现引入了回归，Maintainer 有权 `revert` 并重新开 issue 跟进，不阻塞主干。

---

## 3. 严重级别定义（总览）

| 级别 | 典型场景 | 处理要求 |
|------|----------|----------|
| 🔴 **Blocker** | 安全漏洞、数据丢失/污染、竞态导致崩溃、破坏公共 API、缺关键错误分支 | 合并前必须修复 |
| 🟡 **Suggestion** | 缺输入校验、命名误导、无重要路径测试、N+1/多余分配、代码重复 | 应修复或书面豁免 |
| 💭 **Nit** | 风格不一致、次要命名、文档缺失、更优写法 | 可选 |

---

## 4. 代码审查清单（核心）

> 用法：Reviewer 逐条过，命中项在 PR 评论里引用对应条目编号（如 `§4.1.3`）。

### 4.1 正确性与并发安全（Go 框架级重点）

- **4.1.1 goroutine 泄漏**：新启的 goroutine 必须有明确的退出路径（context 取消 / channel close / `errgroup`）。尤其连接监听、watch、定时任务。
- **4.1.2 锁的正确性**：`sync.Mutex`/`RWMutex` 不要被值拷贝（`go vet -copylocks` 已覆盖，但注意结构体含锁被 `append`/`return` 拷贝）；读多写少用 `RWMutex`；避免锁内做 IO / 远程调用导致长持锁。
- **4.1.3 map / slice 并发**：非 `sync.Map` 的 map 绝不在多 goroutine 下无锁读写；注意 `getty`、注册表、缓存等共享容器。
- **4.1.4 channel 安全**：发送端可能无人接收时要有 `select { case ch <- v: default: }` 或超时；**只由发送方 close**，避免重复 close / close 已 close。
- **4.1.5 context 传递**：网络调用、RPC 调用必须透传 `ctx` 并尊重取消与超时；不要把 `context.Background()` 一路透传到底层网络栈。
- **4.1.6 错误不吞**：不要 `_ = fn()` 吞掉关键错误；返回错误时用 `fmt.Errorf("...: %w", err)` 保留链路；注意「返回非 nil error 但返回了非 nil 接口+typed nil」的经典坑。
- **4.1.7 nil 安全**：接口 nil vs 具体类型 nil 的区别；解引用前判空；`defer` 中访问可能已关闭的资源要判空。
- **4.1.8 panic 边界**：框架对外暴露的入口（协议解码、filter 链、回调）应考虑 `recover`，避免单请求 panic 拖垮整个处理 goroutine；内部逻辑不滥用 recover 掩盖 bug。
- **4.1.9 初始化顺序 / `sync.Once`**：扩展点（SPI：protocol/router/loadbalance/registry/filter）的注册与懒初始化要幂等；注意 `init()` 副作用与循环依赖。

### 4.2 安全性（信任边界）

> 结合安全评估报告中的真实发现，逐条核对。

- **4.2.1 配置中心信任边界** 🔴：来自 ZK/Nacos/Apollo/File 的内容视为**不可完全信任**。路由规则、动态配置、脚本**禁止在运行时从不可信源加载未校验内容**；配置中心侧应启用 ACL / 强认证（见报告 §二.1）。
- **4.2.2 脚本路由执行超时** 🔴：任何执行外部脚本（goja JS）的地方**必须设置中断超时**（`js_instance.go` 当前 `ClearInterrupt` 却从未设 deadline → `while(true){}` 可永久阻塞 → DoS）。强制：
  ```go
  timer := time.AfterFunc(500*time.Millisecond, func() { rt.Interrupt("timeout") })
  defer timer.Stop()
  ```
- **4.2.3 反序列化（Hessian2）** 🔴：保持 `dubbo-go-hessian2` 最新；谨慎启用 generic 服务（暴露面最大）；关注其 Security Advisory。Go 实现不会自动实例化 Java gadget 类，但仍需版本受控。
- **4.2.4 metrics / probe 端点暴露面** 🔴：默认监听 `0.0.0.0` 且无认证会泄露拓扑、成为侦察/SSRF 跳板。合并前确认：绑定 `127.0.0.1` 或加 mTLS/Bearer 鉴权或仅内网 + 网络隔离。
- **4.2.5 TLS 配置** 🟡：显式设置 `MinVersion: tls.VersionTLS12`（建议 TLS1.3）；不要出现 `InsecureSkipVerify: true`。
- **4.2.6 命令注入面** 🟡：禁止 `exec.Command("sh","-c",...)` 拼接环境变量（如 `config_center/file/impl.go` 的 `eval echo ~$USER`），改用 `os.UserHomeDir()` 等原生 API。
- **4.2.7 路径穿越** 🟡：文件型配置中心 / 静态资源路径拼接用 `filepath.Clean` 并校验结果仍在 root 内，防止 `../` 逃逸。
- **4.2.8 密钥与凭证** 🔴：禁止硬编码生产口令/Token；密钥只从配置或环境变量读取；测试占位符不得进非 `_test.go`。
- **4.2.9 依赖漏洞** 🔴：新增/升级依赖前 `govulncheck`（见 §6.2）；`go.mod` 间接依赖也要关注。

### 4.3 性能

- **4.3.1 分配与 GC 压力**：热路径（编解码、filter 链、负载均衡选择）避免不必要的堆分配；复用 buffer / `sync.Pool`；大结构体传指针而非值。
- **4.3.2 N+1 / 批量**：注册中心、配置拉取、批量调用避免逐条网络往返；支持批量与异步。
- **4.3.3 锁粒度**：高频读路径考虑 `atomic`（项目已用 `go.uber.org/atomic`）或 `RWMutex`，避免全局大锁成为瓶颈。
- **4.3.4 context 超时与重试**：网络调用必须有超时；重试用指数退避（`cenkalti/backoff` 已在依赖中），避免重试风暴。
- **4.3.5 连接/资源池**：连接池上限、空闲回收、泄漏检测；关闭时优雅 `Close()`。

### 4.4 可维护性与可读性

- **4.4.1 命名**：包级导出标识符自解释；避免缩写歧义；类型名不带 `Impl`/`Manager` 噪音（除非必要）。
- **4.4.2 函数复杂度**：golangci 已设 `gocyclo=10`；超复杂逻辑拆分，优先早返回。
- **4.4.3 魔法数字 / 常量**：超时、端口、上限等提取为具名常量。
- **4.4.4 注释**：导出的类型/函数必须有 godoc；「为什么」比「做什么」重要；复杂的并发/协议逻辑注释说明不变量。
- **4.4.5 重复代码**：跨模块重复超过 ~100 行（golangci `dupl.threshold=100`）应抽取。
- **4.4.6 日志**：统一走 `logutils.Log`（golangci `depguard` 已禁止直接用 logrus）；日志级别恰当，避免打印敏感字段（PII / 凭证 / attachment）。

### 4.5 接口与兼容性（框架级关键）

- **4.5.1 公共 API 不破坏** 🔴：导出函数/方法签名、接口、结构体字段、配置项 key 的变更视为破坏性。需要破坏时：走 `deprecated` 标记 + 至少一个大版本过渡，并在 PR 描述说明迁移路径。
- **4.5.2 接口稳定性**：被外部实现的接口（filter / router / loadbalance / registry / protocol）新增方法要评估现有实现是否编译失败。
- **4.5.3 默认值与行为**：配置默认值的改变会影响所有用户，需显式标注并在 CHANGELOG 记录。
- **4.5.4 版本分支一致性**：backport（release 分支）改动需确认 `main` 上是否已修复，避免再次引入。

### 4.6 测试与可观测性

- **4.6.1 关键路径覆盖**：新逻辑 / bug 修复必须有测试；并发修复**必须带 `-race` 测试**。
- **4.6.2 表驱动测试**：Go 惯例优先表驱动；边界值（空、超长、超时会话）要覆盖。
- **4.6.3 集成测试**：涉及注册中心 / 协议 / 配置中心的改动，尽量落到 `integrate_test.sh` 能覆盖的路径。
- **4.6.4 可观测性**：新增关键路径应有 metric / trace / log，便于线上排障；不引入新的明文敏感信息暴露。
- **4.6.5 Mock 合理性**：外部依赖（zk/nacos/网络）用 mock；不要为通过测试而放松真实不变量。

### 4.7 Go 专项

- **4.7.1 error wrapping**：用 `%w` 而非 `%v` 以便 `errors.Is/As`。
- **4.7.2 资源释放**：`defer` 关闭连接 / 文件 / 取消；`defer` 在循环里注意及时释放而非等到函数退出。
- **4.7.3 泛型 / 接口取舍**：泛型仅在有真实复用收益时使用，避免过度抽象。
- **4.7.4 不受控的 `init()`**：避免 `init()` 产生副作用或失败导致包加载崩溃。
- **4.7.5 lint 基线**：遵守 `.golangci.yml`（行宽 140、复杂度 10 等），新增代码不应产生新告警。

---

## 5. 问题分级速查（Reviewer 决策树）

```
是否会导致安全漏洞 / 数据丢失 / 生产崩溃 / 破坏公共 API？
   ├─ 是 ───────────────────────► 🔴 Blocker（必须修）
   └─ 否 ── 是否影响可维护性/性能/测试/安全加固？
                 ├─ 是 ────────► 🟡 Suggestion（应修或书面豁免）
                 └─ 否 ────────► 💭 Nit（可选）
```

评论写法示例：
```
🔴 **Security: 脚本路由无执行超时** — `cluster/router/script/instance/js_instance.go:202`
为何：恶意脚本 `while(true){}` 会永久阻塞处理 goroutine → DoS（见安全评估报告 §二.1）。
建议：加 `time.AfterFunc` 中断，见 §4.2.2。
```

---

## 6. 工具链与自动化（对齐现有，补齐缺口）

### 6.1 现有（保留）
- CI：`make check-fmt` / `make test` / `make lint` + codecov。
- `golangci-lint`：govet / ineffassign / misspell / staticcheck / unused / testifylint；`gocyclo=10`、`lll=140`、`depguard`(禁直接 logrus)。
- CodeQL（Go）安全扫描。

### 6.2 工具链落地状态（已补齐）
| 工具 | 作用 | 状态 | 落地方式 |
|------|------|------|----------|
| `go test -race` | 检测数据竞争 | ✅ 已接入 | CI 新增 `make test-race` 步骤；`Makefile` 新增 `test-race` target |
| `gosec` | 安全 lint（注入/明文/弱加密/子进程/路径穿越） | ✅ 已启用 | `.golangci.yml` 加入 `gosec`；排除高噪音 G104 与有意的弱加密(G401)，首轮聚焦真实安全问题 |
| `govulncheck` | 依赖已知漏洞 | ✅ 已接入 | CI 新增 `make vulncheck` 步骤；`Makefile` 新增 `vulncheck` target |
| `CODEOWNERS` | reviewer 路由 | ✅ 已建立 | 仓库根 `CODEOWNERS`（按模块分配，**owner 占位符需替换为真实 handle**） |
| `staticcheck` SA/ST 告警 | 已部分排除 `SA1019/ST1001` | 保留 | 仅对 legacy 路径排除，新代码不豁免 |

> ⚠️ **滚动启用提示**：`gosec` 与 `-race` 在存量代码上可能立即暴露历史问题，导致 CI 首轮变红。这是预期内的「技术债显影」，建议：
> 1. 先本地跑 `make lint` 与 `make test-race` 拿到清单；
> 2. 历史问题单独开 issue 分批清理，新代码严格零新增；
> 3. 若不想一次性阻塞全部 PR，可把 CI 中「Unit Test (race detector)」步骤临时加 `continue-on-error: true`，待清理完再移除。

### 6.3 CODEOWNERS 模板（请按实际 owner 填充）
```
# 模块负责人（示例，需替换为真实 GitHub 账号）
/cluster/                @dubbo-go/cluster-maintainers
/protocol/               @dubbo-go/protocol-maintainers
/registry/               @dubbo-go/registry-maintainers
/config_center/          @dubbo-go/config-maintainers
/common/                 @dubbo-go/core-maintainers
/global/                 @dubbo-go/core-maintainers
/metrics/                @dubbo-go/observability-maintainers
/filter/                 @dubbo-go/filter-maintainers
*                        @dubbo-go/committers
```

---

## 7. 附录：PR 描述模板

```markdown
## 动机（Why）
<!-- 关联 issue #xxxx；解决什么问题 -->

## 改动说明（What）
<!-- 一句话 + 关键文件；若是 backport 标注目标分支 -->

## 影响面 / 兼容性
<!-- 是否破坏公共 API / 默认值 / 行为；是否需要 CHANGELOG -->

## 安全性自查
<!-- 是否触及配置中心/反序列化/脚本/metrics/TLS/密钥（对照 §4.2） -->

## 测试
<!-- 新增/修改的测试；是否跑过 -race；集成测试覆盖 -->

## 自查清单
- [ ] make check-fmt / test / lint 通过
- [ ] 无新增 Blocker 级问题（已对照 §4）
- [ ] 文档/godoc 已更新（如适用）
```

---

## 8. 度量与持续改进

- **审查时效**：跟踪首响时间 / 合并周期，目标逐步收敛。
- **缺陷逃逸率**：合并后由 issue 反推「本应在审查发现却漏掉」的问题，回灌到 §4 清单。
- **清单迭代**：每季度根据安全评估、线上事故、CodeQL/govulncheck 新发现修订本指南与 §4 清单。
- **知识沉淀**：典型 bad case（如脚本路由 DoS）写成复盘，链接进对应清单条目。

---

> 本指南是**活文档**：它服务于「让 dubbo-go 的每次合并都更可信」，而非增加流程负担。当某条规则开始制造无谓摩擦时，优先修订规则，而不是让工程师绕过它。
