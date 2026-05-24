---
name: dubbo-go-extensions
description: Use when implementing or explaining dubbo-go extension points such as filters, load balancers, registries, protocols, routers, loggers, config centers, metrics, tracing, or custom SPI integration.
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

## Overview

Every dubbo-go extension follows the same four-step pattern: implement the interface, register a factory in `init()` through `dubbo.apache.org/dubbo-go/v3/common/extension`, blank-import the package, then activate by name. The string passed to `extension.SetXxx("name", ...)` is the contract - code-API options and YAML must use the exact same name.

## When to Use

- Adding a custom Filter, LoadBalance, Router, Registry, Protocol, ConfigCenter, or Logger backend
- Wiring an existing SPI into an application via blank import
- Diagnosing "filter not found", "loadbalance not found", or similar registration errors

## When NOT to Use

- Built-in filters/load balancers/routers already cover the use case (prefer config over code)
- The user only needs canary/tag/condition routing - use static or dynamic router config from `dubbo-go-guide`
- The change is application boilerplate - use `dubbo-go-scaffolding`

## Pick the Extension Point

| Need | Extension point | Register with |
|---|---|---|
| Intercept RPC calls | `filter.Filter` | `extension.SetFilter` |
| Choose one provider | `loadbalance.LoadBalance` | `extension.SetLoadbalance` |
| Filter provider lists | `router.PriorityRouterFactory` | `extension.SetRouterFactory` |
| Add service discovery backend | `registry.Registry` | `extension.SetRegistry` |
| Add wire protocol | `base.Protocol` | `extension.SetProtocol` |
| Add config center | `config_center.DynamicConfigurationFactory` | `extension.SetConfigCenterFactory` |
| Add logger backend | `logger.Logger` | `extension.SetLogger` |

The name string is the contract. The string in `SetXxx("name", ...)` must match the string used in code API or YAML.

## Filter

Use filters for auth, token injection, tracing, metrics, rate limiting, logging, and graceful shutdown behavior.

```go
package timing

import (
    "context"
    "time"
)

import (
    "github.com/dubbogo/gost/log/logger"
)

import (
    "dubbo.apache.org/dubbo-go/v3/common/extension"
    "dubbo.apache.org/dubbo-go/v3/filter"
    "dubbo.apache.org/dubbo-go/v3/protocol/base"
    "dubbo.apache.org/dubbo-go/v3/protocol/result"
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

Activate:

```go
import _ "github.com/yourorg/yourapp/filter/timing"

_ = pb.RegisterGreetServiceHandler(srv, impl, server.WithFilter("timing"))
svc, _ := pb.NewGreetService(cli, client.WithFilter("timing"))
```

Server-wide and client-wide defaults also exist:

```go
server.NewServer(server.WithServerFilter("timing"))
client.NewClient(client.WithClientFilter("timing"))
```

Multiple filters use comma-separated names, for example `server.WithFilter("timing,auth")`.

## LoadBalance

Implement this interface:

```go
type LoadBalance interface {
    Select([]base.Invoker, base.Invocation) base.Invoker
}
```

Example:

```go
func init() {
    extension.SetLoadbalance("tenant-affinity", func() loadbalance.LoadBalance {
        return &tenantAffinity{}
    })
}

type tenantAffinity struct{}

func (lb *tenantAffinity) Select(invokers []base.Invoker, inv base.Invocation) base.Invoker {
    tenant := inv.Attachment("tenant-id")
    for _, iv := range invokers {
        if iv.GetURL().GetParam("tenant", "") == tenant {
            return iv
        }
    }
    if len(invokers) == 0 {
        return nil
    }
    return invokers[0]
}
```

Activate per reference or client:

```go
svc, _ := pb.NewGreetService(cli, client.WithLoadBalance("tenant-affinity"))
cli, _ := client.NewClient(client.WithClientLoadBalance("tenant-affinity"))
```

Built-ins include Random, RoundRobin, LeastActive, ConsistentHashing, P2C, and adaptive strategies.

## Router

Routers run before load balancing and can prune the invoker list. Current built-ins include condition, tag, script, affinity, and Polaris routers.

Custom routers are uncommon. If the user only wants canary, tags, condition rules, or traffic split, prefer built-in routers and static/dynamic router config.

Custom router shape:

```go
type PriorityRouterFactory interface {
    NewPriorityRouter(url *common.URL) (router.PriorityRouter, error)
}

type PriorityRouter interface {
    Route([]base.Invoker, *common.URL, base.Invocation) []base.Invoker
    URL() *common.URL
    Priority() int64
    Notify([]base.Invoker)
}
```

Register with:

```go
extension.SetRouterFactory("my-router", func() router.PriorityRouterFactory {
    return &myFactory{}
})
```

Routers that accept static config can also implement `router.StaticConfigSetter`.

## Registry

Only write a registry if Nacos, ZooKeeper, etcd v3, or Polaris do not fit.

```go
func init() {
    extension.SetRegistry("myregistry", func(url *common.URL) (registry.Registry, error) {
        return newMyRegistry(url)
    })
}
```

Implement `registry.Registry`: `Register`, `UnRegister`, `Subscribe`, `UnSubscribe`, and `LoadSubscribeInstances`. Study `registry/nacos`, `registry/zookeeper`, or `registry/etcdv3` before creating a new backend.

Activate:

```go
dubbo.WithRegistry(
    registry.WithRegistry("myregistry"),
    registry.WithAddress("127.0.0.1:1234"),
)
```

Registry options now also control metadata/config-center use and registration mode:

```go
registry.WithRegisterService()
registry.WithRegisterInterface()
registry.WithRegisterServiceAndInterface()
registry.WithoutUseAsMetaReport()
registry.WithoutUseAsConfigCenter()
```

## Protocol

Most users should not implement a protocol. Use Triple, Dubbo, JSONRPC, REST, or gRPC-compatible Triple unless there is a strong reason.

Protocol interface:

```go
type Protocol interface {
    Export(base.Invoker) base.Exporter
    Refer(*common.URL) base.Invoker
    Destroy()
}
```

HTTP-backed protocols can optionally implement `base.HTTPHandlerHost` to support `server.AttachHTTPHandler`.

## Config Center and Logger

Config centers use `extension.SetConfigCenterFactory` or `extension.SetConfigCenter`. Existing backends include file, Nacos, ZooKeeper, and Apollo.

Logger backends use:

```go
extension.SetLogger("driver", func(url *common.URL) (logger.Logger, error) {
    return newLogger(url)
})
```

Current logger config supports driver, level, format, appender, file rolling config, and trace integration.

## Troubleshooting

| Symptom | Check |
|---|---|
| `filter not found` | Missing blank import or name mismatch |
| Filter never runs | The package containing `init()` is not imported |
| Custom LB ignored | Use `client.WithLoadBalance` or `client.WithClientLoadBalance` with the registered name |
| Router rule ignored | Confirm router factory name and whether rule is static, dynamic, service-scope, or application-scope |
| Registry not selected | Check registry ID and `WithRegistryIDs` |
| Metadata mapping missing | Use metadata report or set `client.WithProvidedBy` |
| Attached HTTP handler fails | Only Triple supports hosting it; attach before `Serve` and use an explicit port |

## Related Skills

- `dubbo-go-guide` - when deciding whether a built-in already covers the need before writing custom SPI
- `dubbo-go-scaffolding` - when the user also needs the surrounding provider/consumer skeleton
- `dubbo-go-debugging` - when an SPI is registered but does not run (missing blank import, name mismatch)
