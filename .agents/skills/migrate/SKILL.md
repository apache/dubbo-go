---
name: dubbo-go-migration
description: Use when migrating to dubbo-go v3 from gRPC-Go, Spring Cloud, Gin, plain HTTP, older dubbo-go versions, YAML-based apps, Java Dubbo, or legacy Hessian2 services.
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


# Migrating to dubbo-go

## Overview

Migration is mostly mechanical: the source framework decides the concept-mapping table, then dubbo-go v3 code-API patterns slot in. Triple + Protobuf is the default target; Dubbo + Hessian2 is reserved for Java-interface services without a `.proto`.

## Source-System Decision Table

| User says | Jump to |
|---|---|
| "We're on gRPC-Go" | [From gRPC-Go](#from-grpc-go) |
| "Spring Cloud / Java Dubbo" | [From Spring Cloud or Java Dubbo](#from-spring-cloud-or-java-dubbo) |
| "Gin / chi / net/http" | [From Gin or Plain HTTP](#from-gin-or-plain-http) |
| "Old dubbo-go v1/v2 or YAML-heavy v3" | [From dubbo-go v1/v2 or Old v3 YAML](#from-dubbo-go-v1v2-or-old-v3-yaml) |
| "Java owns the contract, no proto" | [From Java-interface Hessian2](#from-java-interface-hessian2) |

Use code API for new work and keep YAML only when preserving an existing deployment model.

## From gRPC-Go

Concept mapping:

| gRPC-Go | dubbo-go v3 |
|---|---|
| `grpc.NewServer()` | `server.NewServer()` |
| `pb.RegisterXxxServer(srv, impl)` | `pb.RegisterXxxHandler(srv, impl)` |
| `grpc.Dial` / `grpc.NewClient` | `client.NewClient()` + `pb.NewXxxService(cli)` |
| Unary interceptor | Filter |
| Service config / resolver | Registry + cluster + load balance |
| Reflection | Triple reflection sample and OpenAPI support |

Protobuf can usually be reused. Generate Triple bindings:

```bash
go install github.com/dubbogo/protoc-gen-go-triple/v3@latest
protoc --go_out=. --go_opt=paths=source_relative \
  --go-triple_out=. --go-triple_opt=paths=source_relative \
  proto/*.proto
```

Direct-mode consumer:

```go
cli, err := client.NewClient(
    client.WithClientURL("127.0.0.1:20000"),
)
svc, err := pb.NewGreetService(cli)
```

Add a registry later with `dubbo.NewInstance` and `dubbo.WithRegistry`.

## From Spring Cloud or Java Dubbo

Concept mapping:

| Spring Cloud / Dubbo Java | dubbo-go v3 |
|---|---|
| Discovery client | `dubbo.WithRegistry(...)` |
| Feign/Dubbo reference | generated `pb.NewXxxService(cli, ...)` |
| Controller or Dubbo provider | `pb.RegisterXxxHandler(srv, impl, ...)` |
| Hystrix / Sentinel | Hystrix or Sentinel filters |
| Sleuth / tracing | OpenTelemetry integration |
| Actuator health | `metrics/probe` sample |
| Java interface + POJO | Dubbo protocol + Hessian2 |

Migration path:

1. Keep existing Java services running.
2. Use Triple + Protobuf for new Go services where possible.
3. Use Dubbo + Hessian2 only when the Java side has no proto contract.
4. Share registry and metadata mapping. If mapping is not available, use `client.WithProvidedBy`.
5. During transition, use `registry.WithRegisterServiceAndInterface()` when both app-level and interface-level discovery are needed.

## From Gin or Plain HTTP

dubbo-go is primarily an RPC and service governance framework, but current develop provides three practical migration options.

Option A: keep HTTP externally, add dubbo-go internally.

- Gin/http handles public REST.
- dubbo-go handles service-to-service RPC.
- Run both in one process if lifecycle is clear.

Option B: attach existing HTTP routes to the Triple listener.

```go
mux := http.NewServeMux()
mux.HandleFunc("/healthz", healthz)

srv, _ := server.NewServer(
    server.WithServerProtocol(
        protocol.WithTriple(),
        protocol.WithIp("127.0.0.1"),
        protocol.WithPort(20000),
    ),
)
_ = srv.AttachHTTPHandler(mux)
```

Rules: attach before `Serve`, attach one root handler, and use an explicit Triple port.

Option C: move APIs to Protobuf + Triple and expose JSON-over-HTTP plus OpenAPI.

```go
server.WithServerProtocol(
    protocol.WithPort(20000),
    protocol.WithTriple(
        triple.WithOpenAPI(triple.OpenAPIEnable()),
    ),
)
```

OpenAPI endpoints default to `/dubbo/openapi`.

## From dubbo-go v1/v2 or Old v3 YAML

Important changes:

| Old style | Current v3 develop |
|---|---|
| Mostly YAML/config globals | Code API preferred |
| `config.Load()` as main entry | `dubbo.NewInstance()` for new apps; `dubbo.Load()`/`config.Load()` for compatibility |
| Interface-based Hessian2 default | Protobuf + Triple preferred |
| Global provider/consumer service registration | Generated `RegisterXxxHandler` and `NewXxxService` helpers |
| viper assumptions | koanf-based loading in current config path |
| Older `protocol` aliases | Prefer `protocol/base` and `protocol/result` packages in new extension code |

Migration sequence:

1. Move service contracts to `.proto` when possible.
2. Generate `*.pb.go` and `*.triple.go`.
3. Replace global registration with generated server/client helpers.
4. Move registry/protocol/application config to `dubbo.NewInstance`.
5. Keep YAML only for externally managed config and test it with `config_yaml` patterns.
6. Run `make rpc-contract-check` if exposed RPC methods use variadic parameters.

## From Java-interface Hessian2

If Java owns the contract and there is no `.proto`, do not invent a partial proto. Mirror Java request/response POJOs in Go, implement `JavaClassName()`, register them with `dubbo-go-hessian2`, and use Dubbo protocol.

See the Java interop skill for exact POJO and protocol rules.

## Validation

For application migration:

```bash
go mod tidy
go test ./...
```

For changes inside the dubbo-go repository:

```bash
make fmt
GOTOOLCHAIN=go1.25.0+auto go test ./...
make rpc-contract-check
```

Use targeted package tests first when the migration touches only one area.

## Related Skills

- `dubbo-go-scaffolding` - for the target-side provider/consumer skeleton after the source-side mapping is decided
- `dubbo-go-java-interop` - when the migration crosses the Java/Go boundary (Hessian2 POJO rules live there)
- `dubbo-go-extensions` - when porting a custom filter/LB/registry from gRPC-Go interceptors or Spring Cloud beans
- `dubbo-go-debugging` - when the migrated service starts but does not register, route, or decode
