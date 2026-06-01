---
name: dubbo-go-debugging
description: Structured diagnosis for dubbo-go v3 runtime errors. Use when the user reports an error, pastes logs, mentions timeout/panic/connection refused/no provider/serialization mismatch, or asks why their dubbo-go service isn't working.
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

# Debugging dubbo-go

When the user shares an error or log, match it to a pattern below. If unclear, ask for:

1. Full error message or stack trace
2. Registry type (Nacos / ZooKeeper / etcd / Polaris / direct)
3. Protocol (Triple / Dubbo / gRPC / JSONRPC)
4. Whether provider and consumer are separate processes

## Quick Triage

| Symptom in logs | Jump to |
|---|---|
| `no provider available`, `no route`, `Should has at least one way to know` | [No provider / no route](#no-provider--no-route) |
| `dial tcp ... connection refused`, `i/o timeout` on connect | [Connection refused / dial error](#connection-refused--dial-error) |
| `hessian: failed to decode`, `protobuf: cannot parse invalid wire-format data` | [Serialization / decode error](#serialization--decode-error) |
| `context deadline exceeded`, `invoke timeout` | [Timeout](#timeout) |
| `filter not found`, panic inside a filter | [Filter not found / filter panic](#filter-not-found--filter-panic) |
| Provider exits seconds after start | [Provider starts but immediately exits](#provider-starts-but-immediately-exits) |
| `404` at `/dubbo/openapi/...`, empty spec | [OpenAPI 404 / empty spec](#openapi-404--empty-spec) |
| `AttachHTTPHandler` returns error | [AttachHTTPHandler errors](#attachhttphandler-errors) |
| Pods drain slowly, in-flight requests truncated on shutdown | [Graceful shutdown](#graceful-shutdown) |

## No provider / no route

**Cause**: Consumer cannot find a provider in the registry.

Checklist:
- [ ] Provider started successfully? Look for `dubbo server started` and `A provider service ... was registered successfully` in provider logs.
- [ ] Same registry address on both sides (`dubbo.WithRegistry(registry.WithAddress(...))` / YAML `dubbo.registries.xxx.address`)?
- [ ] Same application name on both sides (`dubbo.WithName(...)`)? v3 defaults to **application-level** discovery — the registry stores the app name, not the interface FQN.
- [ ] Same service discovery level on both sides? Latest dubbo-go defaults to **application-level** discovery, while older dubbo-go versions commonly defaulted to **interface-level** discovery; application-level and interface-level discovery do not interoperate.
- [ ] Same `interface` name passed to `pb.RegisterXxxHandler` and `pb.NewXxxService`?
- [ ] Same protocol on both sides (both `tri` or both `dubbo`)?
- [ ] Provider visible in the registry?

```bash
# Nacos — application-level discovery, query by app name
curl "http://127.0.0.1:8848/nacos/v1/ns/instance/list?serviceName=<your-app-name>"

# ZooKeeper
zkCli.sh ls /services

# etcd
etcdctl get --prefix /services
```

If the consumer cannot map an interface to an application, hint with `client.WithProvidedBy("<app-name>")`.

## Connection refused / dial error

**Cause**: Network or port misconfiguration.

Checklist:
- [ ] Provider port (`protocol.WithPort(...)`) matches what the consumer is dialing?
- [ ] Firewall / Docker network allows the port?
- [ ] Inside Docker: using container/service name instead of `localhost`?
- [ ] Provider actually listening? `lsof -i :20000` on the provider host.

## Serialization / decode error

```
hessian: failed to decode
protobuf: cannot parse invalid wire-format data
```

**Cause**: Provider and consumer using different serialization formats.

Checklist:
- [ ] Both sides on the same protocol (both `tri` or both `dubbo`)?
- [ ] Protobuf: same `.proto` compiled on both sides? Same `go_package`?
- [ ] Hessian2: POJO `JavaClassName()` matches the Java class FQN exactly? `RegisterPOJO` called in an `init()` that actually runs?

For Hessian2 specifics, see [dubbo-go-java-interop](../java-interop/SKILL.md).

## Timeout

```
context deadline exceeded
invoke timeout
```

**Cause**: Provider too slow, filter blocking, network delay, or low timeout.

Checklist:
- [ ] Provider actually receiving the request? Add a log line at the top of the handler.
- [ ] Any filter that may block? (auth, token, tps-limit)
- [ ] Timeout high enough?

```go
// per-reference
svc, _ := pb.NewGreetService(cli, client.WithRequestTimeout(10*time.Second))

// per-call
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
resp, err := svc.Greet(ctx, req)
```

## Filter not found / filter panic

**Cause**: Filter imported but `init()` not registered, or name mismatch between `extension.SetFilter` and `WithFilter`.

Checklist:
- [ ] Blank import present? e.g. `_ "dubbo.apache.org/dubbo-go/v3/filter/token"`, `_ "github.com/apache/dubbo-go-extensions/filter/hystrix"`, or `_ "github.com/yourorg/yourapp/filter/myfilter"`
- [ ] String passed to `server.WithFilter("xxx")` / `client.WithFilter("xxx")` matches the name in `extension.SetFilter("xxx", ...)`?
- [ ] Built-ins not pulled in? Use `_ "dubbo.apache.org/dubbo-go/v3/imports"` during development to auto-import all built-ins.

## Provider starts but immediately exits

**Cause**: Missing blocking call or wrong shutdown setup.

```go
// Correct: blocking
if err := srv.Serve(); err != nil { panic(err) }

// Wrong: non-blocking, process exits immediately
go srv.Serve()
```

## OpenAPI 404 / empty spec

Triple-only feature. Checklist:
- [ ] `triple.OpenAPIEnable(true)` set in the protocol options?
- [ ] Hitting the right route? Default is `/dubbo/openapi/openapi.json`. See `triple.OpenAPIPath(...)` to customize.
- [ ] Visiting the right port? OpenAPI lives on the Triple port, not a separate one.

## AttachHTTPHandler errors

Checklist:
- [ ] Protocol is Triple? Triple is the only protocol that hosts plain HTTP handlers.
- [ ] Called *before* `srv.Serve()`?
- [ ] Path conflicts with the Triple-reserved namespace (`/dubbo/...`)?

## Graceful shutdown

Checklist:
- [ ] Built-in graceful shutdown filters registered? `_ "dubbo.apache.org/dubbo-go/v3/imports"` includes them; with selective imports, add `_ "dubbo.apache.org/dubbo-go/v3/filter/graceful_shutdown"` so `pshutdown` / `cshutdown` are registered.
- [ ] Shutdown timeout long enough for in-flight calls to drain?
- [ ] If running behind Kubernetes: container `terminationGracePeriodSeconds` longer than dubbo-go's shutdown timeout?

## Reading dubbo-go logs

Enable debug logging:

```go
import "dubbo.apache.org/dubbo-go/v3/logger"

ins, _ := dubbo.NewInstance(
    dubbo.WithLogger(logger.WithLevel("debug")),
)
```

Inject trace IDs into logs (when OpenTelemetry tracing is configured):

```go
dubbo.WithLogger(logger.WithTraceIntegration(true))
```

Key log lines:
- `Registering service: <interface>` — service registration starting
- `A provider service ... was registered successfully` — provider ready
- `export <interface> service failed` — registration error
- `No provider available` / `no route` — consumer lookup failed

## Related Skills

- `dubbo-go-extensions` — when a missing/registered SPI is the root cause
- [dubbo-go-java-interop](../java-interop/SKILL.md) — for Hessian2 decode failures and cross-language discovery
- `dubbo-go-guide` — for the conceptual map (Instance / Server / Client / Protocol / Registry / Filter)
