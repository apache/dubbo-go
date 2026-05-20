# Tooling Flow

## Tool Areas

- `tools/dubbogo-cli`: CLI for bootstrapping and managing dubbo-go applications. It has its own Go module.
- `tools/dubbo-go-schema`: schema support for dubbo-go configuration files.
- `tools/protoc-gen-go-triple`: protoc plugin for Triple client and server code. It has its own Go module.
- `tools/protoc-gen-triple-openapi`: OpenAPI generator for Triple. It has its own Go module.
- `tools/imports-formatter`: import formatting helper used by `make fmt`. It has its own Go module.
- `tools/variadicrpccheck`: root-module scanner for exported variadic RPC contracts.

## Root Make Targets

- `make test`: runs root tests and `tools/dubbogo-cli` tests.
- `make fmt`: runs modernize, `go fmt`, `imports-formatter`, and CLI formatting.
- `make lint`: runs `go vet` and `golangci-lint`.
- `make rpc-contract-check`: runs the variadic RPC contract scanner.

## Generator Checks

1. Identify the source input format, usually proto files or config models.
2. Inspect the generated output expected by tests.
3. Keep output deterministic: stable ordering, stable import grouping, stable comments.
4. Run tests in the nested module that owns the generator.
5. When generated runtime code changes, run relevant protocol or server/client tests in the root module.

## Nested Module Rule

When a tool has its own `go.mod`, run `go test ./...` from that tool directory. Do not assume root `go test ./...` covers it.
