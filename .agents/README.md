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

[English](README.md) | [中文](README_CN.md)

AI agent skills bundled with [Apache dubbo-go](https://github.com/apache/dubbo-go) — make your coding assistant fluent in dubbo-go v3.

## What This Gives You

When you're using dubbo-go, your AI assistant now actually knows what it's doing.

Ask it to scaffold a new provider or consumer and it picks the current code-API style (`dubbo.NewInstance` + `server.WithServerProtocol`), wires up the registry you actually use, and produces something that compiles on the first try. Need a custom filter or load balancer? It knows the SPI pattern — `extension.SetXxx` registration, blank imports, per-service or per-reference activation. Calling a Java Dubbo service? It picks Triple+Protobuf or Dubbo+Hessian2 based on whether you have a `.proto` or a Java interface, and gets the POJO class names right.

Ask why your service won't connect, and it triages by structured checklist — registry, protocol, serialization, filter chain — instead of guessing. Migrating from gRPC-Go or Spring Cloud? It maps the concepts side by side.

Skills activate automatically when the conversation matches; you don't invoke them by name.

> All skills target dubbo-go **v3**. v1 and v2 are deprecated.

## Install

Install paths vary by AI tool. Claude Code and Cursor read a marketplace; Codex and OpenCode are manual.

### Claude Code / Cursor

```bash
/plugin marketplace add apache/dubbo-go
/plugin install dubbo-go@dubbo-go-agent-skills
```

### Codex

Tell Codex:

```
Fetch and follow instructions from https://raw.githubusercontent.com/apache/dubbo-go/main/.agents/.codex/INSTALL.md
```

Full instructions: [.codex/INSTALL.md](.codex/INSTALL.md)

### OpenCode

Add to `opencode.json`:

```json
{
  "plugin": ["dubbo-go-agent-skills@git+https://github.com/apache/dubbo-go.git#path:.agents"]
}
```

### Verify

Open a new session and try one of these:

- "Scaffold a dubbo-go provider with Nacos"
- "Why does my consumer log 'no provider available'?"
- "How do I call a Java Dubbo service from Go?"

The assistant should auto-trigger the matching skill.

## Reference Docs

Use the official Dubbo Golang SDK documentation as the primary reference for user-facing behavior and tutorials:

- [Dubbo Golang SDK documentation](https://github.com/apache/dubbo-website/tree/master/content/zh-cn/overview/mannual/golang-sdk)

## Skills

### scaffolding

Generates provider or consumer skeletons in v3 code-API style (`dubbo.NewInstance` / `server.NewServer` / `client.NewClient`). Asks about protocol (Triple / Dubbo / gRPC) and registry (Nacos / ZooKeeper / etcd / direct) first, then produces a complete, compilable skeleton matching the official samples. Covers OpenAPI, HTTP handler attachment, and HTTP/3.

### extensions

Custom SPI extensions — Filter, LoadBalance, Registry, Protocol, Router, Logger. Explains the uniform pattern (`extension.SetXxx` + blank import + `WithFilter` / `WithLoadBalance`), with runnable templates and the silent-failure modes that catch newcomers.

### java-interop

Cross-language RPC between dubbo-go and dubbo-java. Picks Triple+Protobuf for new services, Dubbo+Hessian2 for existing Java-defined ones, and gets POJO class names, method-name casing, and the curl-friendly HTTP route format right.

### debug

Structured diagnosis for runtime errors. Matches your error or log against known patterns — "no provider available", connection refused, serialization mismatch, timeout, filter panic, OpenAPI 404, AttachHTTPHandler failure, slow shutdown — and gives a targeted checklist.

### guide

Architecture, extension points, and best practices. Covers Instance, Protocol, Registry, Filter, Cluster, LoadBalance, Router, Triple OpenAPI, HTTP/3, CORS, graceful shutdown, observability, and tells agents to read the current `apache/dubbo-go-samples` README and directories when choosing examples.

### migrate

Step-by-step migration guidance:

- **gRPC-Go** — concept mapping, proto reuse, direct mode parallels
- **Spring Cloud (Java)** — registry sharing, Java/Go coexistence path
- **Gin / plain HTTP** — coexistence options and Triple REST mode
- **dubbo-go v1/v2 → v3** — breaking-change table, minimum migration path
- **YAML-heavy v3** — incremental move to the code API

### development

For contributors modifying the `apache/dubbo-go` repository itself — Go toolchain, package boundaries, validation commands, generated files, repository-level do-not rules. Not for application-side scaffolding.

## Contributing

Skills live in `.agents/skills/<name>/SKILL.md`.

1. Edit or add a skill in place.
2. Keep the frontmatter `description` trigger-focused — that line is what the agent reads to decide whether to use the skill.
3. Update `.agents/` metadata only when install paths, skill names, or supported agents change.
4. Submit through the normal dubbo-go contribution workflow.

## License

Apache License 2.0
