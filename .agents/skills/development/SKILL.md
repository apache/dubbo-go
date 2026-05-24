---
name: dubbo-go-development
description: Use when modifying, reviewing, testing, debugging, or documenting the apache/dubbo-go repository itself, including Go code, generated code, config structs, tools, CI, tests, or repository-local agent skills.
---

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


# dubbo-go Repository Development

## Overview

Working inside `apache/dubbo-go` itself, not just using the framework from an application. Treat package boundaries, public APIs, and generated files as load-bearing - break them only with explicit user intent.

## When to Use

- Editing files under `apache/dubbo-go` (`client/`, `server/`, `protocol/`, `registry/`, `cluster/`, `filter/`, `metadata/`, `tools/`, etc.)
- Adding or updating unit tests, mocks, or fixtures inside this repo
- Touching `Makefile`, CI config, generated Protobuf/Triple stubs, or `tools/imports-formatter`
- Modifying `.agents/skills/*` documentation

For application-side changes, use `dubbo-go-scaffolding`. For runtime failures, use `dubbo-go-debugging`.

## Current Branch Facts

- Module path: `dubbo.apache.org/dubbo-go/v3`
- Go version in `go.mod`: `1.25.0`
- Makefile test command uses `GOTOOLCHAIN=go1.25.0+auto`
- CI still runs `make check-fmt`, `make test`, and `make lint`
- Import style is enforced by `tools/imports-formatter`
- `make fmt` also runs the Go modernize analyzer to replace eligible `interface{}` with `any`

## Workflow

1. Read the owning package and nearby tests before editing.
2. Keep changes scoped to the owning module boundary.
3. Prefer current code API patterns over old YAML/config globals for new examples.
4. Preserve compatibility in public APIs unless the user explicitly asks for a breaking change.
5. Add or update focused tests for behavior changes.
6. Run the smallest meaningful validation first, then broader commands when risk is higher.
7. Summarize changed files, behavior, and validation results.

## Important Package Boundaries

- `dubbo.go`, `options.go`, `instance_options_init.go`: instance-level option initialization and root config bridging
- `client/`: reference options, consumer URL construction, invoker creation
- `server/`: provider options, service registration, `AttachHTTPHandler`, service metadata
- `protocol/base`, `protocol/result`, `protocol/invocation`: core invocation interfaces
- `protocol/triple`: main protocol implementation, reflection, HTTP/3, CORS, OpenAPI, attached HTTP hosting
- `registry/`: registry interfaces, service discovery, application-level mapping
- `metadata/`: metadata report and service name mapping
- `cluster/`: cluster strategies, directory, load balance, router chain
- `filter/`: provider/consumer filters; most runtime interception belongs here
- `config/`, `global/`, `config_center/`: YAML/config models and dynamic config loading
- `graceful_shutdown/`: shutdown runtime state and options
- `logger/`, `metrics/`, `otel/`: observability
- `tools/`: CLI, schema, generators, imports formatter, variadic RPC scanner

## Current Implementation Notes

- Config loading uses koanf. Do not introduce viper-based assumptions into new code.
- Triple config now includes CORS, HTTP/3, and OpenAPI sections.
- `server.AttachHTTPHandler` hosts one existing `http.Handler` on an explicit Triple port before `Serve`.
- Graceful shutdown has separate total timeout, step timeout, notify timeout, consumer update wait time, offline request window, and closing invoker expiry.
- Application-level service discovery may require metadata mapping or explicit `client.WithProvidedBy`.
- Registry config can choose service/interface/all registration through registry options.
- Router config supports static injection for routers that implement `router.StaticConfigSetter`.
- New extension code should import `protocol/base` and `protocol/result`, not old aliases.

## Validation Commands

Use the command that matches the scope.

```bash
# format repository code
make fmt

# check formatting exactly like CI
make check-fmt

# targeted package tests
GOTOOLCHAIN=go1.25.0+auto go test ./protocol/triple/...
GOTOOLCHAIN=go1.25.0+auto go test ./server/...

# full repository tests plus dubbogo-cli submodule tests
make test

# lint
make lint

# warn about exported variadic RPC contracts
make rpc-contract-check
```

If only `.agents` documentation changes, validate skill metadata and adapters instead of running the Go suite.

## Generated and Tooling Files

- Do not hand-edit generated Protobuf or Triple files unless the generator source is also in scope.
- For user services, generate with `protoc-gen-go` and `protoc-gen-go-triple/v3`.
- For OpenAPI generation, use runtime Triple OpenAPI or `tools/protoc-gen-triple-openapi` depending on the task.
- `tools/dubbogo-cli` has its own module and tests; run tests from that directory when editing it.

## Do Not

- Do not remove tests to make validation pass.
- Do not change public option names casually.
- Do not flatten package boundaries with unrelated refactors.
- Do not overwrite generated files without explaining the generator source.
- Do not introduce new dependencies unless the existing packages cannot reasonably solve the problem.

## Red Flags - Stop and Reconsider

- About to delete or skip a failing test to "unblock" CI
- About to rename or remove an exported option/function in `client/`, `server/`, `protocol/`, or `registry/` without a migration note
- About to bypass `tools/imports-formatter` by hand-editing import groupings
- About to replace koanf-based config loading with a viper-style assumption
- About to `--no-verify` a commit, or skip `make check-fmt` / `make lint`
- About to introduce a third-party dependency to do something `gost`, `koanf`, or stdlib already covers

If any of these fire, stop and surface the tradeoff to the user before proceeding.

## Related Skills

- `dubbo-go-guide` - conceptual map of packages, options, and current capabilities
- `dubbo-go-extensions` - when the work is adding or fixing an SPI registration in this repo
- `dubbo-go-debugging` - when reproducing a user-reported failure to find the fix point
