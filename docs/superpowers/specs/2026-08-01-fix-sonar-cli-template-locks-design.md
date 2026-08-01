# 修复 CLI 模板 Sonar 依赖锁告警设计

## 背景

`main@cdda54be1f353e372a80940729cdb380702c325e` 的 SonarQube Cloud 分析报告两个 `text:S8566` 漏洞：

- `tools/dubbogo-cli/cmd/testGenCode/template/newApp/go.mod`
- `tools/dubbogo-cli/cmd/testGenCode/template/newDemo/go.mod`

两个模板在 #3585/#3587 的 scaffold 重构中删除了 `go.sum`。生成项目的 `make test` 会先运行 `go mod tidy`，因此正常编译路径仍可工作，但刚生成的项目缺少已提交的依赖 checksum，导致主分支新代码安全评级降为 C。

## 目标

1. `application.Generate` 与 `sample.Generate` 生成的项目立即包含完整、非空的 `go.sum`。
2. 两个 golden template 保存与 generator 输出一致的 `go.sum`，使 Sonar 能在仓库树中识别锁文件。
3. 保留 #3585/#3587 的动态协议源码架构；不恢复内嵌的 `*.pb.go`、`*.triple.go` 或其他已删除生成文件。
4. 生成项目运行 `go mod tidy -diff` 时不需要修正依赖 checksum。

## 方案

恢复一个由 application 与 sample generator 共用的 checksum 源。两个 generator 在写入 `go.mod` 的同时写入相同的 `go.sum`，两个 golden template 保存相同内容。

checksum 必须由当前模板的 `go.mod` 和实际生成后的协议源码通过 Go 1.25/Linux 的 `go mod tidy` 生成，不能沿用旧版本依赖图。`make test` 仍保留 `proto-gen -> tidy -> go test`，用于后续依赖或生成代码发生变化时校正模块元数据。

不采用 Sonar 排除或误报标记，因为生成项目确实应携带可审计的依赖 checksum；不采用在 generator 执行期间调用 `go mod tidy`，避免生成命令新增网络、Go 工具链和外部进程依赖。

## 测试设计

先增加 `TestGeneratedProjectsIncludeDependencyLocks`：

1. 通过公开的 `application.Generate` 和 `sample.Generate` 分别生成项目到 `t.TempDir()`。
2. 读取生成目录中的 `go.sum`。
3. 断言文件非空，并包含当前两个直接依赖版本的 checksum 记录。

该测试在当前 Head 上应因 `go.sum` 不存在而失败。实现后，同一测试应通过；现有 `TestNewApp`/`TestNewDemo` 继续验证 generator 与 golden template 的完整文件集合和逐字节内容一致。

补充验证：

- `go test ./cmd -run '^TestGeneratedProjectsIncludeDependencyLocks$' -count=1 -v`
- `go test ./... -count=1`，工作目录为 `tools/dubbogo-cli`
- 对 fresh `newApp` 和 `newDemo` 执行 `go mod tidy -diff`
- 在 Linux/WSL 执行生成项目 E2E `make test`
- `go vet ./...`
- `git diff --check`

## 发布边界

修复通过正常 PR 合入 `develop`。不修改、重打或重新发布 `v3.3.2`；根模块发布 ZIP 不包含 `tools/dubbogo-cli` 嵌套模块，本修复面向后续主分支质量状态和新的 CLI 源码版本。
