---
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
name: dubbo-go-development
description: Use when modifying, reviewing, testing, or debugging the apache/dubbo-go repository itself - including framework Go code, generated stubs, config structs, tools, CI, tests, or repository-local agent skills. Not for application-side scaffolding.
---

# Contributing to apache/dubbo-go

This skill is for working **inside** the dubbo-go repository, not for using the framework from an application. For app-side work, use `dubbo-go-scaffolding`.

## Branch Facts

- Module path: `dubbo.apache.org/dubbo-go/v3`
- Default branch: `main`
- `go.mod` Go version: `1.25.0`
- Toolchain: `GOTOOLCHAIN=go1.25.0+auto` (referenced by the Makefile)
- Import-block format is enforced by `tools/imports-formatter`
- `make fmt` also runs the Go `modernize` analyzer to replace eligible `interface{}` with `any`

## Validation

Pick the smallest scope that covers the change:

```bash
make fmt              # format the tree
make check-fmt        # exactly what CI runs
make lint             # golangci-lint
make test             # full suite + dubbogo-cli submodule
make rpc-contract-check  # warn on exported variadic RPC contracts

# targeted package tests
GOTOOLCHAIN=go1.25.0+auto go test ./protocol/triple/...
GOTOOLCHAIN=go1.25.0+auto go test ./server/...
```

For `.agents/` doc-only changes, skip the Go suite and validate the skill frontmatter and adapter (`.agents/.opencode/plugins/dubbo-go-agent-skills.js`) instead.

## Package Boundaries

| Path | Scope |
|---|---|
| `dubbo.go`, `options.go`, `instance_options_init.go` | Instance-level options and root config bridging |
| `client/` | Reference options, consumer URL, invoker creation |
| `server/` | Provider options, service registration, `AttachHTTPHandler` |
| `protocol/base`, `protocol/result`, `protocol/invocation` | Core invocation interfaces |
| `protocol/triple` | Triple impl: HTTP/2, HTTP/3, CORS, OpenAPI, reflection, attached HTTP |
| `registry/`, `metadata/` | Registry, service discovery, app-to-interface mapping |
| `cluster/` | Cluster strategies, directory, load balance, router chain |
| `filter/` | Provider/consumer filters |
| `config/`, `global/`, `config_center/` | YAML/koanf config models, dynamic config loading |
| `graceful_shutdown/` | Shutdown timeouts and lifecycle |
| `logger/`, `metrics/`, `otel/` | Observability |
| `tools/` | CLI, schema, generators, imports formatter, RPC contract scanner |

## Current Implementation Notes

- Config loading uses **koanf**. Do not introduce viper-style assumptions in new code.
- Triple config now includes CORS, HTTP/3, and OpenAPI sub-options.
- `server.AttachHTTPHandler` hosts one `http.Handler` on an explicit Triple port and must be called before `Serve`.
- Graceful shutdown has separate total / step / notify / consumer-update / offline-request / closing-invoker timeouts.
- Application-level discovery may need metadata mapping or explicit `client.WithProvidedBy`.
- Registry config can choose service / interface / all registration via registry options.
- Routers implementing `router.StaticConfigSetter` accept static injection.
- New code should import `protocol/base` and `protocol/result` directly, not the older aliases.

## Generated and Tooling Files

- Do not hand-edit generated Protobuf / Triple files unless the generator source is also in scope.
- User-side stubs come from `protoc-gen-go` and `github.com/dubbogo/protoc-gen-go-triple/v3`.
- For OpenAPI: runtime Triple OpenAPI or `tools/protoc-gen-triple-openapi`, depending on task.
- `tools/dubbogo-cli` has its own module — run tests from that directory when editing it.

## Red Flags

Stop and surface the tradeoff before doing any of these:

- Removing or skipping a failing test to "unblock" CI
- Renaming or removing an exported option/function in `client/`, `server/`, `protocol/`, or `registry/` without a migration note
- Bypassing `tools/imports-formatter` with hand-edited import groupings
- Replacing koanf-based config with a different config library
- `--no-verify` on a commit, or skipping `make check-fmt` / `make lint`
- Introducing a third-party dependency for something `gost`, `koanf`, or stdlib already covers

## Related Skills

- `dubbo-go-guide` — conceptual map of packages and options
- `dubbo-go-extensions` — when adding or fixing an SPI in this repo
- `dubbo-go-debugging` — when reproducing a user-reported failure
