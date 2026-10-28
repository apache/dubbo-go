# protoc-gen-triple-openapi

[English](README.md) | 中文

`protoc-gen-triple-openapi` 是一个 `protoc` 插件，用于从 Dubbo Triple 协议使用的 protobuf 服务定义生成 OpenAPI v3 文档。

## 环境要求

- Go 1.23 或更高版本
- `protoc`
- 定义了 Triple 服务的 protobuf 文件

## 安装

```bash
go install dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-triple-openapi@latest
```

请确保 `$(go env GOPATH)/bin` 已加入 `PATH`。

## 使用

生成 YAML，默认格式：

```bash
protoc \
  --triple-openapi_out=. \
  --triple-openapi_opt=format=yaml \
  ./api/greet.proto
```

生成 JSON：

```bash
protoc \
  --triple-openapi_out=. \
  --triple-openapi_opt=format=json \
  ./api/greet.proto
```

生成文件名基于输入 proto 文件名：

- `greet.triple.openapi.yaml`
- `greet.triple.openapi.json`

## Proto 示例

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

## 参数

| 参数 | 可选值 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `format` | `yaml`, `json` | `yaml` | 输出文档格式。 |

## 版本

```bash
protoc-gen-triple-openapi --version
```
