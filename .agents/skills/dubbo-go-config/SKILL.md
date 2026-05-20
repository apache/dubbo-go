---
name: dubbo-go-config
description: >-
  Implements and reviews dubbo-go configuration changes. Use when the user asks about dubbo.Load, config.Load,
  RootConfig, InstanceOptions, global config structs, YAML or JSON loading, koanf parsing, config center integration,
  dynamic configuration, hot reload, config post processors, or packages loader.go, config/, global/, config_center/,
  and common/config/. Do not use for runtime export/refer behavior unless configuration injection is the contract.
---

# dubbo-go Config

## Purpose

Use this skill to change or explain how dubbo-go loads, initializes, validates, hot-reloads, and exposes configuration to runtime client and server code.

## When to use

Use for configuration files, builder APIs, `RootConfig`, `InstanceOptions`, global config models, config center factories, hot reload allowlists, and config post-processing.

Do not use for protocol transport, registry events, or runtime export/refer behavior unless configuration injection is the contract being changed.

## Inputs

Required:
- Config key, config struct, loader API, or package path.
- Intended behavior or observed parse/init bug.

Optional:
- YAML or JSON snippet.
- Config center protocol and data ID or group.
- Related client, server, reference, or service option.

If missing, inspect `loader.go`, `config/config_loader.go`, and the relevant `global/*_config.go` or `config/*_config.go`.

## Workflow

1. Identify whether the task targets top-level `dubbo.Load` and `InstanceOptions` or legacy `config.Load` and `RootConfig`.
2. Read the owning config model under `global/` and compatibility or builder code under `config/`.
3. Trace initialization into `InstanceOptions.init`, `RootConfig.Init`, service/reference init, or config center startup.
4. For dynamic config, inspect `config/config_center_config.go`, `config_center/`, and `common/extension/config_center_factory.go`.
5. For hot reload, inspect `loader.go` and keep `safeChanged` semantics narrow.
6. Read `references/config-flow.md` before changing loader behavior or public config keys.

## Output format

Return changed config keys or structs, loader path, runtime injection impact, validation performed, and any compatibility notes.

## Validation

- Confirm new config fields are represented in the right `global/` or `config/` model and initialized before runtime use.
- Confirm config center changes register the correct extension factory.
- Confirm hot reload accepts only intended mutable keys.
- Run focused tests such as `go test ./config ./global ./config_center/...`; use `make test` for broad config behavior.

## Edge cases

- `dubbo.Load` starts provider and consumer runtime; `config.Load` only initializes `RootConfig`.
- Some config values are transported as URL params, while TLS, application, registry, shutdown, and Triple details often travel as URL attributes.
- Existing compatibility tests may depend on legacy field names.

## References

- Read `references/config-flow.md` for loader, config center, and runtime injection details.
