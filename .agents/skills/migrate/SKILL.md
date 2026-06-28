---
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
name: dubbo-go-migration
description: Guides migration to dubbo-go v3 from gRPC-Go, Spring Cloud, Gin / plain HTTP, dubbo-go v1/v2, YAML-heavy v3 apps, and Java-interface Hessian2 services. Use when the user asks how to migrate, port, or upgrade an existing service to dubbo-go v3.
---

# Migrating to dubbo-go

First question: **where are you migrating from?**

| User says | Jump to |
|---|---|
| "We're on gRPC-Go" | [From gRPC-Go](#from-grpc-go) |
| "Spring Cloud / Java microservices" | [From Spring Cloud](#from-spring-cloud-java) |
| "Gin / chi / net/http" | [From Gin or plain HTTP](#from-gin--plain-http) |
| "dubbo-go v1 or v2" | [From dubbo-go v1/v2 → v3](#from-dubbo-go-v1v2--v3) |
| "YAML-heavy v3 app" | [From YAML-heavy v3](#from-yaml-heavy-v3) |

For Java-interface Hessian2 services, see `dubbo-go-java-interop`.

Use the code API for new work; keep YAML only when preserving an existing deployment model.

## From gRPC-Go

**Concept mapping**

| gRPC-Go | dubbo-go |
|---|---|
| `grpc.NewServer()` | `server.NewServer()` (or `ins.NewServer()` with `dubbo.NewInstance`) |
| `pb.RegisterXxxServer(srv, impl)` | `pb.RegisterXxxHandler(srv, impl)` (Triple) |
| `grpc.Dial(addr)` | `client.NewClient()` → `pb.NewXxxService(cli)` |
| `UnaryServerInterceptor` | Filter |
| `grpc.WithUnaryInterceptor(...)` | `server.WithFilter("my-filter")` + blank import |
| Service reflection | Built-in with Triple |

**Proto file**: no changes needed. Triple is wire-compatible with gRPC.

**Key difference**: dubbo-go adds service discovery on top of gRPC. For direct connection, use direct mode — no registry needed.

```go
// gRPC-Go
conn, _ := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
client := pb.NewGreetClient(conn)

// dubbo-go (direct mode)
cli, _ := client.NewClient(client.WithClientURL("127.0.0.1:20000"))
svc, _ := pb.NewGreetService(cli)
```

See [rpc/grpc](https://github.com/apache/dubbo-go-samples/tree/main/rpc/grpc) and [direct](https://github.com/apache/dubbo-go-samples/tree/main/direct).

## From Spring Cloud (Java)

**Concept mapping**

| Spring Cloud | dubbo-go |
|---|---|
| `@EnableDiscoveryClient` | `dubbo.WithRegistry(registry.WithNacos(), registry.WithAddress(...))` |
| `@FeignClient` | `client.NewClient()` + generated `pb.NewXxxService(cli)` |
| `@RestController` | `pb.RegisterXxxHandler(srv, impl)` + Triple/REST |
| Spring Cloud Gateway | Apache APISIX or direct |
| Hystrix / Resilience4j | Hystrix filter / Sentinel filter |
| Sleuth / Zipkin | OpenTelemetry filter |
| Actuator `/health` | `metrics/probe` (`/live`, `/ready`) |

**Migration path**:
1. Keep Java services running.
2. Add dubbo-go providers for new Go services using Triple + Protobuf.
3. Java consumers can call Go providers via Triple (wire-compatible).
4. Migrate Java consumers to Go one by one.

See [java_interop](https://github.com/apache/dubbo-go-samples/tree/main/java_interop) and `dubbo-go-java-interop`.

## From Gin / plain HTTP

**Concept mapping**

| Gin / HTTP | dubbo-go |
|---|---|
| `gin.Default()` + `r.POST(...)` | `server.NewServer()` + `pb.RegisterXxxHandler(...)` |
| `http.Get(url)` / `http.Client` | `client.NewClient()` + `pb.NewXxxService(cli)` |
| Gin middleware | Filter |
| `r.Run(":8080")` | `srv.Serve()` |
| Route handler func | Method on a service struct |

dubbo-go is an RPC framework, not an HTTP framework. Two coexistence options:

**Option A — Keep Gin for HTTP, add dubbo-go for internal RPC**. No conflict; run both in the same process.

**Option B — Mount the HTTP handler on the Triple port**:

```go
srv.AttachHTTPHandler("/api", r) // r is your Gin router
```

**Option C — Use Triple's OpenAPI / REST mode** to expose RPC methods as HTTP/JSON:

```go
server.WithServerProtocol(
    protocol.WithTriple(triple.OpenAPIEnable(true)),
)
```

```bash
# Triple supports JSON over HTTP/1.1 — call any method without a generated client
curl -H "Content-Type: application/json" \
     -d '{"name":"world"}' \
     http://localhost:20000/greet.GreetService/Greet
```

See [rpc/triple/openapi](https://github.com/apache/dubbo-go-samples/tree/main/rpc/triple/openapi).

## From dubbo-go v1/v2 → v3

**Breaking-change summary**

| v1/v2 | v3 |
|---|---|
| `config.Load()` | `dubbo.NewInstance()` (code API, recommended) or `dubbo.Load()` (YAML, legacy) |
| `hessian.RegisterPOJO()` + Java-interface | Protobuf + generated Triple stubs (recommended) |
| `config.SetProviderService()` | `pb.RegisterGreetServiceHandler(srv, impl)` |
| `config.ConsumerConfig.References` | `client.NewClient()` → `pb.NewGreetService(cli)` |
| `getty` as default transport | Triple (HTTP/2) as default |
| Interface-level discovery | Application-level discovery |

**Recommended path**: migrate to the code API. YAML via `dubbo.Load()` still works, but the YAML key structure is not guaranteed identical across versions — expect to rewrite `registries`, `protocols`, and service registration.

`getty`-style v1/v2 Dubbo+Hessian2 services can keep working with `protocol.WithDubbo()` while interfaces are rebuilt as `.proto` for Triple. See `dubbo-go-java-interop` for the Hessian2 POJO bridge.

## From YAML-heavy v3

You don't have to abandon YAML. `dubbo.Load()` is still supported. Migration to code API is incremental:

1. Move shared config (registries, protocols) into `dubbo.NewInstance(...)`.
2. Replace `dubbo.Load()` with `ins.NewServer()` / `ins.NewClient()` per service.
3. Keep YAML for environment-specific overrides via the config center.

See [config_yaml](https://github.com/apache/dubbo-go-samples/tree/main/config_yaml).

## Validation

```bash
go mod tidy
go build ./...
go test ./...
```

## Reference

Official guide: [dubbo-go v3 user manual](https://cn.dubbo.apache.org/zh-cn/overview/mannual/golang-sdk/) (also available in English on the same site).

## Related Skills

- `dubbo-go-scaffolding` — for the target-side provider/consumer skeleton
- `dubbo-go-java-interop` — when the migration crosses the Java/Go boundary
- `dubbo-go-extensions` — when porting custom interceptors / filters
- `dubbo-go-debugging` — when the migrated service starts but does not register, route, or decode
