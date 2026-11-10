# protoc-gen-go-triple

[English](README.md) | 中文

`protoc-gen-go-triple` 是 protobuf 编译插件，用于生成 dubbo-go Triple 客户端和服务端绑定代码。

它通常与 `protoc-gen-go` 一起使用：`protoc-gen-go` 生成 protobuf 消息类型，`protoc-gen-go-triple` 生成 Triple 服务接口、客户端、handler 和注册辅助函数。

## 环境要求

- Go 1.23 或更高版本
- `protoc`
- `protoc-gen-go`

如果还没有安装 `protoc-gen-go`：

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

## 安装

```bash
go install dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-go-triple@latest
```

请确保 `$(go env GOPATH)/bin` 已加入 `PATH`。

## 使用

```bash
protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-triple_out=. \
  --go-triple_opt=paths=source_relative \
  ./api/greet.proto
```

对于 `greet.proto`，会生成：

- `greet.pb.go`：由 `protoc-gen-go` 生成的 protobuf 消息类型。
- `greet.triple.go`：由 `protoc-gen-go-triple` 生成的 Triple 绑定代码。

## Proto 示例

```proto
syntax = "proto3";

package greet;

option go_package = "example.com/hello/api;api";

message GreetRequest {
  string name = 1;
}

message GreetResponse {
  string greeting = 1;
}

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse) {}
}
```

protobuf 的 `package` 会影响 Triple 服务在协议层暴露的服务名；`go_package` 用于控制生成文件的 Go import path 和 package name。

## 生成 API

当前默认生成代码使用 dubbo-go v3 Triple API：

```go
greeter, err := api.NewGreetService(cli)
```

```go
err := api.RegisterGreetServiceHandler(srv, &GreetServiceImpl{})
```

生成文件后缀是 `.triple.go`。

## 旧版本兼容

dubbo-go 3.2.0 及之后版本应使用默认生成结果。

如果需要生成兼容更早 dubbo-go 版本的代码，可以传入 `useOldVersion=true`：

```bash
protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-triple_out=. \
  --go-triple_opt=paths=source_relative,useOldVersion=true \
  ./api/greet.proto
```

旧版本兼容模式仅用于迁移。新项目应使用默认输出。
