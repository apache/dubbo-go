# Apache Dubbo for Golang

[![CI](https://github.com/apache/dubbo-go/workflows/CI/badge.svg)](https://github.com/apache/dubbo-go/actions)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/apache/dubbo-go/v3?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-go)](https://goreportcard.com/report/github.com/apache/dubbo-go)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

---

English | [中文](README_CN.md)

Apache Dubbo-go is a high-performance RPC and microservice framework compatible with other Dubbo language implementations. It leverages Golang's concurrency features to provide efficient service governance, including service discovery, load balancing, and traffic management. Dubbo-go supports multiple protocols, such as Dubbo, JSONRPC, Triple(gRPC-compatible), gRPC, HTTP, HTTP2, and HTTP/3 (experimental), ensuring seamless integration in heterogeneous environments.

You can visit [the official website](https://dubbo.apache.org/) for more information.

## Recent Updates

For detailed changes, refer to CHANGELOG.md.

- **3.3.1**: Optimized configuration hot-reloading with content-based caching to prevent redundant notifications. Added experimental HTTP/3 support, Apollo integration, and Triple protocol OpenAPI generation. Fixed critical race conditions in service discovery under high-concurrency.

- **3.3.0**: Introduced script routing, multi-destination conditional routing, Triple protocol keepalive and connection pooling, Nacos multi-category subscriptions, and enhancements for observability and interoperability.

## Getting started

### Prerequisites

- Go 1.24 or later (recommended for compatibility with recent updates).

### Installation

To install Dubbo-go, use the following command:


```Bash
go get dubbo.apache.org/dubbo-go/v3@latest
```

### Quick Example

You can learn how to develop a dubbo-go RPC application step by step in 5 minutes by following our [Quick Start](https://github.com/apache/dubbo-go-samples/tree/main/helloworld) demo. 

It's as simple as the code shown below, you define a service with Protobuf, provide your own service implementation, register it to a server, and start the server.

```go
func (srv *GreetTripleServer) Greet(ctx context.Context, req *greet.GreetRequest) (*greet.GreetResponse, error) {
	resp := &greet.GreetResponse{Greeting: req.Name}
	return resp, nil
}

func main() {
	srv, _ := server.NewServer(
		server.WithServerProtocol(
			protocol.WithPort(20000),
			protocol.WithTriple(),
		),
	)

	_ := greet.RegisterGreetServiceHandler(srv, &GreetTripleServer{})

	if err := srv.Serve(); err != nil {
		logger.Error(err)
	}
}
```

After the server is up and running, call your service via cURL:

```shell
curl \
    --header "Content-Type: application/json" \
    --data '{"name": "Dubbo"}' \
    http://localhost:20000/greet.GreetService/Greet
```

Or, you can start a standard dubbo-go client to call the service:

```go
func main() {
	cli, _ := client.NewClient(
		client.WithClientURL("127.0.0.1:20000"),
	)

	svc, _ := greet.NewGreetService(cli)

	resp, _ := svc.Greet(context.Background(), &greet.GreetRequest{Name: "hello world"})
	
	logger.Infof("Greet response: %s", resp.Greeting)
}
```

See the [samples](https://github.com/apache/dubbo-go-samples) for detailed information on usage. Next, learn how to deploy, monitor and manage the traffic of your dubbo-go application by visiting the official website.

## Features

![dubbo-go-architecture](./doc/imgs/arc.png)

Dubbo-go provides robust service governance capabilities:

- **RPC Protocols**: Triple, gRPC compatible and HTTP-friendly
- **Service Discovery**: Nacos, Zookeeper, Etcd, Polaris-mesh.
- **Load Balance**: Adaptive, Random, RoundRobin, LeastActive, ConsistentHash
- **Traffic Management**: traffic split, timeout, rate limiting, canary release
- **Configuration**: yaml file, dynamic configuration(Nacos, Apollo, Zookeeper, etc.).
- **Observability**: Metrics (Prometheus), tracing (OpenTelemetry v1.21.0+ with insecure options and standardized span names), logging (with service registration lifecycle events).
- **HA Strategy**: Failover, Failfast, Failsafe/Failback, Available, Broadcast, Forking.
- **Interoperability**: Full compatibility with Apache Dubbo (Java) via Triple protocol generic calls, group/version wildcard matching, and TLS API redesign.


## ️ Tools

The `tools/` directory and the `dubbogo/tools` repository provide several utilities to streamline your Dubbo-Go development experience.

### [dubbo-go-schema](https://github.com/apache/dubbo-go/tree/main/tools/dubbo-go-schema)

A tool that provides JSON Schema for Dubbo-Go configuration files. This simplifies the configuration process by enabling editor assistance.

**Features:**

* **Intelligent Assistance:** Enables code completion, hints, and real-time validation for configuration files in supported IDEs.
* **Simplified Configuration:** Helps you write valid and accurate configurations with ease.

For usage details, see the [dubbo-go-schema README](./tools/dubbo-go-schema/README.md).


### [dubbogo-cli](https://github.com/apache/dubbo-go/tree/main/tools/dubbogo-cli)

A comprehensive command-line tool for bootstrapping, managing, and debugging your Dubbo-Go applications.

**Features:**

* **Project Scaffolding:** Quickly create new application templates.
* **Tool Management:** Install and manage essential development tools.
* **Interface Debugging:** Provides commands to debug your services.

- [dubbogo-cli](https://github.com/apache/dubbo-go/tree/main/tools/dubbogo-cli)
- [dubbogo-cli-v2](https://github.com/dubbogo/tools/tree/master/cmd/dubbogo-cli-v2)


### [protoc-gen-go-triple](https://github.com/dubbogo/protoc-gen-go-triple)

A `protoc` plugin that generates golang code for the Triple protocol from your `.proto` (Protocol Buffer) definition files.

**Features:**

* **Code Generation:** Generates golang client and server stubs for the Triple protocol.
* **Seamless Integration:** Works alongside the official `protoc-gen-go` to produce both Protobuf message code (`.pb.go`) and Triple interface code (`.triple.go`).

*Note: This tool replaces the deprecated [protoc-gen-dubbo3grpc](https://github.com/dubbogo/tools/tree/master/cmd/protoc-gen-dubbo3grpc) and deprecated [protoc-gen-go-triple](https://github.com/dubbogo/tools/tree/master/cmd/protoc-gen-go-triple).*

For usage details, see the [protoc-gen-go-triple README](https://github.com/dubbogo/protoc-gen-go-triple).


### [imports-formatter](https://github.com/dubbogo/tools?tab=readme-ov-file#imports-formatter)

This is a plugin for dubbo-go developers. A code formatting tool that organizes golang `import` blocks according to the Dubbo-Go community style guide.

**Features:**

* **Automatic Formatting:** Splits `import` statements into three distinct groups: golang standard library, third-party libraries, and internal project packages.
* **Code Consistency:** Enforces a clean and consistent import style across the codebase.

For usage details, see the [imports-formatter README](https://github.com/dubbogo/tools?tab=readme-ov-file#imports-formatter).

## Ecosystem
- [dubbo-go-samples](https://github.com/apache/dubbo-go-samples)
- [dubbo-go-pixiu which acting as a proxy to solve Dubbo multi-language interoperability](https://github.com/apache/dubbo-go-pixiu)
- [Interoperability with Dubbo Java](https://cn.dubbo.apache.org/zh-cn/overview/mannual/golang-sdk/tutorial/interop-dubbo/)
- [Protoc-gen-go-triple](https://github.com/dubbogo/protoc-gen-go-triple/)
- [Console](https://github.com/apache/dubbo-kubernetes), under development

## Documentation

- Official Website: https://dubbo.apache.org/
- Dubbo-go Documentation: https://cn.dubbo.apache.org/zh-cn/overview/mannual/golang-sdk/
- CHANGELOG: https://github.com/apache/dubbo-go/blob/main/CHANGELOG.md

## Community

- GitHub Issues: https://github.com/apache/dubbo-go/issues
- Mailing Lists: https://dubbo.apache.org/en/community/

Contributions, issues, and discussions are welcome. Please visit [CONTRIBUTING](./CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## Contact

Join our discussion group through Ding talk, WeChat, or Discord.

discord https://discord.gg/C5ywvytg
![invite.png](./doc/imgs/invite.png)


## [User List](https://github.com/apache/dubbo-go/issues/2)

If you are using [apache/dubbo-go](https://github.com/apache/dubbo-go) and think that it helps you or want to contribute code for Dubbo-go, please add your company to [the user list](https://github.com/apache/dubbo-go/issues/2) to let us know your needs.


<div>
<table>
  <tbody>
  <tr></tr>
    <tr>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-beike.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-gaode.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-eht.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="https://github.com/mosn" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-mosn.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-pdd.png">
        </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td align="center"  valign="middle">
        <a href="http://www.j.cn" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-jd.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-kaikele.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-sohu.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="https://www.zto.com" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-zto.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-tianyi.png">
        </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td align="center"  valign="middle">
        <a href="http://www.mgtv.com" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-mgtv.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-vivo.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="http://www.ruubypay.com" target="_blank">
           <img width="222px"  src="./doc/imgs/usecase-tuya.png">
        </a>
      </td>
      <td align="center"  valign="middle">
          <a href="https://www.dingtalk.com" target="_blank">
             <img width="222px"  src="./doc/imgs/usecase-xiecheng.png">
          </a>
      </td>
      <td align="center"  valign="middle">
          <a href="https://www.autohome.com.cn" target="_blank">
             <img width="222px"  src="./doc/imgs/usecase-autohome.png">
          </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td align="center"  valign="middle">
        <a href="https://www.mi.com/" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-mi.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="https://opayweb.com/" target="_blank">
          <img width="222px"  src="./doc/imgs/usecase-zonghengwenxue.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="" target="_blank">
          <img width="222px" src="./doc/imgs/usecase-tiger-brokers.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="" target="_blank">
          <img width="222px" src="./doc/imgs/usecase-zhanzhang.png" >
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="" target="_blank">
          <img width="222px" src="./doc/imgs/usecase-genshuixue.png" >
        </a>
      </td>
    </tr>
    <tr></tr>
  </tbody>
</table>
</div>

[See more user cases](https://github.com/apache/dubbo-go/issues/2)

## License

Apache Dubbo-go software is licensed under the Apache License Version 2.0. See the [LICENSE](./LICENSE) file for details.