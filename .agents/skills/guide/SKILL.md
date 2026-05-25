---
name: dubbo-go-guide
description: Explains dubbo-go v3 architecture, extension points, and best practices. Use when the user asks how dubbo-go works, what a concept means (Instance, Protocol, Registry, Filter, Cluster, LoadBalance, Router, OpenAPI, graceful shutdown), which sample to look at, or asks for best practices.
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

## Core Concepts

**Instance** — top-level entry point. `dubbo.NewInstance()` configures the application globally; create server and client from it.

**Protocol** — how bytes travel on the wire. Triple is the default (HTTP/2, gRPC-compatible). Dubbo is for Java interop with Hessian2. JSONRPC and REST are also built-in.

**Registry** — service discovery. Provider registers itself; consumer queries to find providers. Built-ins: Nacos, ZooKeeper, etcd, Polaris, Kubernetes.

**Filter** — interceptor chain around every RPC call, on both sides. Used for auth, metrics, tracing, rate limiting.

**Cluster** — fault-tolerance strategy when calling a provider group. Failover (default), Failfast, Failsafe, Forking, Broadcast.

**LoadBalance** — selects one provider from the candidate list. Random (default), RoundRobin, LeastActive, ConsistentHash, P2C.

**Router** — prunes the provider list before load balancing. Tag, condition, script. Rules typically come from the config center, not from custom code.

**MetadataReport / ConfigCenter** — application-level metadata storage and dynamic configuration. Same backend as the registry in most setups.

## SPI Extension Pattern

Every dubbo-go extension follows the same shape:

```go
// 1. Implement the interface
type myFilter struct{}
func (f *myFilter) Invoke(ctx context.Context, invoker base.Invoker, inv base.Invocation) result.Result {
    res := invoker.Invoke(ctx, inv)
    return res
}
func (f *myFilter) OnResponse(ctx context.Context, res result.Result, invoker base.Invoker, inv base.Invocation) result.Result {
    return res
}

// 2. Register in init()
func init() {
    extension.SetFilter("my-filter", func() filter.Filter { return &myFilter{} })
}

// 3. Activate with a blank import in main.go
//    import _ "github.com/yourorg/yourapp/filter/myfilter"

// 4. Enable per-service via the code API
//    pb.RegisterGreetServiceHandler(srv, impl, server.WithFilter("my-filter"))
```

The same pattern applies to Protocol, Registry, LoadBalance, Router, ConfigCenter, MetadataReport, Logger.

> Note: older samples still show `protocol.Invoker/Invocation/Result` aliases. They resolve to the same types in `protocol/base` and `protocol/result` — prefer the new packages in new code.

## Best Practices

- Use `_ "dubbo.apache.org/dubbo-go/v3/imports"` in development for convenience; switch to selective imports in production for smaller binaries.
- Define Triple services with Protobuf for cross-language interop.
- Configure via the code API: `dubbo.NewInstance(dubbo.WithRegistry(...), dubbo.WithProtocol(...))`. YAML via `dubbo.Load()` is the legacy path.
- Enable graceful shutdown by blank-importing `_ "dubbo.apache.org/dubbo-go/v3/graceful_shutdown"`.
- Use `client.WithProvidedBy` when application-level discovery cannot resolve which application owns the interface.
- Apply per-reference timeouts: `client.WithRequestTimeout(...)` or a `context.WithTimeout` at the call site.

## Triple-Only Capabilities

- **OpenAPI** — `triple.OpenAPIEnable(true)` publishes a runtime OpenAPI spec.
- **HTTP handler attachment** — `srv.AttachHTTPHandler("/path", handler)` mounts plain HTTP on the same port.
- **HTTP/3** — `triple.Http3Enable()` (experimental, TLS required).
- **CORS** — `triple.CORSAllowOrigins(...)` etc.
- **Reflection** — gRPC reflection is on by default; tools like `grpcurl` work without extra config.

## Samples Index

See [samples-index.md](samples-index.md) for the full `dubbo-go-samples` directory organized by scenario (Quick Start, Protocols, Service Discovery, Filters, Routing, Streaming, Observability, Security, Java Interop, Advanced).

## Related Skills

- `dubbo-go-scaffolding` — when the user wants to generate a provider/consumer
- `dubbo-go-extensions` — when a built-in is not enough and the user needs a custom SPI
- `dubbo-go-java-interop` — when the question involves dubbo-java compatibility
- `dubbo-go-debugging` — when the question is "why is X failing"
- `dubbo-go-migration` — when the question is "how do I move from Y to dubbo-go"
