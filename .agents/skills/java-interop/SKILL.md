---
name: dubbo-go-java-interop
description: Guides interoperability between dubbo-go v3 and dubbo-java — cross-language RPC using Triple+Protobuf or Dubbo+Hessian2. Use when the user asks how to call a Java Dubbo service from Go, expose a Go service to Java clients, share a proto/interface across languages, or pick between Triple and Dubbo for interop.
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

# Dubbo Java and Go Interop

A single service definition can be served by Go and consumed by Java (or vice versa) without a gateway. Two viable paths:

| Path | Wire format | Serialization | When to use |
|---|---|---|---|
| **Triple + Protobuf** (recommended) | HTTP/2 | Protobuf | New services, polyglot teams, gRPC-compatible clients |
| **Dubbo + Hessian2** | TCP (dubbo) | Hessian2 | Calling existing Java services that were originally defined as Java interfaces (no `.proto`) |

Pick Triple unless integrating with an existing Hessian2 Java codebase.

## Path 1: Triple + Protobuf (recommended)

A shared `.proto` file drives both sides. Set both `go_package` and `java_package`:

```protobuf
syntax = "proto3";

package org.apache.dubbo.sample;

option go_package = "github.com/yourorg/yourapp/proto;greet";
option java_package = "org.apache.dubbo.sample";
option java_multiple_files = true;

service Greeter {
  rpc SayHello(HelloRequest) returns (HelloReply);
}

message HelloRequest  { string name = 1; }
message HelloReply    { string message = 1; }
```

**Go server** — nothing interop-specific, just Triple:

```go
srv, _ := server.NewServer(
    server.WithServerProtocol(
        protocol.WithPort(20000),
        protocol.WithTriple(),
    ),
)
greet.RegisterGreeterHandler(srv, &impl{})
srv.Serve()
```

**Test interop with curl** — Triple supports JSON over HTTP/1.1:

```bash
curl -H "Content-Type: application/json" \
     -d '{"name":"Dubbo"}' \
     http://localhost:20000/org.apache.dubbo.sample.Greeter/sayHello
```

The path is `/<java_package>.<service>/<method>` — the Java side uses lowercase-first-letter method names by convention.

**Java client** uses standard dubbo-java APIs; no Go-specific code. See [java_interop/protobuf-triple](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/protobuf-triple) for the working pair.

### Rules of thumb for Triple interop

- **Method-name casing**: Protobuf generators emit `SayHello` in Go and `sayHello` on the Java wire. The HTTP route uses the lowercase-first form.
- **Package matters**: `java_package` in the proto becomes the Java interface FQN — pick deliberately, renaming later is breaking.
- **Reflection**: Triple exposes gRPC reflection automatically; tools like `grpcurl` and `bloomrpc` work without extra config.

## Path 2: Dubbo + Hessian2 (Java-defined services)

Use this when the service is **defined by a Java interface** (no `.proto`) and you need to add a Go side.

The Go side mirrors the Java POJO via Hessian2 registration. Field names, types, and `JavaClassName()` must match the Java class.

```go
package greet

import (
    dubbo_go_hessian2 "github.com/apache/dubbo-go-hessian2"
)

type GreetRequest struct {
    Name string
}

func (x *GreetRequest) JavaClassName() string {
    return "org.apache.dubbo.hessian2.api.GreetRequest"
}

type GreetResponse struct {
    Greeting string
}

func (x *GreetResponse) JavaClassName() string {
    return "org.apache.dubbo.hessian2.api.GreetResponse"
}

func init() {
    dubbo_go_hessian2.RegisterPOJO(new(GreetRequest))
    dubbo_go_hessian2.RegisterPOJO(new(GreetResponse))
}
```

**Go server** — pick the Dubbo protocol, not Triple:

```go
srv, _ := server.NewServer(
    server.WithServerProtocol(
        protocol.WithPort(20000),
        protocol.WithDubbo(),
    ),
)
greet.RegisterGreetServiceHandler(srv, &impl{})
srv.Serve()
```

See [java_interop/non-protobuf-dubbo](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/non-protobuf-dubbo) for the canonical end-to-end example.

### Hessian2 gotchas

- **Field names are case-sensitive on the wire** — Go's exported names are serialized lowercase-first to match Java conventions. The hessian2 library handles this; custom struct tags can break it.
- **Unsupported types**: Go generics, channels, functions will not serialize. Stick to primitives, slices, maps, and other registered POJOs.
- **Unregistered POJOs** → `hessian: failed to decode` at runtime. Always `RegisterPOJO` in the package's `init()`.
- **Java `enum` maps to Go `int`** (the ordinal), not a string — unless the Java side customized serialization.
- **Time**: Java `Date` ↔ Go `time.Time` generally works; `java.time.*` types need extra care.

## Service Discovery Across Languages

When both sides use the **same registry** (Nacos / ZooKeeper), they discover each other automatically — the registry entry is language-agnostic, just a URL with protocol + address.

```go
dubbo.NewInstance(
    dubbo.WithName("polyglot-service"), // same application name as the Java side
    dubbo.WithRegistry(registry.WithNacos(), registry.WithAddress("127.0.0.1:8848")),
    dubbo.WithProtocol(protocol.WithTriple(), protocol.WithPort(20000)),
)
```

If application-level discovery cannot resolve which application owns the interface, hint with `client.WithProvidedBy("<java-app-name>")`.

See [java_interop/service_discovery](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/service_discovery).

## Decision Table

| Scenario | Path |
|---|---|
| New service, Go and Java teams both need clients | **Triple + Protobuf** |
| Calling an existing Java Dubbo service defined by a Java interface | **Dubbo + Hessian2** |
| Legacy Java service exports both Triple and Dubbo endpoints | **Triple + Protobuf** on the Go side |
| Need a gRPC-compatible wire | **Triple + Protobuf** |
| Cannot touch the Java side, and it is not Protobuf | **Dubbo + Hessian2** |

## Reference Samples

- [java_interop/protobuf-triple](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/protobuf-triple) — both directions, Triple + Protobuf
- [java_interop/non-protobuf-dubbo](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/non-protobuf-dubbo) — Hessian2 POJO bridge
- [java_interop/non-protobuf-triple](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/non-protobuf-triple) — Triple without Protobuf (rare)
- [java_interop/service_discovery](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/service_discovery) — Nacos-based cross-language discovery

Official guide: [Interoperability with Dubbo Java](https://cn.dubbo.apache.org/zh-cn/overview/mannual/golang-sdk/tutorial/interop-dubbo/).

## Related Skills

- `dubbo-go-scaffolding` — for the Go side of a polyglot service
- `dubbo-go-extensions` — when adding a Hessian2 codec, Java-aware filter, or shared registry
- `dubbo-go-migration` — when porting an existing Java service or splitting a Spring Cloud monolith
- `dubbo-go-debugging` — when the symptom is `hessian: failed to decode`, missing-provider across languages, or registry-mode mismatch
