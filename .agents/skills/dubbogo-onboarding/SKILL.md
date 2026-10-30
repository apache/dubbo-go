---
name: dubbogo-onboarding
description: "Dubbo-go 新手/初学者上手助手。帮助刚接触本仓库的用户快速理解项目结构、构建测试方式、扩展(SPI)机制，并指导完成『添加 filter / protocol / registry / 跑通测试』等常见任务。当用户提到 \"dubbogo 怎么上手\"、\"dubbo-go 入门\"、\"新人 onboarding\"、\"项目结构\"、\"怎么跑起来\"、\"怎么测试\"、\"怎么加一个 filter/protocol\"、\"extension 机制\"、\"explore dubbo-go\"、\"dubbogo architecture\" 时使用。触发前先确认用户是否在本仓库根目录。"
metadata:
  display_name: "dubbogo-onboarding"
  display_name_en: "dubbogo-onboarding"
  visibility: "public"
---

# Dubbo-go 上手 Skill（面向初学者 + AI Coding Agent）

> 目标：让第一次接触 apache/dubbo-go 的人，能在 AI 编码助手的引导下，5 分钟内看懂项目骨架，30 分钟内跑通构建与单测，并知道「加一个新能力」该动哪些文件。

---

## 何时触发（Trigger）

- 用户刚 clone 本仓库，想了解「这是什么、怎么组织、怎么跑」
- 用户问：怎么 build / test / 加一个 filter / 加一个 protocol / 看懂 extension 机制
- 关键词：dubbogo 入门、onboarding、上手、项目结构、架构、怎么跑、怎么测、添加filter、添加protocol、SPI、扩展机制

**前置确认**：先确认当前工作目录是 dubbo-go 仓库根（`go.mod` 的 module 为 `dubbo.apache.org/dubbo-go/v3`）。若不在，提示用户切到仓库根再继续。

---

## 项目地图（Project Map）

顶层目录 → 职责（让 agent 先建立全局认知）：

| 目录 | 职责 |
|------|------|
| `dubbo.go` / `options.go` / `instance_options_init.go` / `loader.go` | 应用入口、配置项、配置加载 |
| `client/` | 客户端调用（reference 配置与调用链路） |
| `cluster/` | 集群治理：路由(router)、负载均衡(loadbalance)、重试、容错 |
| `common/` | 公共能力与**扩展注册表 `common/extension`**（SPI 核心） |
| `config_center/` | 配置中心（nacos/zk/apollo…）适配 |
| `filter/` | 过滤器链（tracing/metrics/accesslog/限流…） |
| `global/` | 全局配置结构定义 |
| `logger/` | 日志抽象与实现 |
| `metadata/` | 元数据中心 |
| `metrics/` / `otel/` | 可观测性（Prometheus / OpenTelemetry） |
| `protocol/` | 协议层（dubbo/triple/grpc…） |
| `proxy/` | 代理生成（invoker 包装） |
| `registry/` | 注册中心（zookeeper/nacos/etcd…） |
| `remoting/` | 底层通信（getty/exchange） |
| `graceful_shutdown/` | 优雅下线 |
| `imports/` | **聚合所有扩展的 `init()` 注册**（让扩展在二进制里生效） |
| `doc/` | 架构图（PNG）与说明 |
| `tools/` | 仓库内部工具（如 variadic rpc 检查） |

---

## 环境与构建（Environment & Build）

- **Go 版本**：先读取 `go.mod`；当前要求 `go 1.25.0`。按当前 shell 的语法设置环境变量 `GOTOOLCHAIN=go1.25.0+auto` 后再构建。Makefile 通过 `GO` 变量调用 Go，不会自动设置该环境变量。
- **常用命令**（来自 `Makefile`）：
  - 跑单测：`make test`（根模块生成覆盖率文件，并测试 `tools/dubbogo-cli` 独立模块）
  - 全量竞态测试：`make test-race`（含 Makefile 中列明的存量测试排除项）
  - 静态检查：`make lint`（使用 Makefile 固定版本的 golangci-lint）
  - 格式化及语法更新：`make fmt`；CI 用 `make check-fmt` 校验 gofmt 结果
  - 集成测试：`./integrate_test.sh`；先读 `.github/workflows/github-actions.yml` 和脚本，确认参数及外部依赖
- **按修改范围选择验证**：Go 代码先运行受影响包的测试和 `go test -race ./<changed-package>`，再按风险扩大检查。文档改动核对路径、命令和接口说明；报告实际运行的检查及未验证项。

---

## 核心入口（Core Entry Points）— 给 agent 的阅读顺序

1. `dubbo.go`：应用/框架启动与配置装配入口
2. `options.go` + `instance_options_init.go`：可配置项与默认值
3. `loader.go`：配置加载逻辑
4. `common/extension/`：全局扩展注册表（理解「实现如何被找到」）
5. `imports/`：所有扩展的 init 注册汇总（理解「实现如何被启用」）

---

## 扩展机制（Extension / SPI）— 最关键的概念

dubbo-go 用**全局注册表 + `init()` 自注册**做依赖注入，而不是在调用点直接 new：

1. 各模块在 `init()` 里调用 `common/extension.SetXxx(name, constructor)` 注册实现。
2. 运行时通过 `common/extension.GetXxx(name)` 按名取出。
3. `imports/` 包空导入（blank import）这些模块，确保它们的 `init()` 被执行——**不 import，扩展就不生效**。

**新增一个扩展的标准三步**（agent 指导用户时照此）：
1. 实现对应接口（如 `filter.Filter`、`protocol/base.Protocol`、`registry.Registry`）。
2. 在自己的包里写 `func init() { extension.SetXxx("my-name", newMyImpl) }`。
3. 在 `imports/` 下增加对该包的引用（blank import），否则注册不会发生。

> ⚠️ 常见坑：只写了实现和 `init()` 却在 `imports/` 里忘了引用 → 运行时不报错但扩展「不存在」。

---

## 常见任务（Common Tasks）— 给 agent 的执行步骤

### 任务 A：加一个 Filter
1. `Grep` 现有 filter（如 `filter/`（目录下 `accesslog`、`tps` 等）理解接口签名 `filter.Filter`）。
2. 在 `filter/` 下新建子包，实现 `filter.Filter` 的 `Invoke`/`OnResponse`。
3. `init()` 中 `extension.SetFilter("my-filter", newMyFilter)`。
4. 在 `imports/` 增加 blank import。
5. 先对新增 filter 包运行 `go test -race ./filter/<new-filter>`，再按影响范围决定是否运行 `make test-race`。

### 任务 B：加一个 Protocol
1. 先读 `protocol/base/base_protocol.go` 中的 `base.Protocol`，参考 `protocol/dubbo/` 或 `protocol/triple/`。
2. 对照 `common/extension/protocol.go` 实现注册：`extension.SetProtocol("my-proto", func() base.Protocol { ... })`。
3. `imports/` 引用。
4. `make test` 验证。

### 任务 C：加一个 Registry
1. 参考 `registry/zookeeper/` 或 `registry/nacos/`。
2. 实现 `registry.Registry` / `registry.ServiceDiscovery` 相关接口，`init()` 注册。
3. `imports/` 引用。

### 任务 D：看架构 / 画调用链
1. 先读 `doc/` 下的 PNG 架构图。
2. 用 `Grep` 在 `common/extension` 找目标能力的注册/获取点，顺藤摸瓜到实现。

---

## 文档索引（Docs）

- `README.md` / `README_CN.md`：项目总览与快速开始
- `CONTRIBUTING.md`：贡献与 PR 流程
- `Makefile`、`.github/workflows/`：当前构建、测试和 CI 的实际入口
- `doc/`：架构图与说明
- `metrics/`、`otel/`：可观测性接入方式

---

## 给 Agent 的约束（Rules for the Agent）

1. **先读真实代码，再下结论**：动笔前用 `Glob`/`Grep`/`Read` 在仓库内确认 API 与接口签名，不要凭印象或旧版本写代码。
2. **扩展必须注册**：任何新实现都要在 `init()` 注册，并在 `imports/` 引用，否则不生效。
3. **遵循项目规范**：先读 `CONTRIBUTING.md`，以及当前工作区实际存在的项目指令；不要引用其他 PR 尚未合入的规范文件。
4. **测试驱动**：修复行为缺陷时先复现，再验证修复；涉及并发的 Go 改动运行受影响包的 race 测试。
5. **不确定就问/查**：对 dubbo-go 特有机制（SPI、URL 模型、Invoker 链）不确定时，先在仓库内搜示例，再动手；必要时向用户确认范围。
6. **版本对齐**：构建始终带 `GOTOOLCHAIN=go1.25.0+auto`，避免本地 Go 版本不符导致失败。
