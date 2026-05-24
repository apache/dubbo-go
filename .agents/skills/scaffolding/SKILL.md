---
name: dubbo-go-scaffolding
description: Use when creating, bootstrapping, or updating a dubbo-go v3 provider, consumer, sample, proto-based service, direct connection demo, registry-backed service, OpenAPI service, or HTTP-mounted Triple service.
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


# Scaffolding dubbo-go Services

## Overview

Generate dubbo-go v3 provider/consumer skeletons in the current code-API style: `server.NewServer`, `client.NewClient`, and `dubbo.NewInstance`. Reach for YAML only when migrating an existing config-driven app.

## When to Use

- New provider, consumer, sample, or `.proto`-based service
- Adding direct mode, registry-backed, OpenAPI, HTTP/3, CORS, or HTTP-handler-mounted variants
- Generating Triple bindings from a fresh `.proto`

## When NOT to Use

- The repo already has the skeleton and you only need to add a method or option - edit in place
- The user wants to write a custom SPI extension - use `dubbo-go-extensions`
- The user is migrating from another framework - use `dubbo-go-migration`

## Current Defaults

- Module path: `dubbo.apache.org/dubbo-go/v3`
- Go version in `go.mod`: `1.25.0`
- Preferred protocol: Triple (`protocol.WithTriple()`), gRPC-compatible and HTTP-friendly
- Default direct server port in samples: `20000`
- Generated Triple files: `*.pb.go` and `*.triple.go`
- Recommended sample baseline: `dubbo-go-samples/helloworld`

## Ask First

Clarify these before generating files:

1. Protocol: Triple by default; Dubbo/Hessian2 for legacy Java interface interop; gRPC only when explicitly required.
2. Discovery mode: direct URL for local demos; Nacos, ZooKeeper, etcd, or Polaris for service discovery.
3. Service definition: reuse an existing `.proto` if available; otherwise create one with stable `package` and `go_package`.
4. Extras: OpenAPI docs, HTTP/3, CORS, TLS, metrics/probe, or an attached existing `http.Handler`.

## Layout

Mirror the sample repository shape:

```text
myservice/
|-- go.mod
|-- proto/
|   |-- greet.proto
|   |-- greet.pb.go
|   `-- greet.triple.go
|-- go-server/cmd/main.go
`-- go-client/cmd/main.go
```

`dubbogo-cli newDemo .` can generate a direct-mode demo. Use `dubbogo-cli newApp .` for a larger application template with `api`, `cmd`, `conf`, `pkg`, `build`, and `chart`.

## Proto

```protobuf
syntax = "proto3";

package greet;

option go_package = "github.com/yourorg/myservice/proto;greet";

message GreetRequest {
  string name = 1;
}

message GreetResponse {
  string greeting = 1;
}

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse);
}
```

Install generators:

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

The `.proto` `package` is part of the Triple HTTP path. For the example above, JSON-over-HTTP requests use `/greet.GreetService/Greet`.

## Direct Provider

```go
package main

import (
    "context"
)

import (
    _ "dubbo.apache.org/dubbo-go/v3/imports"
    "dubbo.apache.org/dubbo-go/v3/protocol"
    "dubbo.apache.org/dubbo-go/v3/server"

    "github.com/dubbogo/gost/log/logger"
)

import (
    greet "github.com/yourorg/myservice/proto"
)

type GreetTripleServer struct{}

func (s *GreetTripleServer) Greet(ctx context.Context, req *greet.GreetRequest) (*greet.GreetResponse, error) {
    return &greet.GreetResponse{Greeting: req.Name}, nil
}

func main() {
    srv, err := server.NewServer(
        server.WithServerProtocol(
            protocol.WithPort(20000),
            protocol.WithTriple(),
        ),
    )
    if err != nil {
        logger.Fatalf("failed to create server: %v", err)
    }

    if err := greet.RegisterGreetServiceHandler(srv, &GreetTripleServer{}); err != nil {
        logger.Fatalf("failed to register greet service handler: %v", err)
    }

    if err := srv.Serve(); err != nil {
        logger.Fatalf("failed to serve: %v", err)
    }
}
```

## Direct Consumer

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

import (
    greet "github.com/yourorg/myservice/proto"
)

func main() {
    cli, err := client.NewClient(
        client.WithClientURL("127.0.0.1:20000"),
    )
    if err != nil {
        logger.Fatalf("failed to create client: %v", err)
    }

    svc, err := greet.NewGreetService(cli)
    if err != nil {
        logger.Fatalf("failed to create greet service: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    resp, err := svc.Greet(ctx, &greet.GreetRequest{Name: "world"})
    if err != nil {
        logger.Fatalf("failed to call greet: %v", err)
    }
    logger.Infof("Greet response: %s", resp.Greeting)
}
```

## Registry-backed Provider and Consumer

Use `dubbo.NewInstance` when provider and consumer should share application, registry, protocol, logger, shutdown, metrics, or tracing config.

```go
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
if err != nil {
    logger.Fatalf("new instance: %v", err)
}

srv, err := ins.NewServer()
if err != nil {
    logger.Fatalf("new server: %v", err)
}
```

Registry options:

- Nacos: `registry.WithNacos()`
- ZooKeeper: `registry.WithZookeeper()`
- etcd v3: `registry.WithEtcdV3()`
- Polaris: `registry.WithPolaris()`
- Multiple registries: use `registry.WithID("nacos")` and select with `client.WithRegistryIDs("nacos")` or `server.WithRegistryIDs([]string{"nacos"})`

For application-level service discovery, a consumer can use metadata mapping or set the provider application explicitly:

```go
svc, err := greet.NewGreetService(
    cli,
    client.WithInterface("greet.GreetService"),
    client.WithProvidedBy("myservice-provider"),
)
```

## OpenAPI, CORS, HTTP/3, HTTP Handler Mount

Enable runtime OpenAPI docs on Triple:

```go
server.WithServerProtocol(
    protocol.WithPort(20000),
    protocol.WithTriple(
        triple.WithOpenAPI(
            triple.OpenAPIEnable(),
            triple.OpenAPIInfoTitle("My Service"),
            triple.OpenAPIInfoVersion("1.0.0"),
        ),
    ),
)
```

Useful endpoints:

- `/dubbo/openapi/openapi.json`
- `/dubbo/openapi/openapi.yaml`
- `/dubbo/openapi/api-docs/default.json`
- `/dubbo/openapi/swagger-ui`
- `/dubbo/openapi/redoc`

Assign services to a docs group with `server.WithOpenAPIGroup("group-name")`.

Attach an existing HTTP handler to the same Triple listener:

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
if err := srv.AttachHTTPHandler(mux); err != nil {
    logger.Fatalf("attach http handler: %v", err)
}
```

Rules: call `AttachHTTPHandler` before `Serve`, attach only one root handler, and configure an explicit Triple port.

HTTP/3 is experimental and requires TLS-capable setup:

```go
protocol.WithTriple(triple.Http3Enable())
```

## Validation

For generated app skeletons:

```bash
go mod tidy
go test ./...
```

For changes inside the dubbo-go repository, use the repository development skill and prefer `make fmt`, targeted `go test`, then broader `make test` when relevant.

## Related Skills

- `dubbo-go-guide` - for the broader concept map and current capabilities
- `dubbo-go-extensions` - when scaffolding a service that also needs a custom Filter/LB/Registry
- `dubbo-go-java-interop` - when the new service must talk to dubbo-java
- `dubbo-go-migration` - when the user is moving from gRPC-Go, Spring Cloud, Gin, or older dubbo-go
- `dubbo-go-debugging` - when the new skeleton fails to start, register, or connect
