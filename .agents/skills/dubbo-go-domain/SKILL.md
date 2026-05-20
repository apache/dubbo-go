---
name: dubbo-go-domain
description: >-
  Use this skill when working on the dubbo-go codebase, including implementing features, fixing bugs, writing tests,
  reviewing code, updating documentation, or routing work to a narrower dubbo-go-* domain skill. Use for cross-module
  questions, end-to-end flows, subsystem boundaries, or deciding whether config, runtime, protocol, registry, metadata,
  cluster routing, filters and observability, or tools guidance applies.
---

# dubbo-go Project Guide

## Project Overview

dubbo-go is the Go implementation of Apache Dubbo. It is an RPC and microservice framework that provides service export and reference, registry and service discovery, protocol interoperability, cluster fault tolerance, traffic governance, configuration, metadata, metrics, tracing, and developer tools.

It mainly provides:
- Public client and server APIs for Go services.
- Dubbo, Dubbo3, Triple, gRPC, REST, JSONRPC, and related transport support.
- Registry integration with Nacos, Zookeeper, Etcd, Polaris, and application-level service discovery.
- Metadata collection, metadata reporting, service name mapping, routing, load balancing, filters, metrics, tracing, and CLI or code generation tools.

This repository does not own external registry servers, Dubbo Java implementation, sample applications outside this repository, or production deployment infrastructure.

## Tech Stack

- Language: Go, module `dubbo.apache.org/dubbo-go/v3`.
- Go version: `go 1.25.0` in `go.mod`; root `Makefile` uses `GOTOOLCHAIN=go1.25.0+auto` for tests.
- Build: Go modules plus nested Go modules under selected `tools/` directories.
- Test: `go test`, `testify`, `gomock`, and focused package tests.
- Lint and format: `go fmt`, `imports-formatter`, `go vet`, `golangci-lint`, and `make fmt` or `make lint`.

Do not introduce a new framework or dependency unless the task requires it and the compatibility impact is clear.

## Repository Structure

- `dubbo.go`, `loader.go`: public instance and top-level load/start APIs.
- `config/`, `global/`, `config_center/`, `common/config/`: config models, loading, config center, and compatibility behavior.
- `client/`, `server/`, `proxy/`, `graceful_shutdown/`: runtime client/server construction, refer/export, proxy, and shutdown behavior.
- `protocol/`, `remoting/`, `common/url.go`: protocol interfaces, codecs, transports, URL contract, invokers, and exporters.
- `registry/`, `metadata/`: registry discovery, service discovery, metadata, reports, mapping, and revisions.
- `cluster/`: directories, routers, load balancing, and cluster fault tolerance.
- `filter/`, `metrics/`, `otel/`, `logger/`: filters, observability, tracing, metrics, and logging.
- `tools/`: CLI, schema generator, protoc plugins, import formatter, and RPC contract scanner.
- `docs/`, `doc/`: project documentation and images.

## Workflow

When modifying code:

1. Classify the owning subsystem and load the narrower `dubbo-go-*` skill when relevant.
2. Read the existing implementation and tests before editing.
3. Make the smallest correct change that follows current package boundaries.
4. Prefer existing abstractions, extension registries, URL params, and URL attributes over new patterns.
5. Add or update focused tests for behavior changes.
6. Run the most relevant validation command.
7. Summarize changed files, behavior, compatibility impact, and validation result.

When fixing bugs:

1. Reproduce the issue or identify the failing path from tests, logs, or code.
2. Find the smallest root cause.
3. Make a minimal safe change.
4. Add a regression test when feasible.
5. Explain why the fix works and what risk remains.

When reviewing code:

1. Check behavior, compatibility, concurrency, error handling, generated file handling, and missing tests first.
2. Ground findings in file and line references.
3. Keep summary secondary to concrete risks.

## Coding Guidelines

- Keep changes focused and reviewable.
- Follow existing package structure, naming, and extension registration patterns.
- Preserve backward compatibility for public APIs, config keys, URL params, wire behavior, and generated output unless explicitly requested.
- Use structured config, URL, and parser APIs instead of ad hoc string handling when available.
- Avoid broad refactors during bug fixes.
- Do not change unrelated files.

## Testing and Validation

Use the most relevant command:

```bash
go test ./...
make test
make fmt
make lint
make rpc-contract-check
```

For focused work, prefer package-level tests first, for example:

```bash
go test ./config ./global ./config_center/...
go test ./client ./server ./proxy ./graceful_shutdown
go test ./protocol/... ./remoting/...
go test ./registry/... ./metadata/...
go test ./cluster/...
go test ./filter/... ./metrics/... ./otel/...
```

For nested tools, run tests from the owning module directory.

If validation cannot be run or fails, report the command, the error, and whether it appears related to the current change.

## Do Not

- Do not rewrite large parts of the project without explicit instruction.
- Do not introduce new dependencies or frameworks unless necessary.
- Do not change public APIs, config keys, wire contracts, or generated output without documenting compatibility impact.
- Do not remove or weaken tests to make validation pass.
- Do not commit secrets, tokens, passwords, private keys, or private config.
- Do not modify generated files manually unless the generation source is also updated or the repository already treats that file as hand-maintained.
- Do not ignore existing uncommitted user changes.

## Common Tasks

### Choose a Domain Skill

1. Map the request to the primary package.
2. Use `references/architecture-map.md` when ownership is unclear.
3. Load one narrower skill when possible.
4. If multiple domains are involved, follow runtime flow: config, runtime, protocol, registry or metadata, cluster, filters, observability, tools.

### Add a Feature

1. Find the owning module and existing tests.
2. Add the core behavior using existing abstractions.
3. Add or update tests.
4. Update documentation when public behavior changes.
5. Run focused validation.

### Add a Configuration Option

1. Define the config key in the owning config model.
2. Add default and initialization behavior.
3. Wire it into the consuming module through existing options or URL params/attributes.
4. Add tests for default and custom behavior.
5. Update docs or schemas when relevant.

### Fix a Bug

1. Identify the root cause.
2. Make the minimal fix.
3. Add a regression test when possible.
4. Verify with the narrowest relevant command.

## Output Format

Return the selected skill or subsystem, changed files, behavior change, compatibility impact, validation performed, and any residual risk.

## Edge cases

- `dubbo.Load` and `config.Load` both load configuration, but they serve different public APIs.
- Registry discovery and metadata reporting are tightly coupled; use `dubbo-go-metadata` when revisions or metadata reports control behavior.
- Filters wrap protocols through `protocol/protocolwrapper`; use `dubbo-go-protocol` when transport semantics change and `dubbo-go-observability-filters` when interception semantics change.
- Some `tools/` subdirectories are independent Go modules.

## References

- Read `references/architecture-map.md` for subsystem ownership and end-to-end flow.
- Read `references/project-rules.md` for compatibility, generated-file, validation, and PR guidance.
