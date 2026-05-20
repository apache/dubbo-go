---
name: dubbo-go-protocol
description: >-
  Implements and reviews dubbo-go protocol and transport changes. Use when the user asks about Protocol, Invoker,
  Exporter, Invocation, Result, common.URL, protocol wrapper behavior, Dubbo, Dubbo3, Triple, gRPC, REST, JSONRPC,
  HTTP or HTTP/3 transport, codecs, remoting clients or servers, or packages protocol/, remoting/, common/url.go, and
  protocol/protocolwrapper/. Do not use for registry subscription or cluster routing unless protocol invoker semantics
  are the changed contract.
---

# dubbo-go Protocol

## Purpose

Use this skill to change or explain transport behavior from URL construction through protocol export, refer, invocation, codec, remoting, result mapping, and destroy semantics.

## When to use

Use for `base.Protocol`, `base.Invoker`, `base.Exporter`, protocol implementations, codecs, remoting clients and servers, Triple/gRPC/Dubbo/REST/JSONRPC behavior, and URL param or attribute semantics.

Do not use for registry discovery, metadata storage, or cluster selection unless the protocol invoker contract is the changed behavior.

## Inputs

Required:
- Protocol name, transport package path, invocation behavior, or URL shape.
- Intended wire behavior or observed protocol bug.

Optional:
- Request or response payload, codec frame, headers, attachments, service info, or interoperability target.

If missing, inspect `protocol/base/`, `protocol/protocolwrapper/`, and the concrete protocol package.

## Workflow

1. Read `protocol/base/base_protocol.go`, `base_invoker.go`, `base_exporter.go`, and invocation/result contracts.
2. Check how the protocol is registered through `common/extension/protocol.go`.
3. Trace filter wrapping through `protocol/protocolwrapper/protocol_filter_wrapper.go`.
4. Inspect the concrete protocol implementation under `protocol/{dubbo,dubbo3,triple,grpc,rest,jsonrpc}`.
5. For network behavior, follow into `remoting/` or protocol-specific server/client packages.
6. Verify URL params versus attributes before changing cross-package contracts.
7. Read `references/protocol-flow.md` before changing export, refer, codec, or destroy semantics.

## Output format

Return protocol path, URL contract, export or refer flow, invocation/result behavior, changed files, and validation performed.

## Validation

- Confirm concrete protocols register with the expected extension name.
- Confirm `Export`, `Refer`, `Invoke`, and `Destroy` semantics remain compatible with wrappers and graceful shutdown.
- Confirm wire changes preserve Dubbo Java or gRPC interoperability when relevant.
- Run focused tests for the changed protocol such as `go test ./protocol/triple/...`; use `go test ./protocol/... ./remoting/...` or `make test` for broad transport changes.

## Edge cases

- `protocolwrapper.FILTER` is registered as a protocol and delegates to the concrete protocol named in the URL.
- `common.URL` params are transported; attributes are process-local and should not be serialized.
- Destroyed invokers return a safe internal URL placeholder.
- Triple IDL, Triple non-IDL, and Dubbo interoperability paths can differ.

## References

- Read `references/protocol-flow.md` for export, refer, wrapper, URL, and destroy details.
