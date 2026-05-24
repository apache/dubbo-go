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

This directory contains AI agent skills bundled with [Apache dubbo-go](https://github.com/apache/dubbo-go). The skills help coding agents work with the dubbo-go repository, dubbo-go v3 applications, examples, extension points, Java interoperability, debugging, and migration tasks.

The package lives under `.agents/` so it can include multi-agent distribution metadata without overwriting dubbo-go repository root files.

> Note: All skills target dubbo-go v3. v1 and v2 are deprecated.

## Layout

- `skills/`: Codex-compatible skills. Codex reads this directory when working in this repository.
- `.codex/INSTALL.md`: manual Codex installation notes for using these skills outside this checkout.
- `.claude-plugin/marketplace.json`: Claude Code plugin marketplace metadata for this `.agents` package.
- `.opencode/plugins/dubbo-go-agent-skills.js`: OpenCode adapter that exposes the bundled skills.
- `GEMINI.md` and `gemini-extension.json`: Gemini CLI extension entry points.
- `plugin.json` and `package.json`: generic plugin/package metadata for tools that can consume a package rooted at `.agents`.

## Using These Skills

When working inside an Apache dubbo-go checkout, Codex can load the repository-local skills from `.agents/skills` directly. No separate clone of a skills repository is required.

For global Codex usage outside this checkout, follow [.codex/INSTALL.md](.codex/INSTALL.md).

For OpenCode, Gemini CLI, or Claude Code packaging, treat `.agents` as the package root. The metadata files in this directory are intentionally relative to `.agents`.

## Skills

### development

Guides agents modifying the apache/dubbo-go repository itself. It covers current Go/toolchain expectations, package boundaries, validation commands, generated files, and repository-specific do-not rules.

### scaffolding

Generates provider or consumer skeletons in the current dubbo-go v3 code-API style. It covers direct mode, registry-backed services, Protobuf generation, OpenAPI, HTTP handler mounting, HTTP/3, and current sample patterns.

### extensions

Guides custom SPI extensions such as Filter, LoadBalance, Router, Registry, Protocol, ConfigCenter, and Logger. It covers the `extension.SetXxx` registration pattern, blank imports, activation options, and common failure modes.

### java-interop

Guides interoperability between dubbo-go and dubbo-java. It helps choose Triple+Protobuf or Dubbo+Hessian2, handle service discovery mapping, and debug cross-language serialization.

### debug

Provides structured diagnosis for runtime issues such as missing providers, registry mapping, connection failures, serialization mismatches, timeouts, OpenAPI issues, attached HTTP handlers, shutdown behavior, and filter panics.

### guide

Explains current dubbo-go architecture, extension points, and best practices. It covers Instance, Protocol, Registry, Metadata, Filter, Cluster, LoadBalance, Router, Triple OpenAPI, HTTP/3, CORS, graceful shutdown, observability, and sample locations.

### migrate

Guides migration from gRPC-Go, Spring Cloud, Gin/plain HTTP, older dubbo-go v1/v2 usage, YAML-heavy apps, Java Dubbo, and Hessian2 services to current dubbo-go v3 patterns.

## Contributing

Skills live in `.agents/skills`.

1. Add or edit a skill under `.agents/skills/<name>/SKILL.md`.
2. Keep `SKILL.md` frontmatter concise and trigger-focused.
3. Update this directory's metadata only when install paths, skill names, or supported agents change.
4. Submit the change through the normal dubbo-go contribution workflow.

## License

Apache License 2.0
