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

随 [Apache dubbo-go](https://github.com/apache/dubbo-go) 一起维护的 AI Agent Skills——让你的编码助手真正会写 dubbo-go v3。

## 它能给你什么

你在用 dubbo-go 时，AI 助手现在知道自己在做什么了。

让它搭一个新的 provider 或 consumer，它会选对当前的 code-API 风格（`dubbo.NewInstance` + `server.WithServerProtocol`），接上你实际用的注册中心，第一次就能编译过。要写自定义 filter 或负载均衡？它懂 SPI 模式——`extension.SetXxx` 注册、blank import、按 service 或 reference 启用。要和 Java Dubbo 服务互通？它会根据你有的是 `.proto` 还是 Java 接口，正确选 Triple+Protobuf 或 Dubbo+Hessian2，POJO 类名也能对齐。

问它服务为什么连不上，它按结构化清单逐步排查——注册中心、协议匹配、序列化、filter 链——而不是瞎猜。要从 gRPC-Go 或 Spring Cloud 迁移？它会把概念逐一映射。

Skill 在上下文匹配时自动触发，不需要手动调用。

> 所有 skill 仅支持 dubbo-go **v3**，v1/v2 已废弃。

## 安装

不同 AI 工具安装方式不同。Claude Code 和 Cursor 内置市场，Codex 和 OpenCode 需要手动安装。

### Claude Code / Cursor

```bash
/plugin marketplace add apache/dubbo-go
/plugin install dubbo-go@dubbo-go-agent-skills
```

### Codex

告诉 Codex：

```
Fetch and follow instructions from https://raw.githubusercontent.com/apache/dubbo-go/main/.agents/.codex/INSTALL.md
```

详细文档：[.codex/INSTALL.md](.codex/INSTALL.md)

### OpenCode

在 `opencode.json` 中加：

```json
{
  "plugin": ["dubbo-go-agent-skills@git+https://github.com/apache/dubbo-go.git#path:.agents"]
}
```

### Gemini CLI

```bash
gemini extensions install https://github.com/apache/dubbo-go --path .agents
```

更新：

```bash
gemini extensions update dubbo-go-agent-skills
```

### 验证

新开一个会话，试试这些：

- "帮我搭一个用 Nacos 的 dubbo-go provider"
- "consumer 报 'no provider available' 是什么原因"
- "怎么从 Go 调一个 Java Dubbo 服务？"

助手应该会自动触发对应的 skill。

## 参考文档

面向用户的行为说明和教程，以官方 Dubbo Golang SDK 文档为主要参考：

- [Dubbo Golang SDK 文档](https://github.com/apache/dubbo-website/tree/master/content/zh-cn/overview/mannual/golang-sdk)

## Skills

### scaffolding

按 v3 code-API 风格（`dubbo.NewInstance` / `server.NewServer` / `client.NewClient`）生成 provider 或 consumer 骨架。先确认协议（Triple / Dubbo / gRPC）和注册中心（Nacos / ZooKeeper / etcd / 直连），再生成与官方 samples 一致、可直接编译运行的骨架。覆盖 OpenAPI、HTTP handler 挂载、HTTP/3。

### extensions

自定义 SPI 扩展——Filter、LoadBalance、Registry、Protocol、Router、Logger。讲清统一模式（`extension.SetXxx` + blank import + `WithFilter` / `WithLoadBalance`），附可运行模板和新人最常踩的"静默失败"陷阱。

### java-interop

dubbo-go 与 dubbo-java 的跨语言 RPC。新服务选 Triple+Protobuf，老 Java 接口选 Dubbo+Hessian2，POJO 类名、方法名大小写、curl 友好的 HTTP 路由格式都对齐好。

### debug

运行时报错的结构化排查。把错误或日志和已知模式匹配——"no provider available"、连接被拒、序列化不匹配、超时、filter panic、OpenAPI 404、AttachHTTPHandler 失败、shutdown 偏慢——给出针对性的检查清单。

### guide

架构、扩展点、最佳实践。覆盖 Instance、Protocol、Registry、Filter、Cluster、LoadBalance、Router、Triple OpenAPI、HTTP/3、CORS、graceful shutdown、可观测性，并要求 Agent 在选择示例时读取当前 `apache/dubbo-go-samples` README 和目录。

### migrate

分步骤迁移指引：

- **gRPC-Go**——概念映射、proto 复用、直连模式对照
- **Spring Cloud（Java）**——注册中心复用、Java/Go 共存
- **Gin / 纯 HTTP**——共存方案和 Triple REST 模式
- **dubbo-go v1/v2 → v3**——breaking-change 对照、最小迁移路径
- **YAML 重度依赖的 v3**——平滑迁移到 code API

### development

面向 `apache/dubbo-go` 仓库本身的贡献者——Go 工具链、包边界、校验命令、生成文件、仓库级禁止事项。**不**用于应用侧脚手架。

## 贡献

Skill 文件位于 `.agents/skills/<name>/SKILL.md`。

1. 直接修改或新增 skill。
2. 保持 frontmatter 的 `description` 聚焦触发场景——Agent 就是靠这一行决定要不要使用该 skill。
3. 只有当安装路径、skill 名称或支持的 Agent 发生变化时，才修改 `.agents/` 下的元数据。
4. 按 dubbo-go 正常贡献流程提交修改。

## License

Apache License 2.0
