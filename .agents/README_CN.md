<!--
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# dubbo-go Agent Skills

[English](README.md) | 中文

本目录存放随 [Apache dubbo-go](https://github.com/apache/dubbo-go) 仓库一起维护的 AI Agent Skills。这些 skills 用于帮助编码 Agent 处理 dubbo-go 仓库源码、dubbo-go v3 应用、示例生成、扩展点、Java 互通、问题排查和迁移任务。

这套包放在 `.agents/` 下，是为了把多 Agent 分发元数据集中在一起，同时避免覆盖 dubbo-go 仓库根目录已有文件。

> 注意：所有 skills 都面向 dubbo-go v3。v1 和 v2 已废弃。

## 目录结构

- `skills/`：Codex 可直接加载的 skills。Codex 在本仓库工作时会读取该目录。
- `.codex/INSTALL.md`：在本仓库外全局使用这些 skills 的 Codex 手动安装说明。
- `.claude-plugin/marketplace.json`：面向 Claude Code 插件市场的元数据。
- `.opencode/plugins/dubbo-go-agent-skills.js`：OpenCode 适配器，用于暴露本目录内的 skills。
- `GEMINI.md` 和 `gemini-extension.json`：Gemini CLI 扩展入口。
- `plugin.json` 和 `package.json`：供能够以 `.agents` 为包根目录的工具读取的通用元数据。

## 使用方式

在 Apache dubbo-go 仓库内工作时，Codex 可以直接从 `.agents/skills` 加载仓库内置 skills，不需要再单独克隆一个 skills 仓库。

如果要在本仓库外全局使用这套 Codex skills，请参考 [.codex/INSTALL.md](.codex/INSTALL.md)。

如果用于 OpenCode、Gemini CLI 或 Claude Code 的插件包装，请把 `.agents` 视为包根目录。本目录中的元数据路径都按 `.agents` 为根来组织。

## Skills

### development

指导 Agent 修改 apache/dubbo-go 仓库本身，覆盖当前 Go/toolchain 要求、包边界、验证命令、生成文件和仓库级禁止事项。

### scaffolding

生成当前 dubbo-go v3 code API 风格的 provider 或 consumer 骨架，覆盖直连模式、注册中心服务、Protobuf 生成、OpenAPI、HTTP handler 挂载、HTTP/3 和当前示例模式。

### extensions

指导编写自定义 SPI 扩展，例如 Filter、LoadBalance、Router、Registry、Protocol、ConfigCenter、Logger。覆盖 `extension.SetXxx` 注册模式、blank import、启用方式和常见失败原因。

### java-interop

指导 dubbo-go 和 dubbo-java 互通。根据服务形态和兼容性需求，在 Triple+Protobuf 与 Dubbo+Hessian2 之间做选择，并处理服务发现映射和跨语言序列化问题。

### debug

为运行时问题提供结构化排查流程，例如 provider 找不到、注册中心映射、连接失败、序列化不匹配、超时、OpenAPI、HTTP handler 挂载、shutdown 和 filter panic。

### guide

解释当前 dubbo-go 架构、扩展点和最佳实践，覆盖 Instance、Protocol、Registry、Metadata、Filter、Cluster、LoadBalance、Router、Triple OpenAPI、HTTP/3、CORS、graceful shutdown、可观测性和示例位置。

### migrate

指导从 gRPC-Go、Spring Cloud、Gin/纯 HTTP、dubbo-go v1/v2、YAML-heavy 应用、Java Dubbo 和 Hessian2 服务迁移到当前 dubbo-go v3 模式。

## 贡献

Skills 位于 `.agents/skills`。

1. 在 `.agents/skills/<name>/SKILL.md` 下新增或修改 skill。
2. 保持 `SKILL.md` frontmatter 简洁，并聚焦触发场景。
3. 只有当安装路径、skill 名称或支持的 Agent 发生变化时，才更新本目录的元数据。
4. 按 dubbo-go 正常贡献流程提交修改。

## License

Apache License 2.0
