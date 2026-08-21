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

## Options

| Option | Values | Default | Description |
| --- | --- | --- | --- |
| `format` | `yaml`, `json` | `yaml` | Output document format. |

## Version

```bash
protoc-gen-triple-openapi --version
```
