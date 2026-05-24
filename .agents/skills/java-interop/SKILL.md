---
name: dubbo-go-java-interop
description: Use when connecting dubbo-go v3 with dubbo-java, choosing Triple versus Dubbo protocol, sharing Protobuf contracts, mirroring Java interfaces, handling Hessian2 POJOs, or debugging cross-language discovery and serialization.
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


# Dubbo Java and Go Interoperability

## Overview

Two interop paths, one decision: Triple + Protobuf for everything new and polyglot; Dubbo + Hessian2 only when the Java side already owns a non-proto interface. Picking the wrong path locks the team into bespoke serialization code.

## When to Use

- Bridging a Go service to existing Java Dubbo providers/consumers
- Choosing protocol/serialization for a new cross-language service
- Diagnosing `hessian: failed to decode`, missing-provider, or class-name-mismatch errors across languages

## Path Selection

| Path | Protocol | Serialization | Use when |
|---|---|---|---|
| Triple + Protobuf | Triple over HTTP/2 | Protobuf | New services, polyglot teams, gRPC-compatible clients, OpenAPI docs |
| Dubbo + Hessian2 | Dubbo TCP | Hessian2 | Existing Java Dubbo interfaces with POJO request/response classes |

## Triple + Protobuf

Shared `.proto`:

```protobuf
syntax = "proto3";

package org.apache.dubbo.sample;

option go_package = "github.com/yourorg/yourapp/proto;greet";
option java_package = "org.apache.dubbo.sample";
option java_multiple_files = true;

service Greeter {
  rpc SayHello(HelloRequest) returns (HelloReply);
}

message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

Go server:

```go
srv, err := server.NewServer(
    server.WithServerProtocol(
        protocol.WithPort(20000),
        protocol.WithTriple(),
    ),
)
if err != nil {
    return err
}

if err := greet.RegisterGreeterHandler(srv, &impl{}); err != nil {
    return err
}
return srv.Serve()
```

Triple exposes JSON-over-HTTP, which is useful for smoke tests:

```bash
curl -H "Content-Type: application/json" \
  -d '{"name":"Dubbo"}' \
  http://127.0.0.1:20000/org.apache.dubbo.sample.Greeter/SayHello
```

The path is `/<proto package>.<service>/<rpc method>`. Use the method name from the `.proto` file, for example `Greet` or `SayHello`.

## Triple Interop Rules

- `package` in `.proto` controls the Triple service path.
- `go_package` controls Go package path and package name.
- `java_package` controls Java generated classes and interface package.
- Renaming proto package, service, or method names is a wire-level breaking change.
- Use the same `.proto` source for Go and Java generation.
- Triple supports reflection and HTTP-friendly testing; use OpenAPI when documenting REST-like access for humans.

## Dubbo + Hessian2

Use this for Java-defined services without Protobuf. The Go side must mirror Java classes and register POJOs.

```go
package greet

import (
    dubbohessian "github.com/apache/dubbo-go-hessian2"
)

type GreetRequest struct {
    Name string
}

func (*GreetRequest) JavaClassName() string {
    return "org.apache.dubbo.hessian2.api.GreetRequest"
}

type GreetResponse struct {
    Greeting string
}

func (*GreetResponse) JavaClassName() string {
    return "org.apache.dubbo.hessian2.api.GreetResponse"
}

func init() {
    dubbohessian.RegisterPOJO(new(GreetRequest))
    dubbohessian.RegisterPOJO(new(GreetResponse))
}
```

Server protocol:

```go
srv, err := server.NewServer(
    server.WithServerProtocol(
        protocol.WithPort(20000),
        protocol.WithDubbo(),
    ),
)
```

Consumer reference:

```go
svc, err := greet.NewGreetingsService(
    cli,
    client.WithProtocolDubbo(),
    client.WithSerialization("hessian2"),
)
```

## Hessian2 Rules

- Every POJO type must implement `JavaClassName()` and be registered with `RegisterPOJO`.
- Fully-qualified Java class names must match exactly.
- Prefer primitives, strings, slices, maps, and registered nested POJOs.
- Avoid Go-only shapes such as channels, functions, and generics in RPC DTOs.
- Java enum mapping depends on Java-side serialization; verify before assuming string values.
- `hessian: failed to decode` usually means class name mismatch, missing `RegisterPOJO`, field/type mismatch, or wrong protocol/serialization.

## Service Discovery Across Languages

Use the same registry cluster and compatible registry mode on both sides.

Provider:

```go
ins, err := dubbo.NewInstance(
    dubbo.WithName("polyglot-provider"),
    dubbo.WithRegistry(registry.WithNacos(), registry.WithAddress("127.0.0.1:8848")),
    dubbo.WithProtocol(protocol.WithTriple(), protocol.WithPort(20000)),
)
```

Consumer:

```go
svc, err := greet.NewGreeterService(
    cli,
    client.WithInterface("org.apache.dubbo.sample.Greeter"),
    client.WithProvidedBy("polyglot-provider"),
)
```

`WithProvidedBy` is useful when application-level service discovery cannot load interface-to-application mapping from the metadata center.

Registry modes:

```go
registry.WithRegisterService()
registry.WithRegisterInterface()
registry.WithRegisterServiceAndInterface()
```

Use `WithRegisterServiceAndInterface` when bridging old interface-level discovery and newer application-level discovery during migration.

## Decision Table

| Scenario | Choice |
|---|---|
| New Go and Java service | Triple + Protobuf |
| Java service already has `.proto` | Triple + Protobuf |
| Existing Java interface only, no proto | Dubbo + Hessian2 |
| Need browser/curl smoke tests or OpenAPI docs | Triple + Protobuf |
| Need to preserve old Java POJOs exactly | Dubbo + Hessian2 |
| Metadata mapping is missing | Add metadata report or set `client.WithProvidedBy` |

## Reference Samples

- `dubbo-go-samples/java_interop/protobuf-triple`
- `dubbo-go-samples/java_interop/non-protobuf-dubbo`
- `dubbo-go-samples/java_interop/non-protobuf-triple`
- `dubbo-go-samples/java_interop/service_discovery`
- `dubbo-go-samples/helloworld`
- `dubbo-go-samples/direct`
- `dubbo-go-samples/http3`

## Related Skills

- `dubbo-go-scaffolding` - when generating the Go side of a polyglot service
- `dubbo-go-extensions` - when adding a Hessian2 codec, Java-aware filter, or custom registry shared with Java
- `dubbo-go-migration` - when porting an existing Java Dubbo service or splitting a Spring Cloud monolith
- `dubbo-go-debugging` - when the symptom is `hessian: failed to decode`, missing provider across languages, or registry-mode mismatch
