---
name: dubbo-go-debugging
description: Use when diagnosing dubbo-go runtime failures, logs, startup errors, missing providers, registry problems, serialization errors, timeouts, filter panics, OpenAPI issues, HTTP handler mounting, or graceful shutdown behavior.
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


# Debugging dubbo-go

## Overview

Most dubbo-go runtime failures collapse into a small set of root causes: name/contract mismatch, registry/discovery mode mismatch, protocol mismatch, blocking filter, or missing blank import. Diagnose by symptom first, then verify the contract on both sides (provider and consumer).

## Quick Triage

| Symptom in logs | Jump to |
|---|---|
| `no provider available`, `no route`, `Should has at least one way to know` | [Missing Provider or No Route](#missing-provider-or-no-route) |
| `dial tcp ... connection refused`, `i/o timeout` on connect | [Connection Refused or Dial Error](#connection-refused-or-dial-error) |
| `hessian: failed to decode`, `protobuf: cannot parse invalid wire-format data` | [Serialization or Decode Error](#serialization-or-decode-error) |
| `context deadline exceeded`, `invoke timeout` | [Timeout](#timeout) |
| `filter not found`, panic inside a filter | [Filter Not Found or Filter Panic](#filter-not-found-or-filter-panic) |
| Provider process exits seconds after start | [Provider Starts Then Exits](#provider-starts-then-exits) |
| `404` at `/dubbo/openapi/...`, empty spec | [OpenAPI 404 or Empty Spec](#openapi-404-or-empty-spec) |
| `AttachHTTPHandler` returns error | [Attached HTTP Handler Errors](#attached-http-handler-errors) |
| Pods drain slowly, in-flight requests truncated on shutdown | [Graceful Shutdown Issues](#graceful-shutdown-issues) |

## Collect First

Before diving in, gather:

1. Full error message and stack trace
2. Protocol: Triple, Dubbo, JSONRPC, REST, or gRPC-compatible path
3. Discovery mode: direct URL, Nacos, ZooKeeper, etcd, Polaris, mesh
4. Provider and consumer snippets for `NewInstance`, `NewServer`, `NewClient`, and `Register/NewService`
5. Whether the service uses Protobuf IDL, non-IDL, Hessian2, generic, OpenAPI, HTTP/3, or attached HTTP handlers

## Missing Provider or No Route

Common logs:

```text
no provider available
No provider available
no route
Should has at least one way to know which services this interface belongs to
```

Check:

- Provider reached `srv.Serve()` and did not exit.
- Provider registered the service: `pb.RegisterXxxHandler(srv, impl, ...)` returned nil.
- Consumer constructed the generated client with the same interface name. Override with `client.WithInterface("...")` if provider used `server.WithInterface("...")`.
- Registry addresses and IDs match. Multiple registries require `registry.WithID(...)` plus `client.WithRegistryIDs(...)` or server equivalents.
- Application-level discovery can resolve interface-to-application mapping. If metadata mapping is unavailable, set `client.WithProvidedBy("provider-app-name")`.
- Registry registration mode is correct: `registry.WithRegisterService`, `registry.WithRegisterInterface`, or `registry.WithRegisterServiceAndInterface`.
- Registry is not disabled by `server.WithNotRegister()` or service `server.WithNotRegister()`.
- Protocol matches. Triple consumers default to `tri`; Dubbo/Hessian2 consumers must use Dubbo protocol explicitly.

For Nacos, inspect by application name for service-discovery mode and by interface if interface registration is enabled.

## Connection Refused or Dial Error

Check:

- Provider is listening on the expected IP and port.
- For direct mode, `client.WithClientURL("127.0.0.1:20000")` or `tri://127.0.0.1:20000` points to the provider.
- Docker/Kubernetes clients are not using `localhost` to reach another container/pod.
- Triple server configured an explicit port if using `AttachHTTPHandler`.
- TLS/HTTP3 clients match server TLS and HTTP/3 setup.

Useful commands:

```bash
lsof -i :20000
curl -v http://127.0.0.1:20000/greet.GreetService/Greet \
  -H "Content-Type: application/json" \
  -d '{"name":"Dubbo"}'
```

## Serialization or Decode Error

Examples:

```text
hessian: failed to decode
protobuf: cannot parse invalid wire-format data
```

Check:

- Triple path uses Protobuf-generated code and the same `.proto` contract.
- Dubbo protocol with Hessian2 registers every POJO through `dubbo-go-hessian2.RegisterPOJO`.
- Java class names from `JavaClassName()` exactly match the Java fully-qualified class.
- Client protocol and provider protocol match.
- Generic calls use the expected generic type: `"true"`, `"gson"`, `"protobuf"`, or `"protobuf-json"`.

## Timeout

Examples:

```text
context deadline exceeded
invoke timeout
```

Check:

- Provider receives the request.
- Filters are not blocking, rate-limiting, or waiting on external systems.
- Consumer timeout is long enough.

Per-reference timeout:

```go
svc, _ := pb.NewGreetService(cli, client.WithRequestTimeout(10*time.Second))
```

Client-wide timeout:

```go
cli, _ := client.NewClient(client.WithClientRequestTimeout(10*time.Second))
```

Call-site timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
```

## Filter Not Found or Filter Panic

Check:

- The filter package is imported with `_ "path/to/filter"`.
- `extension.SetFilter("name", ...)` matches `server.WithFilter("name")`, `client.WithFilter("name")`, or YAML config.
- If using built-ins in examples, `_ "dubbo.apache.org/dubbo-go/v3/imports"` is present.
- Panic is not from filter order or nil attachment assumptions.

## Provider Starts Then Exits

`srv.Serve()` is blocking and should not be launched in a goroutine unless something else blocks the process.

```go
if err := srv.Serve(); err != nil {
    logger.Fatalf("failed to serve: %v", err)
}
```

## OpenAPI 404 or Empty Spec

Check:

- OpenAPI was enabled through `triple.WithOpenAPI(triple.OpenAPIEnable(), ...)`.
- Service was registered after OpenAPI-enabled Triple protocol was configured.
- Correct path is used. Defaults include:
  - `/dubbo/openapi/openapi.json`
  - `/dubbo/openapi/openapi.yaml`
  - `/dubbo/openapi/api-docs/default.json`
  - `/dubbo/openapi/swagger-ui`
  - `/dubbo/openapi/redoc`
- Group-specific services use `server.WithOpenAPIGroup("group")` and can be requested with `/dubbo/openapi/api-docs/<group>.json`.

## Attached HTTP Handler Errors

`srv.AttachHTTPHandler(handler)` rules:

- Handler must not be nil.
- Attach before `Serve`.
- Only one root handler can be attached.
- At least one Triple protocol must be configured.
- Triple protocol needs an explicit port.

Use a mux for multiple HTTP subroutes.

## Graceful Shutdown Issues

Current shutdown config includes total timeout, step timeout, notify timeout, consumer update wait time, offline request window timeout, internal signal handling, and closing invoker expiry.

Check:

- Graceful shutdown filters are imported or included through `_ "dubbo.apache.org/dubbo-go/v3/imports"`.
- Shutdown config is passed through `dubbo.NewInstance`, `client.WithClientShutdown`, or root YAML.
- Long-running calls fit within `step-timeout` and `timeout`.
- `internal-signal` is disabled only if the application handles OS signals itself.

## Logs

Enable debug logging:

```go
ins, _ := dubbo.NewInstance(
    dubbo.WithLogger(
        logger.WithZap(),
        logger.WithLevel("debug"),
        logger.WithFormat("text"),
    ),
)
```

Trace integration can inject trace IDs into logs when OpenTelemetry tracing is configured:

```go
dubbo.WithLogger(logger.WithTraceIntegration(true))
```

## Related Skills

- `dubbo-go-extensions` - when a missing/registered SPI is the root cause (filter, LB, registry not found)
- `dubbo-go-java-interop` - when the symptom is Hessian2 decode failure or cross-language discovery
- `dubbo-go-guide` - for the conceptual map (Instance / Server / Client / Protocol / Registry / Filter)
- `dubbo-go-development` - when the issue is reproducible only inside the `apache/dubbo-go` repo and may need a fix or test there
