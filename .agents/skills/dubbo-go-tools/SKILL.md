---
name: dubbo-go-tools
description: >-
  Implements and reviews dubbo-go developer tools and code generation. Use when the user asks about tools/dubbogo-cli,
  tools/dubbo-go-schema, tools/protoc-gen-go-triple, tools/protoc-gen-triple-openapi, tools/imports-formatter,
  tools/variadicrpccheck, generated Triple code, OpenAPI generation, schema output, import formatting, RPC contract
  scanning, Makefile targets, or nested tool modules under tools/. Do not use for runtime protocol behavior unless the
  generator output contract is the issue.
---

# dubbo-go Tools

## Purpose

Use this skill to change or explain dubbo-go developer tooling, generators, schema helpers, CLI behavior, import formatting, and contract scanning.

## When to use

Use for `tools/` changes, generated code templates, protoc plugins, schema generation, CLI commands, Makefile tool targets, and nested tool module validation.

Do not use for runtime protocol behavior unless the generated code contract is the changed behavior.

## Inputs

Required:
- Tool name, command, generated file, template, or package path.
- Intended tool behavior or observed failure.

Optional:
- CLI arguments, proto input, generated output diff, schema sample, or build/test output.

If missing, inspect the specific tool README and its local `go.mod` when present.

## Workflow

1. Identify whether the change affects a root-module tool or a nested Go module under `tools/`.
2. Read the tool README and command entrypoint.
3. Inspect templates or generated output contracts before changing generator code.
4. For Makefile behavior, read the root `Makefile` and any nested module commands.
5. For generated runtime code, coordinate with `dubbo-go-protocol` only when wire or invocation semantics change.
6. Read `references/tooling-flow.md` before changing generators or validation commands.

## Output format

Return tool entrypoint, input and output contract, changed files, validation commands, and any runtime compatibility impact.

## Validation

- Run tests from the owning module directory when the tool has its own `go.mod`.
- Run root commands for root-module tools, such as `go test ./tools/variadicrpccheck` or `make rpc-contract-check`.
- Confirm generated output is deterministic.
- Run `make fmt` or the tool-specific formatter only when code changes require it.

## Edge cases

- Several `tools/` subdirectories are nested Go modules and should be tested from their own directory.
- Generated code can affect public APIs even when the generator change is small.
- `imports-formatter` is part of `make fmt`; avoid changing import output without checking repository formatting expectations.

## References

- Read `references/tooling-flow.md` for tool modules, entrypoints, and validation details.
