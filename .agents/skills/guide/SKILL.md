---
name: dubbo-go-guide
description: Use when explaining dubbo-go concepts, architecture, current v3 APIs, protocols, registries, routers, filters, OpenAPI, graceful shutdown, observability, samples, or best practices.
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


# Guiding dubbo-go Development

## Overview

Develop-branch map for dubbo-go v3: code API first, Triple as the default wire protocol, application-level discovery as the default registration mode. Reach for this skill when the user asks "how does X work" or "which sample should I follow"; switch to `dubbo-go-scaffolding` when generating an app, `dubbo-go-extensions` when adding an SPI, or `dubbo-go-debugging` when troubleshooting a runtime failure.

## When to Use

- Conceptual questions: instance/server/client/protocol/registry/cluster/router/filter
- Choosing between Triple, Dubbo, JSONRPC, REST, gRPC-compatible Triple, HTTP/3
- Pointing the user to the right `dubbo-go-samples` directory
- Explaining OpenAPI, CORS, AttachHTTPHandler, graceful shutdown, observability defaults

## Project Snapshot

- Module: `dubbo.apache.org/dubbo-go/v3`
- Go version in `go.mod`: `1.25.0`
- Main entry points: `dubbo.NewInstance`, `server.NewServer`, `client.NewClient`
- Preferred user style: code API
- Legacy/compat style: YAML-driven `dubbo.Load()` / `config.Load()` still exists and now uses koanf-based loading
- Preferred wire protocol: Triple
- Important recent capabilities: Triple OpenAPI, HTTP/3, CORS, attached HTTP handlers, refined graceful shutdown, logger trace integration, router static config, `variadicrpccheck`

## Repository Map

- `client/`: consumer/reference options, URL processing, invoker construction
- `server/`: provider/server options, service registration, `AttachHTTPHandler`
- `protocol/`: protocol abstractions and implementations; `protocol/triple` is the primary path
- `registry/`: registry and application-level service discovery implementations
- `metadata/`: metadata report and service-to-application mapping
- `cluster/`: cluster strategies, router chain, load balancing
- `filter/`: provider and consumer filter implementations
- `config/`, `global/`: YAML/config structs and runtime config models
- `config_center/`: Nacos, ZooKeeper, Apollo, and file dynamic config
- `graceful_shutdown/`: shutdown options and runtime state
- `metrics/`, `otel/`, `logger/`: observability integrations
- `tools/`: CLI, schema generator, Triple/OpenAPI generators, imports formatter, RPC contract scanner

## Core Concepts

**Instance**: global application context. Use `dubbo.NewInstance` when sharing application, registry, protocol, logger, shutdown, metrics, tracing, TLS, or router settings across clients and servers.

**Server**: owns provider-side config and service exports. Register generated Triple handlers with `pb.RegisterXxxHandler`.

**Client**: owns consumer-side config. Create generated service clients with `pb.NewXxxService`.

**Protocol**: wire transport. Triple is HTTP/2 and gRPC-compatible; Dubbo is for legacy Hessian2 interop; JSONRPC and REST also exist; HTTP/3 is experimental under Triple.

**Registry**: service discovery. Built-ins include Nacos, ZooKeeper, etcd v3, and Polaris.

**Metadata mapping**: application-level discovery needs a mapping from interface to provider application. If metadata mapping is unavailable, use `client.WithProvidedBy("provider-app")`.

**Filter**: interceptor chain around RPC calls. Filters run on provider and consumer paths and are used for auth, metrics, tracing, rate limiting, graceful shutdown, logging, and generic calls.

**Cluster**: fault-tolerance strategy such as Failover, Failfast, Failsafe, Failback, Available, Broadcast, Forking, ZoneAware, and AdaptiveService.

**LoadBalance**: provider selection. Built-ins include Random, RoundRobin, LeastActive, ConsistentHashing, P2C, and adaptive strategies.

**Router**: filters providers before load balancing. Built-ins include tag, condition, script, affinity, and Polaris routers. Static router config can be supplied through client/root router options; dynamic rules can come from config centers.

## Current API Patterns

Direct server:

```go
srv, err := server.NewServer(
    server.WithServerProtocol(
        protocol.WithPort(20000),
        protocol.WithTriple(),
    ),
)
```

Registry-backed instance:

```go
ins, err := dubbo.NewInstance(
    dubbo.WithName("orders-provider"),
    dubbo.WithRegistry(registry.WithNacos(), registry.WithAddress("127.0.0.1:8848")),
    dubbo.WithProtocol(protocol.WithTriple(), protocol.WithPort(20000)),
)
```

Reference options:

```go
svc, err := pb.NewOrderService(
    cli,
    client.WithInterface("shop.OrderService"),
    client.WithProvidedBy("orders-provider"),
    client.WithRequestTimeout(3*time.Second),
)
```

Service options:

```go
err := pb.RegisterOrderServiceHandler(
    srv,
    impl,
    server.WithInterface("shop.OrderService"),
    server.WithGroup("beta"),
    server.WithVersion("v2"),
    server.WithOpenAPIGroup("orders-v2"),
)
```

## Extension Pattern

Most extension points follow:

1. Implement the interface.
2. Register a factory in `init()` through `common/extension`.
3. Blank-import the package so `init()` runs.
4. Activate by name through code API or config.

Examples:

- Filter: `extension.SetFilter("name", func() filter.Filter { ... })`
- LoadBalance: `extension.SetLoadbalance("name", func() loadbalance.LoadBalance { ... })`
- Registry: `extension.SetRegistry("name", func(*common.URL) (registry.Registry, error) { ... })`
- Protocol: `extension.SetProtocol("name", func() base.Protocol { ... })`
- Router: `extension.SetRouterFactory("name", func() router.PriorityRouterFactory { ... })`

The string name is the contract. Keep registration and activation names identical.

## Triple Extras

OpenAPI:

```go
protocol.WithTriple(
    triple.WithOpenAPI(
        triple.OpenAPIEnable(),
        triple.OpenAPIPath("/dubbo/openapi"),
    ),
)
```

CORS:

```go
protocol.WithTriple(
    triple.WithCORS(
        triple.CORSAllowOrigins("https://example.com"),
        triple.CORSAllowMethods("POST", "GET", "OPTIONS"),
    ),
)
```

HTTP/3:

```go
protocol.WithTriple(triple.Http3Enable())
```

Attached HTTP handler:

```go
_ = srv.AttachHTTPHandler(mux)
```

Attach before `Serve`, use one root handler, and configure an explicit Triple port.

## Samples

See [samples-index.md](samples-index.md) for the current dubbo-go-samples map.

Start with:

- `helloworld`: minimal direct Triple
- `direct`: direct URL connection
- `registry/nacos`: Nacos service discovery
- `rpc/triple/openapi`: runtime Triple OpenAPI
- `http3`: experimental HTTP/3
- `graceful_shutdown`: shutdown flow
- `logger/trace-integration`: trace IDs in logs

## Best Practices

- Prefer Protobuf plus Triple for new services.
- Use `_ "dubbo.apache.org/dubbo-go/v3/imports"` for examples and local demos; use selective blank imports in production when binary size matters.
- Use code API for new code; keep YAML only for existing config-driven applications.
- Use `client.WithProvidedBy` when application-level discovery cannot resolve interface-to-application mapping.
- Do not assume viper APIs in config work; current loading is koanf-centered.
- Run repository validation commands from the development skill before finishing code changes.

## Related Skills

- `dubbo-go-scaffolding` - when the user wants to generate a provider/consumer instead of read about it
- `dubbo-go-extensions` - when explaining a built-in is not enough and the user needs a custom SPI
- `dubbo-go-java-interop` - when the question involves dubbo-java compatibility
- `dubbo-go-debugging` - when the question is "why is X failing" rather than "how does X work"
- `dubbo-go-migration` - when the question is "how do I move from Y to dubbo-go"
- `dubbo-go-development` - when the explanation must dig into the `apache/dubbo-go` repo internals
