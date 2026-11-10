# dubbogo-cli

[English](README.md) | 中文

`dubbogo-cli` 是 Apache dubbo-go 工具链的命令行助手，用于创建项目脚手架、安装配套工具、查看注册中心或元数据中心信息，以及在本地调试 Dubbo 服务。

## 安装

```bash
go install dubbo.apache.org/dubbo-go/v3/tools/dubbogo-cli@latest
```

请确保 `$GOPATH/bin` 或 `$(go env GOPATH)/bin` 已加入 `PATH`。

## 命令速览

```bash
dubbogo-cli version
dubbogo-cli install all
dubbogo-cli install triple
dubbogo-cli install triple-openapi
dubbogo-cli install formatter
dubbogo-cli newDemo .
dubbogo-cli newApp .
dubbogo-cli show --r zookeeper --h 127.0.0.1:2181
dubbogo-cli show --mc zookeeper --h 127.0.0.1:2181
dubbogo-cli call --h localhost --p 20001 --proto dubbo --i com.example.UserProvider --method GetUser --sendObj ./request.json --recvObj ./response.json
dubbogo-cli hessian --include ./
```

## 安装配套工具

`dubbogo-cli install all` 会安装当前 dubbo-go 项目常用的配套工具：

- `protoc-gen-go-triple`：生成 Triple 客户端和服务端绑定代码。
- `protoc-gen-triple-openapi`：从 Triple protobuf 定义生成 OpenAPI v3 文档。
- `imports-formatter`：按照 dubbo-go 的 import 分组规则整理 Go import 块。

也可以单独安装：`dubbogo-cli install triple`、`dubbogo-cli install triple-openapi`、`dubbogo-cli install formatter`。

## 创建 Demo

```bash
mkdir hello-triple
cd hello-triple
dubbogo-cli newDemo .
make proto-gen
go mod tidy
```

生成目录：

```text
.
|-- Makefile
|-- api
|   `-- samples_api.proto
|-- go-client
|   `-- cmd
|       `-- client.go
|-- go-server
|   `-- cmd
|       `-- server.go
`-- go.mod
```

启动服务端：

```bash
go run ./go-server/cmd
```

在另一个终端启动客户端：

```bash
go run ./go-client/cmd
```

脚手架只保留 protobuf 源文件。执行 `make proto-gen` 后，会通过 `protoc-gen-go` 和 `protoc-gen-go-triple` 生成 `*.pb.go` 与 `*.triple.go`。

## 创建应用模板

```bash
mkdir dubbo-go-app
cd dubbo-go-app
dubbogo-cli newApp .
make proto-gen
go mod tidy
```

生成目录：

```text
.
|-- Makefile
|-- api
|   `-- api.proto
|-- build
|   `-- Dockerfile
|-- chart
|   |-- app
|   `-- nacos_env
|-- cmd
|   `-- app.go
|-- conf
|   `-- dubbogo.yaml
|-- go.mod
`-- pkg
    `-- service
        `-- service.go
```

常见开发流程：

1. 修改 `api/api.proto`。
2. 执行 `make proto-gen`。
3. 执行 `go mod tidy`。
4. 在 `pkg/service` 中实现服务。
5. 修改 `Makefile` 和 `chart/app/values.yaml` 中的镜像与 Helm 配置。
6. 使用模板提供的 Make target 构建、发布和部署。

常用 Make target：

```bash
make proto-gen
make build
make buildx-publish
make deploy
make remove
```

## 查看注册中心和元数据中心

Zookeeper 注册中心：

```bash
dubbogo-cli show --r zookeeper --h 127.0.0.1:2181
```

Zookeeper 元数据中心：

```bash
dubbogo-cli show --mc zookeeper --h 127.0.0.1:2181
```

命令会输出注册的接口和方法名。Nacos 与 Istio 支持尚未实现。

## 调试 Dubbo 协议服务

`dubbogo-cli call` 可以通过 JSON 文件描述请求和响应结构，并调用 Dubbo 协议服务。

```bash
dubbogo-cli call \
  --h localhost \
  --p 20001 \
  --proto dubbo \
  --i com.example.UserProvider \
  --method GetUser \
  --sendObj ./request.json \
  --recvObj ./response.json
```

请求字段使用 `type@value` 形式：

```json
{
  "ID": "string@A000",
  "Male": "bool@true",
  "JavaClassName": "string@com.example.CallUserStruct"
}
```

每个 Hessian 对象都应提供 `JavaClassName`，并与服务端类型保持一致。

## 调试 Triple 服务

Triple 服务兼容 gRPC 生态。服务端开启 reflection 后，可以使用 `grpc_cli` 查看和调用接口：

```bash
grpc_cli ls localhost:20000 -l
grpc_cli type localhost:20000 org.apache.dubbo.samples.HelloRequest
grpc_cli call localhost:20000 SayHello "name: 'laurence'"
```

`grpc_cli` 请参考 gRPC 官方文档安装。
