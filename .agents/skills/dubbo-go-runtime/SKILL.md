---
name: dubbo-go-runtime
description: >-
  Implements and reviews dubbo-go public client and server lifecycle changes. Use when the user asks about
  dubbo.NewInstance, Instance.NewClient, Instance.NewServer, dubbo.Load runtime startup, client.NewClient,
  server.NewServer, ReferenceOptions.Refer, ServiceOptions.Export, service registration, proxy creation, graceful
  shutdown, direct URLs, mesh mode, TLS injection, or packages dubbo.go, client/, server/, proxy/, and graceful_shutdown/.
  Do not use for low-level codec or registry implementation unless export/refer lifecycle owns the contract.
---

# dubbo-go Runtime

## Purpose

Use this skill to change or explain the public lifecycle from configuration and options to client reference creation, provider export, proxy generation, and shutdown.

## When to use

Use for public APIs, client/server options, reference/service initialization, export and refer behavior, direct URL mode, mesh mode, provider registration, consumer proxy creation, and graceful shutdown.

Do not use for low-level protocol codecs, registry internals, or metadata storage unless runtime code controls the contract.

## Inputs

Required:
- Client, server, reference, service, proxy, or lifecycle behavior.
- Intended public API behavior or observed runtime bug.

Optional:
- Config snippet, service interface, generated client/server code, URL, or stack trace.

If missing, inspect `dubbo.go`, then the matching path in `client/` or `server/`.

## Workflow

1. Start at the public entry point: `dubbo.go`, `client/client.go`, `server/server.go`, or `loader.go`.
2. For consumer behavior, read `client/options.go` and `client/action.go`.
3. For provider behavior, read `server/options.go`, `server/action.go`, and `server/server.go`.
4. Trace URL construction and attributes before following into protocol, registry, metadata, or filters.
5. Trace proxy behavior through `proxy/` and `proxy/proxy_factory/`.
6. Check shutdown and destroy behavior in `graceful_shutdown/` and protocol `Destroy` methods.
7. Read `references/client-server-lifecycle.md` before changing export or refer flow.

## Output format

Return the public API impact, reference or service lifecycle path, URL or proxy contract, changed files, and validation performed.

## Validation

- Confirm user-provided options override cloned config when that is the existing contract.
- Confirm service/reference URLs carry required params and attributes.
- Confirm export/refer paths register shutdown and destroy the right protocols or registries.
- Run focused tests such as `go test ./client ./server ./proxy ./graceful_shutdown`; use `make test` for shared lifecycle behavior.

## Edge cases

- `ReferenceOptions.Refer` may use direct URLs or registry URLs.
- Mesh mode rewrites the reference URL and requires Triple.
- Provider export may use multiple protocol configs and random ports.
- Non-IDL and IDL Triple paths use different attributes and generated metadata.

## References

- Read `references/client-server-lifecycle.md` for client, server, export, refer, and shutdown flow.
