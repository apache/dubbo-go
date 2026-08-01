# dubbogo-cli

English | [中文](README_CN.md)

`dubbogo-cli` is the command-line helper for the Apache dubbo-go toolchain. It can scaffold dubbo-go projects, install companion tools, inspect registry or metadata-center entries, and call Dubbo services during local debugging.

## Installation

```bash
go install dubbo.apache.org/dubbo-go/v3/tools/dubbogo-cli@latest
```

Make sure `$GOPATH/bin` or `$(go env GOPATH)/bin` is in your `PATH`.

## Commands

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

## Install Toolchain

`dubbogo-cli install all` installs the companion tools used by current dubbo-go projects:

- `protoc-gen-go-triple`: generates Triple client and server bindings.
- `protoc-gen-triple-openapi`: generates OpenAPI v3 documents from Triple protobuf definitions.
- `imports-formatter`: formats Go import blocks using dubbo-go import grouping rules.

You can install a single tool with `dubbogo-cli install triple`, `dubbogo-cli install triple-openapi`, or `dubbogo-cli install formatter`.

## Create a Demo

```bash
mkdir hello-triple
cd hello-triple
dubbogo-cli newDemo .
make proto-gen
go mod tidy
```

Generated layout:

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

Run the server:

```bash
go run ./go-server/cmd
```

Run the client in another terminal:

```bash
go run ./go-client/cmd
```

The template keeps only protobuf source files in the scaffold. `make proto-gen` generates `*.pb.go` and `*.triple.go` files with `protoc-gen-go` and `protoc-gen-go-triple`.

## Create an Application

```bash
mkdir dubbo-go-app
cd dubbo-go-app
dubbogo-cli newApp .
make proto-gen
go mod tidy
```

Generated layout:

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

Typical workflow:

1. Edit `api/api.proto`.
2. Run `make proto-gen`.
3. Run `go mod tidy`.
4. Implement the service in `pkg/service`.
5. Update image and Helm settings in `Makefile` and `chart/app/values.yaml`.
6. Build, publish, and deploy with the provided Make targets.

Useful Make targets:

```bash
make proto-gen
make build
make buildx-publish
make deploy
make remove
```

## Inspect Registry and Metadata

Zookeeper registry:

```bash
dubbogo-cli show --r zookeeper --h 127.0.0.1:2181
```

Zookeeper metadata center:

```bash
dubbogo-cli show --mc zookeeper --h 127.0.0.1:2181
```

The command prints registered interfaces and method names. Nacos and Istio support are not implemented yet.

## Debug Dubbo Protocol Services

`dubbogo-cli call` can invoke Dubbo protocol services with request and response shapes described as JSON files.

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

Request fields use the form `type@value`, for example:

```json
{
  "ID": "string@A000",
  "Male": "bool@true",
  "JavaClassName": "string@com.example.CallUserStruct"
}
```

Every Hessian object should provide a `JavaClassName` value that matches the provider-side type.

## Debug Triple Services

Triple services are compatible with the gRPC ecosystem. If reflection is enabled on the provider, `grpc_cli` can inspect and call services:

```bash
grpc_cli ls localhost:20000 -l
grpc_cli type localhost:20000 org.apache.dubbo.samples.HelloRequest
grpc_cli call localhost:20000 SayHello "name: 'laurence'"
```

Install `grpc_cli` from the gRPC project documentation.
