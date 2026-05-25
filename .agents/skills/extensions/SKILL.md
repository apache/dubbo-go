---
name: dubbo-go-extensions
description: Guides writing custom dubbo-go v3 extensions — Filter, LoadBalance, Registry, Protocol, Router, Logger, ConfigCenter — through the SPI pattern. Use when the user asks how to write a filter, custom interceptor, custom load balancer, plug in a new registry, or hook into dubbo-go's extension points.
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

# Writing dubbo-go Extensions

dubbo-go exposes nearly every runtime behavior through a uniform SPI pattern:

1. Implement an interface
2. Call `extension.SetXxx(name, factory)` in an `init()`
3. Blank-import the package so `init()` runs
4. Enable the extension by name at the call site (`server.WithFilter`, `client.WithFilter`, etc.)

The extension point you pick depends on **what you want to intercept**:

| You want to... | Extension point | `extension.Set*` fn |
|---|---|---|
| Intercept every RPC call (auth, logging, metrics) | Filter | `SetFilter` |
| Choose which provider instance to call | LoadBalance | `SetLoadbalance` |
| Prune the provider list before LB (canary, A/B) | Router | `SetRouterFactory` (rules usually pushed via config center) |
| Plug in a new service-discovery backend | Registry | `SetRegistry` |
| Add a new wire protocol | Protocol | `SetProtocol` |
| Replace the logger backend | Logger | `SetLogger` |

## Filter (the 90% case)

Filters run on every RPC. They're the right extension point for auth, token injection, metrics, tracing, rate limiting, and request/response logging.

**Package layout**:
```
yourapp/filter/myfilter/myfilter.go
```

**myfilter.go**:
```go
package myfilter

import (
    "context"
    "time"
)

import (
    "dubbo.apache.org/dubbo-go/v3/common/extension"
    "dubbo.apache.org/dubbo-go/v3/filter"
    "dubbo.apache.org/dubbo-go/v3/protocol/base"
    "dubbo.apache.org/dubbo-go/v3/protocol/result"

    "github.com/dubbogo/gost/log/logger"
)

func init() {
    extension.SetFilter("timing", func() filter.Filter { return &timingFilter{} })
}

type timingFilter struct{}

func (f *timingFilter) Invoke(ctx context.Context, invoker base.Invoker, inv base.Invocation) result.Result {
    start := time.Now()
    res := invoker.Invoke(ctx, inv)
    logger.Infof("call %s took %s", inv.MethodName(), time.Since(start))
    return res
}

func (f *timingFilter) OnResponse(ctx context.Context, res result.Result, invoker base.Invoker, inv base.Invocation) result.Result {
    return res
}
```

**Activate in main.go**:
```go
import _ "github.com/yourorg/yourapp/filter/myfilter" // triggers init()

// Server side, per service
pb.RegisterGreetServiceHandler(srv, impl, server.WithFilter("timing"))

// Client side, per reference
svc, _ := pb.NewGreetService(cli, client.WithFilter("timing"))

// Or globally on the server/client:
//   server.WithServerFilter("timing")
//   client.WithClientFilter("timing")
```

**Chain multiple filters** with comma separation: `server.WithFilter("timing,auth")`. Filters run in order on the way in, reverse order on `OnResponse`.

**Modify attachments** (e.g. add a request ID) in `Invoke` via `inv.SetAttachment(key, val)`; read on the other side with `inv.Attachment(key)`.

**Short-circuit a call** by returning a result without invoking the next link:
```go
func (f *authFilter) Invoke(ctx context.Context, invoker base.Invoker, inv base.Invocation) result.Result {
    if !isAuthorized(inv) {
        return &result.RPCResult{Err: errors.New("unauthorized")}
    }
    return invoker.Invoke(ctx, inv)
}
```

See [filter/custom](https://github.com/apache/dubbo-go-samples/tree/main/filter/custom) for the canonical example.

## LoadBalance

Choose one provider out of a list. Built-ins: Random (default), RoundRobin, LeastActive, ConsistentHash, P2C.

Write a custom one only when the built-ins do not cover your selection rule (e.g. pick by request header, pick by tenant ID).

```go
package affinityLB

import (
    "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
    "dubbo.apache.org/dubbo-go/v3/common/extension"
    "dubbo.apache.org/dubbo-go/v3/protocol/base"
)

func init() {
    extension.SetLoadbalance("tenant-affinity", func() loadbalance.LoadBalance {
        return &tenantAffinityLB{}
    })
}

type tenantAffinityLB struct{}

func (lb *tenantAffinityLB) Select(invokers []base.Invoker, inv base.Invocation) base.Invoker {
    tenant := inv.Attachment("tenant-id")
    for _, iv := range invokers {
        if iv.GetURL().GetParam("tenant", "") == tenant {
            return iv
        }
    }
    return invokers[0]
}
```

Activate per reference:
```go
svc, _ := pb.NewGreetService(cli, client.WithLoadBalance("tenant-affinity"))
```

Or as the client default:
```go
cli, _ := client.NewClient(client.WithClientLoadBalance("tenant-affinity"))
```

## Registry (new service-discovery backend)

Only needed when the built-in set (Nacos, ZooKeeper, etcd, Polaris, Kubernetes) does not cover your platform.

```go
import (
    "dubbo.apache.org/dubbo-go/v3/common"
    "dubbo.apache.org/dubbo-go/v3/common/extension"
    "dubbo.apache.org/dubbo-go/v3/registry"
)

func init() {
    extension.SetRegistry("myregistry", func(url *common.URL) (registry.Registry, error) {
        return newMyRegistry(url)
    })
}
```

Implementing the full `registry.Registry` interface is a non-trivial undertaking — read `registry/nacos` or `registry/zookeeper` for reference.

Enable:
```go
dubbo.WithRegistry(registry.WithRegistry("myregistry"), registry.WithAddress("..."))
```

## Router (traffic-shaping)

Routers prune the provider list before the load balancer runs. Built-ins: tag, condition, script.

**Custom routers are uncommon.** The shipped routers read rules from the config center at runtime, so what users usually want is *rule authoring*, not a new router. See:
- [router/tag](https://github.com/apache/dubbo-go-samples/tree/main/router/tag) — tag-based canary
- [router/condition](https://github.com/apache/dubbo-go-samples/tree/main/router/condition) — predicate-based
- [router/script](https://github.com/apache/dubbo-go-samples/tree/main/router/script) — scripted rules

If you genuinely need a new routing algorithm, implement `router.PriorityRouter` and register via `extension.SetRouterFactory`.

## Protocol (new wire protocol)

Rarely needed. Implement `base.Protocol`: `Export(invoker)`, `Refer(url)`, `Destroy()`. Register with `extension.SetProtocol(name, factory)`. Most users should stick with Triple, Dubbo, JSONRPC, or REST.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| `filter not found` at startup | Blank import missing, or string in `WithFilter("x")` does not match `SetFilter("x", ...)` |
| Filter registered but never called | `init()` lives in a file that is never imported — check the blank-import path |
| LoadBalance not used | Built-in LB took precedence; ensure the name in `WithLoadBalance(...)` matches your registered name |
| Filter modifies attachment but not visible on the other side | Triple attachments travel as HTTP/2 headers; reserved header names may be dropped |
| Custom registry not picked up | `extension.SetRegistry("name", ...)` must match `registry.WithRegistry("name")` exactly |

## Key Rule of Thumb

The **name string is the contract**. Whatever you pass to `extension.SetFilter("timing", ...)` must be the exact string passed to `server.WithFilter("timing")` or `client.WithFilter("timing")`. Typos here produce silent failures — the extension simply does not run.

## Related Skills

- `dubbo-go-guide` — when deciding whether a built-in already covers the need
- `dubbo-go-scaffolding` — when the user also needs the surrounding provider/consumer skeleton
- `dubbo-go-debugging` — when an SPI is registered but does not run
