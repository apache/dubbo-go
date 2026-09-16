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

如果要让 `google.api.http` 注解驱动 REST 契约，请开启 HTTP rules：

```bash
protoc \
  --triple-openapi_out=. \
  --triple-openapi_opt=format=yaml,use-http-rules=true \
  ./api/greet.proto
```

开启后，带注解的 RPC 会使用声明的 HTTP method 和 path；路径变量会生成为
参数，`body` 会生成为 `requestBody`，`response_body` 控制成功响应内容，
`additional_bindings` 会生成额外 operation。没有注解的 RPC 仍保留标准 Triple
`POST /Service/Method` operation。

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

带注解的 RPC 可以直接暴露 REST 路由：

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

运行时 Triple 转码和 OpenAPI 可以分别开启：

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

`http-transcoding.enabled` 开启后才会注册 REST 路由；只有两个开关同时开启时，
运行时 OpenAPI 才会展示注解路由。
同一路径存在多个 Dubbo group/version 时，可通过 `tri-service-group` 和
`tri-service-version` 请求头选择具体实现。

## 参数

| 参数 | 可选值 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `format` | `yaml`, `json` | `yaml` | 输出文档格式。 |
| `use-http-rules` | `true`, `false` | `false` | 使用 `google.api.http` 生成 REST method/path、参数和请求/响应 schema。 |

## 版本

```bash
protoc-gen-triple-openapi --version
```
