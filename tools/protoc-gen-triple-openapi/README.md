# protoc-gen-triple-openapi

English | [中文](README_CN.md)

`protoc-gen-triple-openapi` is a `protoc` plugin that generates OpenAPI v3 documents from protobuf services used with the Dubbo Triple protocol.

## Requirements

- Go 1.23 or later
- `protoc`
- protobuf files that define Triple services

## Installation

```bash
go install dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-triple-openapi@latest
```

Make sure `$(go env GOPATH)/bin` is in your `PATH`.

## Usage

Generate YAML, the default format:

```bash
protoc \
  --triple-openapi_out=. \
  --triple-openapi_opt=format=yaml \
  ./api/greet.proto
```

Generate JSON:

```bash
protoc \
  --triple-openapi_out=. \
  --triple-openapi_opt=format=json \
  ./api/greet.proto
```

To use `google.api.http` annotations as the REST contract, enable HTTP rules:

```bash
protoc \
  --triple-openapi_out=. \
  --triple-openapi_opt=format=yaml,use-http-rules=true \
  ./api/greet.proto
```

With HTTP rules enabled, annotated RPCs use their declared HTTP method and
path. Path variables become parameters, `body` becomes `requestBody`,
`response_body` controls the success payload, and `additional_bindings` produce
additional operations. RPCs without an annotation keep the canonical Triple
`POST /Service/Method` operation.

Generated file names use the input proto base name:

- `greet.triple.openapi.yaml`
- `greet.triple.openapi.json`

## Proto Example

```proto
syntax = "proto3";

package org.apache.dubbo.samples.greet;

option go_package = "example.com/hello/api;api";

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse) {}
}

message GreetRequest {
  string name = 1;
}

message GreetResponse {
  string greeting = 1;
}
```

An annotated RPC can expose the same contract over REST:

```proto
import "google/api/annotations.proto";

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse) {
    option (google.api.http) = {
      get: "/v1/greetings/{name}"
    };
  }
}
```

For runtime Triple transcoding, enable the route and OpenAPI features in the
server configuration independently:

```yaml
dubbo:
  protocols:
    triple:
      triple:
        http-transcoding:
          enabled: true
        openapi:
          enabled: true
```

The REST routes are available when `http-transcoding.enabled` is true. Runtime
OpenAPI uses the annotated routes only when both switches are enabled.
When one HTTP path is exported by multiple Dubbo group/version pairs, select
the implementation with the `tri-service-group` and `tri-service-version`
request headers.

## Options

| Option | Values | Default | Description |
| --- | --- | --- | --- |
| `format` | `yaml`, `json` | `yaml` | Output document format. |
| `use-http-rules` | `true`, `false` | `false` | Generate REST method/path, parameters, and request/response schemas from `google.api.http`. |

## Version

```bash
protoc-gen-triple-openapi --version
```
