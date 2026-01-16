# Apache Dubbo for Golang

[![CI](https://github.com/apache/dubbo-go/workflows/CI/badge.svg)](https://github.com/apache/dubbo-go/actions)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/apache/dubbo-go/v3?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-go)](https://goreportcard.com/report/github.com/apache/dubbo-go)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

---

[English](README.md) | 中文

**Apache Dubbo-go** 是一款高性能、功能丰富的微服务框架。作为 Apache Dubbo 生态的 Go 语言实现，它充分利用 Golang 的并发特性，助力开发者在云原生时代构建扩展性强、可靠性高的分布式应用。

在全新的 **v3.3.x** 系列版本中，Dubbo-go 已从传统的 RPC 框架演进为**云原生智能微服务治理平台**，引入了深度的 AI 集成与 Proxyless Mesh（无代理网格）能力。

您可以访问 [官网](https://dubbo.apache.org/) 以获取更多信息。

## 快速开始

### 环境准备

* Go 1.24 或更高版本（建议使用最新版本以获得最佳性能与兼容性）。

### 安装

使用以下命令安装 Dubbo-go：

```bash
go get dubbo.apache.org/dubbo-go/v3@latest

```

### 快速示例

通过我们的 [快速开始 (Helloworld)](https://github.com/apache/dubbo-go-samples/tree/main/helloworld) 示例，您可以在 5 分钟内掌握如何开发一个 RPC 应用。

过程如下方代码所示，非常简单：您使用 Protobuf 定义一个服务，提供您自己的服务实现，将其注册到服务器，然后启动服务器。

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

服务器启动并运行后，您可以通过 cURL 调用您的服务：

```shell
curl \
    --header "Content-Type: application/json" \
    --data '{"name": "Dubbo"}' \
    http://localhost:20000/greet.GreetService/Greet
```

或者，您也可以启动一个标准的 dubbo-go 客户端来调用该服务：

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

有关详细用法，请参阅[示例](https://github.com/apache/dubbo-go-samples)。接下来，请访问官网，了解如何部署、监控和管理您的 dubbo-go 应用的流量。

## 功能特性

![dubbo-go-architecture](./doc/imgs/arc.png)

* **RPC 协议**: Triple (兼容 gRPC 且对 HTTP 友好)、Dubbo、JSONRPC、HTTP/2、HTTP/3 (实验性)。
* **服务发现**: Nacos、Zookeeper、Etcd、Polaris-mesh。
* **负载均衡**: 自适应、随机、轮询、最少活跃调用、一致性哈希。
* **流量管理**: 流量切分、超时设置、速率限制、金丝雀发布。
* **配置管理**: YAML 文件、动态配置（Nacos、Apollo、Zookeeper 等），支持基于指纹去重的优化版文件监听。
* **可观测性**: 指标（Prometheus）、追踪（OpenTelemetry v1.21.0+ 标准化 Span 名）、日志（完整的生命周期事件记录）。
* **高可用策略**: 故障转移 (Failover)、快速失败 (Failfast)、失败安全/失败自动恢复 (Failsafe/Failback)、广播 (Broadcast)、并行调用 (Forking)。
* **跨语言互通**: 通过 Triple 协议泛化调用、版本通配符匹配等技术，实现与 Java 版 Dubbo 的完美互通。

## ️ 工具生态

`tools/` 目录和 `dubbogo/tools` 仓库提供了一些实用工具，以简化您的 Dubbo-Go 开发体验。

### [dubbo-go-schema](https://github.com/apache/dubbo-go/tree/main/tools/dubbo-go-schema)

该工具为 Dubbo-Go 配置文件提供 JSON Schema，通过编辑器辅助功能简化配置过程。

**功能：**

* **智能辅助：** 在支持的 IDE 中为配置文件启用代码补全、提示和实时校验。
* **简化配置：** 帮助您轻松编写有效且准确的配置。

有关使用详情，请参阅 [dubbo-go-schema README](./tools/dubbo-go-schema/README.md)。

### [dubbogo-cli](https://github.com/apache/dubbo-go/tree/main/tools/dubbogo-cli)

一个功能全面的命令行工具，用于引导、管理和调试您的 Dubbo-Go 应用。

**功能：**

* **项目脚手架：** 快速创建新的应用模板。
* **工具管理：** 安装和管理必要的开发工具。
* **接口调试：** 提供命令来调试您的服务。

- [dubbogo-cli](https://github.com/apache/dubbo-go/tree/main/tools/dubbogo-cli)
- [dubbogo-cli-v2](https://github.com/dubbogo/tools/tree/master/cmd/dubbogo-cli-v2)


### [protoc-gen-go-triple](https://github.com/dubbogo/protoc-gen-go-triple)

一个 `protoc` 插件，用于从您的 `.proto` (Protocol Buffer) 定义文件为 Triple 协议生成 Golang 代码。

**功能：**

* **代码生成：** 为 Triple 协议生成 Golang 客户端和服务器存根（stub）。
* **无缝集成：** 与官方的 `protoc-gen-go` 协同工作，同时生成 Protobuf 消息代码（`.pb.go`）和 Triple 接口代码（`.triple.go`）。

*注意：该工具取代了已废弃的 [protoc-gen-dubbo3grpc](https://github.com/dubbogo/tools/tree/master/cmd/protoc-gen-dubbo3grpc) 和已废弃的 [protoc-gen-go-triple](https://github.com/dubbogo/tools/tree/master/cmd/protoc-gen-go-triple)。*

有关使用详情，请参阅 [protoc-gen-go-triple README](https://github.com/dubbogo/protoc-gen-go-triple)。

### [imports-formatter](https://github.com/dubbogo/tools?tab=readme-ov-file#imports-formatter)

这是一个面向 Dubbo-Go 开发者的插件，它是一款代码格式化工具，它根据 Dubbo-Go 社区的风格指南来组织 Golang 的 `import` 代码块。

**功能：**

* **自动格式化：** 将 `import` 语句分为三个独立的组：Golang 标准库、第三方库和项目内部包。
* **代码一致性：** 在整个代码库中强制执行干净、一致的导入风格。

有关使用详情，请参阅 [imports-formatter README](https://github.com/dubbogo/tools?tab=readme-ov-file#imports-formatter)。

## 生态系统

- [dubbo-go-samples](https://github.com/apache/dubbo-go-samples)
- [dubbo-go-pixiu: 作为代理解决 Dubbo 多语言互通问题](https://github.com/apache/dubbo-go-pixiu)
- [与 Dubbo Java 互通](https://cn.dubbo.apache.org/zh-cn/overview/mannual/golang-sdk/tutorial/interop-dubbo/)
- [Protoc-gen-go-triple](https://github.com/dubbogo/protoc-gen-go-triple/)
- [控制台 (Console)](https://github.com/apache/dubbo-kubernetes)，开发中

## 社区与文档

* **官方网站**: https://dubbo.apache.org/
* **官方文档**: https://cn.dubbo.apache.org/zh-cn/overview/mannual/golang-sdk/
* **问题反馈**: [GitHub Issues](https://github.com/apache/dubbo-go/issues)

关于提交补丁和贡献流程的详细信息，请访问 [CONTRIBUTING](./CONTRIBUTING.md)。

## 联系我们

通过钉钉、微信或 Discord 加入我们的讨论组。

discord https://discord.gg/C5ywvytg
![invite.png](./doc/imgs/invite.png)


## [用户列表](https://github.com/apache/dubbo-go/issues/2)

如果您正在使用 [apache/dubbo-go](https://github.com/apache/dubbo-go) 并且它对您有所帮助，或者您想为 Dubbo-go 贡献代码，请将您的公司添加到[用户列表](https://github.com/apache/dubbo-go/issues/2)，让我们了解您的需求。

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

[查看更多用户案例](https://github.com/apache/dubbo-go/issues/2)

## 许可证

Apache Dubbo-go 软件基于 Apache 许可证 2.0 版本进行许可。详情请参阅 [LICENSE](./LICENSE) 文件。
