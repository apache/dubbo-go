# CLI 模板依赖锁修复实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法跟踪进度。

**目标：** 恢复 `dubbogo-cli` 生成项目的完整 `go.sum`，清除主分支 Sonar `text:S8566` 新代码漏洞。

**架构：** application 与 sample generator 复用一个静态 checksum 源，并分别生成 `go.sum`；golden template 保存同一内容。现有动态协议源码生成和 `make test` 的 tidy 链路保持不变。

**技术栈：** Go 1.25、Go Modules、GNU Make、SonarQube Cloud、GitHub Actions。

---

## 文件职责与变更结构

- `tools/dubbogo-cli/cmd/gen_test.go`：从公开 generator 入口验证新项目立即包含依赖 checksum。
- `tools/dubbogo-cli/generator/internal/scaffold/gosum.go`：application 与 sample generator 共用的完整 checksum 内容。
- `tools/dubbogo-cli/generator/application/gomod.go`：为 newApp 注册 `go.sum` 输出。
- `tools/dubbogo-cli/generator/sample/mod.go`：为 newDemo 注册 `go.sum` 输出。
- `tools/dubbogo-cli/cmd/testGenCode/template/newApp/go.sum`：newApp golden checksum。
- `tools/dubbogo-cli/cmd/testGenCode/template/newDemo/go.sum`：newDemo golden checksum。

### 任务 1：建立 Sonar 回归红灯

- [ ] **步骤 1：增加行为测试**

在 `tools/dubbogo-cli/cmd/gen_test.go` 添加一个 table-driven 测试，依次调用 `application.Generate` 和 `sample.Generate`，读取生成目录的 `go.sum`，断言文件非空，并包含：

```text
dubbo.apache.org/dubbo-go/v3 v3.3.1
google.golang.org/protobuf v1.34.2
```

- [ ] **步骤 2：运行测试确认正确失败**

运行：

```powershell
go test ./cmd -run '^TestGeneratedProjectsIncludeDependencyLocks$' -count=1 -v
```

预期：`newApp` 和 `newDemo` 子测试因生成目录缺少 `go.sum` 而失败；失败不能来自编译、路径拼写或依赖下载。

### 任务 2：恢复完整 dependency checksum 输出

- [ ] **步骤 1：生成当前依赖图的完整 go.sum**

在隔离 probe 中生成 fresh newApp/newDemo，执行协议生成和 `go mod tidy`，确认两者的 `go.sum` 完全相同。不得直接复用依赖版本已经变化的旧 checksum。

- [ ] **步骤 2：恢复共享 checksum 与 generator 注册**

创建 `generator/internal/scaffold/gosum.go`，并在 application/sample 的模块文件注册逻辑中新增 `go.sum` file generator。

- [ ] **步骤 3：恢复两个 golden go.sum**

将同一份完整 checksum 写入 newApp 和 newDemo golden template。

- [ ] **步骤 4：运行回归测试确认绿灯**

运行：

```powershell
go test ./cmd -run '^TestGeneratedProjectsIncludeDependencyLocks$|^TestNewApp$|^TestNewDemo$' -count=1 -v
```

预期：三个顶层测试及所有子测试通过。

### 任务 3：验证并交付 PR

- [ ] **步骤 1：验证测试质量**

按 test-guard 检查新增测试只验证公开行为、没有 mock、两个 generator 变体使用同一 table-driven 测试，并明确对应 Sonar 回归。

- [ ] **步骤 2：执行模块验证**

运行：

```powershell
go test ./... -count=1
go vet ./...
```

工作目录：`tools/dubbogo-cli`。预期全部退出 0。

- [ ] **步骤 3：执行 Linux/WSL 生成项目验证**

运行 CI 等价的 `make test-generated-projects`。该目标为 fresh newApp/newDemo 执行现有的 `proto-gen -> tidy -> go test` 链路，预期两个子测试均退出 0。

模板刻意保留只含直接依赖的最小 `go.mod`，所以不把独立 `go mod tidy -diff` 的零差异作为验收条件；锁文件正确性由生成器回归测试、golden 全文件比对和真实 E2E 构建共同验证。

- [ ] **步骤 4：执行仓库状态和差异门禁**

运行：

```powershell
git diff --check
git status --short
```

预期：无空白错误；只包含设计、计划、回归测试、共享 checksum、两个 generator 注册和两个 golden `go.sum`。

- [ ] **步骤 5：提交、审查、推送和创建 PR**

使用带 Signed-off-by 的 Lore 格式提交；请求独立代码审查，修复 Critical/Important 反馈；重新验证后推送 `codex/fix-sonar-cli-template-locks`，创建以 `develop` 为 Base 的 PR，并复检 PR Head、Diff 文件、checks 和 Sonar 状态。
