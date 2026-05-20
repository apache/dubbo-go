# dubbo-go Project Rules

## Compatibility

- Treat public Go APIs, config keys, URL params, URL attributes, protocol wire behavior, generated code shape, and registry or metadata semantics as compatibility surfaces.
- Document behavior and compatibility impact when changing any compatibility surface.
- Preserve Dubbo Java, Triple, and gRPC interoperability when touching protocol or generated-code paths.

## Generated Files

- Do not modify generated files manually unless the generation source is also updated or the file is intentionally hand-maintained in this repository.
- For proto-related output, inspect the owning tool under `tools/protoc-gen-go-triple` or `tools/protoc-gen-triple-openapi`.
- Keep generated output deterministic: stable ordering, imports, names, and comments.

## Validation Selection

- Start with the narrowest package tests that cover the change.
- Use `make test` for cross-cutting runtime behavior.
- Use `make fmt` after Go code changes that affect formatting or imports.
- Use `make lint` for changes that touch shared APIs, concurrency, generated code, or error handling.
- Use `make rpc-contract-check` when changing exported RPC service interfaces.

## Nested Modules

Run tests from the nested module directory when a tool has its own `go.mod`:

- `tools/dubbogo-cli`
- `tools/imports-formatter`
- `tools/protoc-gen-go-triple`
- `tools/protoc-gen-triple-openapi`

## PR Guidance

- Keep PRs small and reviewable.
- Include tests or explain why tests are not feasible.
- Include compatibility notes for public behavior changes.
- Do not include unrelated formatting churn.
