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
name: dubbo-go-scaffolding
description: Generates dubbo-go v3 provider or consumer skeletons in the code-API style (dubbo.NewInstance / server.NewServer / client.NewClient). Use when the user asks to create, bootstrap, or scaffold a new dubbo-go service, provider, or consumer, including direct mode, registry-backed, OpenAPI, HTTP-mounted, or HTTP/3 variants.
---

# Scaffolding dubbo-go Services

dubbo-go v3 has two entry styles. Use the **code API** by default — it is the style the samples repo has converged on. Fall back to YAML-driven `dubbo.Load()` only if the user is migrating an existing YAML project.

Before generating code, confirm three things:
1. **Protocol** — Triple (default, HTTP/2, gRPC-compatible) / Dubbo (Java interop with Hessian2) / JSONRPC / REST. Pick Triple unless the user has a reason.
2. **Registry** — Nacos / ZooKeeper / etcd / Polaris / direct (no registry). Pick direct for local demos, Nacos for production Apache stacks.
3. **Service definition** — does the user already have a `.proto` file?

## Project Layout

Mirror the [samples repo](https://github.com/apache/dubbo-go-samples) layout so the user can cross-reference:

```
myservice/
├── go.mod
├── proto/
│   ├── greet.proto
│   ├── greet.pb.go          # protoc-gen-go
│   └── greet.triple.go      # protoc-gen-go-triple
├── go-server/cmd/main.go
└── go-client/cmd/main.go
```

## Step 1: Proto Definition

```protobuf
syntax = "proto3";
package greet;
option go_package = "github.com/yourorg/myservice/proto;greet";

message GreetRequest  { string name = 1; }
message GreetResponse { string greeting = 1; }

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse);
}
```

Install codegen plugins (one-time):
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/dubbogo/protoc-gen-go-triple/v3@latest
```

Generate:
```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-triple_out=. --go-triple_opt=paths=source_relative \
       proto/greet.proto
```

## Step 2: Provider

**Direct mode** (no registry, simplest local demo):

```go
package main

import "context"

import (
    _ "dubbo.apache.org/dubbo-go/v3/imports"
    "dubbo.apache.org/dubbo-go/v3/protocol"
    "dubbo.apache.org/dubbo-go/v3/server"

    "github.com/dubbogo/gost/log/logger"
)

import greet "github.com/yourorg/myservice/proto"

type GreetTripleServer struct{}

func (s *GreetTripleServer) Greet(ctx context.Context, req *greet.GreetRequest) (*greet.GreetResponse, error) {
    return &greet.GreetResponse{Greeting: "hello " + req.Name}, nil
}

func main() {
    srv, err := server.NewServer(
        server.WithServerProtocol(
            protocol.WithPort(20000),
            protocol.WithTriple(),
        ),
    )
    if err != nil {
        logger.Fatalf("new server: %v", err)
    }

    if err := greet.RegisterGreetServiceHandler(srv, &GreetTripleServer{}); err != nil {
        logger.Fatalf("register handler: %v", err)
    }

    if err := srv.Serve(); err != nil {
        logger.Fatalf("serve: %v", err)
    }
}
```

**Registry-backed** — wrap with `dubbo.NewInstance` so registry and protocol are applied globally:

```go
import (
    "dubbo.apache.org/dubbo-go/v3"
    "dubbo.apache.org/dubbo-go/v3/registry"
)

func main() {
    ins, err := dubbo.NewInstance(
        dubbo.WithName("myservice-provider"),
        dubbo.WithRegistry(
            registry.WithNacos(),
            registry.WithAddress("127.0.0.1:8848"),
        ),
        dubbo.WithProtocol(
            protocol.WithTriple(),
            protocol.WithPort(20000),
        ),
    )
    if err != nil { panic(err) }

    srv, err := ins.NewServer()
    if err != nil { panic(err) }
    if err := greet.RegisterGreetServiceHandler(srv, &GreetTripleServer{}); err != nil { panic(err) }
    if err := srv.Serve(); err != nil { panic(err) }
}
```

Swap registries by changing two lines:
- Nacos: `registry.WithNacos()` + `registry.WithAddress("127.0.0.1:8848")`
- ZooKeeper: `registry.WithZookeeper()` + `registry.WithAddress("127.0.0.1:2181")`
- etcd: `registry.WithEtcdV3()` + `registry.WithAddress("127.0.0.1:2379")`
- Polaris: `registry.WithPolaris()` + `registry.WithAddress("127.0.0.1:8091")`

## Step 3: Consumer

**Direct mode**:

```go
package main

import (
    "context"
    "time"
)

import (
    "dubbo.apache.org/dubbo-go/v3/client"
    _ "dubbo.apache.org/dubbo-go/v3/imports"

    "github.com/dubbogo/gost/log/logger"
)

import greet "github.com/yourorg/myservice/proto"

func main() {
    cli, err := client.NewClient(client.WithClientURL("127.0.0.1:20000"))
    if err != nil { logger.Fatalf("new client: %v", err) }

    svc, err := greet.NewGreetService(cli)
    if err != nil { logger.Fatalf("new service: %v", err) }

    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    resp, err := svc.Greet(ctx, &greet.GreetRequest{Name: "world"})
    if err != nil { logger.Fatalf("call: %v", err) }
    logger.Infof("resp: %s", resp.Greeting)
}
```

**With registry** — drop `WithClientURL`, attach the same registry to an `Instance`:

```go
ins, _ := dubbo.NewInstance(
    dubbo.WithName("myservice-consumer"),
    dubbo.WithRegistry(registry.WithNacos(), registry.WithAddress("127.0.0.1:8848")),
)
cli, _ := ins.NewClient()
svc, _ := greet.NewGreetService(cli)
```

If application-level discovery cannot resolve which application owns the interface, hint with:

```go
svc, _ := greet.NewGreetService(cli, client.WithProvidedBy("myservice-provider"))
```

## Triple Extras

**OpenAPI docs** — Triple can publish a runtime OpenAPI spec:

```go
server.WithServerProtocol(
    protocol.WithPort(20000),
    protocol.WithTriple(
        triple.OpenAPIEnable(true),
        triple.OpenAPIInfoTitle("Greet API"),
    ),
)
```

See [rpc/triple/openapi](https://github.com/apache/dubbo-go-samples/tree/main/rpc/triple/openapi).

**Mount an HTTP handler** on the same Triple port (Triple-only):

```go
srv.AttachHTTPHandler("/healthz", http.HandlerFunc(healthz))
```

**HTTP/3** — experimental, requires TLS:

```go
protocol.WithTriple(triple.Http3Enable())
```

See [http3](https://github.com/apache/dubbo-go-samples/tree/main/http3).

## Conventions

These are non-obvious rules the samples repo follows; preserve them when scaffolding:

- **Three import blocks**: stdlib, third-party, same-module — separated by blank lines. The repo's `tools/imports-formatter` enforces this.
- **Blank-import `imports` during development**: `_ "dubbo.apache.org/dubbo-go/v3/imports"` pulls in every built-in filter, protocol, registry, serializer. Switch to selective imports for production builds.
- **Fail fast on startup** — samples `panic` or `logger.Fatalf` on `NewInstance` / `NewServer` / `NewClient` / `Register*Handler` errors. Do not swallow them.
- **Application name matters** — v3 defaults to **application-level** service discovery, so `dubbo.WithName("...")` is what appears in the registry, not the interface FQN.

## Quick Decision Table

| User says | Use |
|---|---|
| "just want to try it" / "local demo" | direct mode, no registry |
| "production" / "service discovery" | `dubbo.NewInstance` + Nacos or ZK |
| "talk to Java Dubbo" | Triple + Protobuf — see `dubbo-go-java-interop` |
| "expose REST or curl-friendly endpoint" | Triple + OpenAPI, or Triple REST mode |
| "existing YAML config" | `dubbo.Load()` (legacy) — see `config_yaml` sample |

## Reference Samples

- [helloworld](https://github.com/apache/dubbo-go-samples/tree/main/helloworld) — minimal Triple, direct mode
- [direct](https://github.com/apache/dubbo-go-samples/tree/main/direct) — explicit `tri://` URL
- [registry/nacos](https://github.com/apache/dubbo-go-samples/tree/main/registry/nacos)
- [registry/zookeeper](https://github.com/apache/dubbo-go-samples/tree/main/registry/zookeeper)
- [registry/etcd](https://github.com/apache/dubbo-go-samples/tree/main/registry/etcd)
- [rpc/triple/openapi](https://github.com/apache/dubbo-go-samples/tree/main/rpc/triple/openapi) — runtime OpenAPI
- [http3](https://github.com/apache/dubbo-go-samples/tree/main/http3) — Triple over HTTP/3

## Validation

```bash
go mod tidy
go build ./...
go test ./...
```

## Related Skills

- `dubbo-go-guide` — broader concept map and best practices
- `dubbo-go-extensions` — when the new service needs a custom Filter/LB/Registry
- `dubbo-go-java-interop` — when the new service must talk to dubbo-java
- `dubbo-go-debugging` — when the new skeleton fails to start, register, or connect
