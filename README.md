# Apache Dubbo for Golang

[![Build Status](https://github.com/apache/dubbo-go/workflows/CI/badge.svg)](https://travis-ci.org/apache/dubbo-go)
[![codecov](https://codecov.io/gh/apache/dubbo-go/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-go)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/apache/dubbo-go?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-go)](https://goreportcard.com/report/github.com/apache/dubbo-go)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

---

Apache Dubbo is an easy-to-use Web and RPC framework that provides multiple
language implementations(Go, [Java](https://github.com/apache/dubbo), [Rust](https://github.com/apache/dubbo-rust), [Node.js](https://github.com/apache/dubbo-js), [Web](https://github.com/apache/dubbo-js)) for communication, service discovery, traffic management,
observability, security, tools, and best practices for building enterprise-ready microservices.

Dubbo-go is the Go implementation of [triple protocol](https://dubbo.apache.org/zh-cn/overview/reference/protocols/triple-spec/)(a fully gRPC compatible and HTTP-friendly protocol) and the various features for building microservice architecture designed by Dubbo. 

Visit [the official website](https://dubbo.apache.org/) for more information.

## Getting started

You can learn how to develop a dubbo-go RPC application step by step in 5 minutes by following our [quick start]() demo. It's as simple as the code shown below, you define a service with Protobuf, provide your own service implementation, register it to a server, and start the server.

```go
func (s *GreeterServer) SayHello(ctx context.Context, in *greet.HelloRequest) (*greet.User, error) {
    return &greet.User{Name: "Hello " + in.Name, Id: "12345", Age: 21}, nil
}

func main() {
	s := config.NewServer()
	s.RegisterService(&GreeterServer{})
	s.Serve(net.Listen("tcp", ":50051"))
}
```

After the server is up and running, call your service via cURL:

```
curl -XPOST \
     -H 'Content-Type: application/json' \
     -H 'Micro-Endpoint: Helloworld.Greeting' \
     -d '{"name": "alice"}' \
      http://localhost:8080
```

See the [samples](https://github.com/apache/dubbo-go-samples) for detailed information on usage. Next, learn how to [deploy](), [monitor]() and [manage the traffic]() of your dubbo-go application.

## Features

<img src="https://dubbogo.github.io/img/devops.png" height="300px" display="display: block, margin: auto" />

- **RPC Protocols**: Triple(gRPC compatible and HTTP-friendly), Dubbo2(TCP)
- **Service Discovery**: Nacos, Zookeeper, Etcd, Polaris-mesh, Consul.
- **Load Balance**: Adaptive, Random, RoundRobin, LeastActive, ConsistentHash
- **Traffic Management**: traffic split, timeout, rate limiting, canary release
- **Configuration**: yaml file, dynamic configuration(Nacos, Zookeeper, etc.).
- **Observability**: metrics(Prometheus, Grafana) and tracing(Jaeger, Zipkin).
- **HA Strategy**: Failover, Failfast, Failsafe/Failback, Available, Broadcast, Forking

## Ecosystem
- [CLI]()
- [Console]()
- [Samples]()
- [Generator]()
- [Interoperability with Dubbo2 (tcp + hessian2)]()

## Contributing

Please visit [CONTRIBUTING](./CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## Contact

- [DingTalk Group](https://www.dingtalk.com/en): 23331795

## [User List](https://github.com/apache/dubbo-go/issues/2)

If you are using [apache/dubbo-go](https://github.com/apache/dubbo-go) and think that it helps you or want to contribute code for Dubbo-go, please add your company to [the user list](https://github.com/apache/dubbo-go/issues/2) to let us know your needs.


<div>
<table>
  <tbody>
  <tr></tr>
    <tr>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="https://pic.c-ctrip.com/common/c_logo2013.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="https://user-images.githubusercontent.com/52339367/84628582-80512200-af1b-11ea-945a-c6b4b9ad31f2.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/tuya.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="https://github.com/mosn" target="_blank">
          <img width="222px"  src="https://raw.githubusercontent.com/mosn/community/master/icons/png/mosn-labeled-horizontal.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="" target="_blank">
          <img width="222px"  src="https://festatic.estudy.cn/assets/xhx-web/layout/logo.png">
        </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td align="center"  valign="middle">
        <a href="http://www.j.cn" target="_blank">
          <img width="222px"  src="http://image.guang.j.cn/bbs/imgs/home/pc/icon_8500.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="https://www.genshuixue.com/" target="_blank">
          <img width="222px"  src="https://i.gsxcdn.com/0cms/d/file/content/2020/02/5e572137d7d94.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="http://www.51h5.com" target="_blank">
          <img width="222px"  src="https://fs-ews.51h5.com/common/hw_220_black.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="https://www.zto.com" target="_blank">
          <img width="222px"  src="https://fscdn.zto.com/fs8/M02/B2/E4/wKhBD1-8o52Ae3GnAAASU3r62ME040.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="https://www.icsoc.net/" target="_blank">
          <img width="222px"  src="https://help.icsoc.net/img/icsoc-logo.png">
        </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td align="center"  valign="middle">
        <a href="http://www.mgtv.com" target="_blank">
          <img width="222px"  src="https://ugc.hitv.com/platform_oss/F6077F1AA82542CDBDD88FD518E6E727.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="http://www.dmall.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/duodian.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="http://www.ruubypay.com" target="_blank">
           <img width="222px"  src="http://website.ruubypay.com/wifi/image/line5.png">
        </a>
      </td>
      <td align="center"  valign="middle">
          <a href="https://www.dingtalk.com" target="_blank">
             <img width="222px"  src="https://gw.alicdn.com/tfs/TB1HPATMrrpK1RjSZTEXXcWAVXa-260-74.png">
          </a>
      </td>
      <td align="center"  valign="middle">
          <a href="https://www.autohome.com.cn" target="_blank">
             <img width="222px"  src="https://avatars.githubusercontent.com/u/18279051?s=200&v=4">
          </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td align="center"  valign="middle">
        <a href="https://www.mi.com/" target="_blank">
          <img width="222px"  src="https://s02.mifile.cn/assets/static/image/logo-mi2.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="https://opayweb.com/" target="_blank">
          <img width="222px"  src="https://open.opayweb.com/static/img/logo@2x.35c6fe4c.jpg">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="http://www.zongheng.com/" target="_blank">
          <img width="222px" src="https://img.xmkanshu.com/u/202204/01/201253131.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://amap.com/" target="_blank">
          <img width="222px" src="https://github.com/seven-tan/static/blob/main/logo.png" >
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://chinaz.com/" target="_blank">
          <img width="222px" src="https://img.chinaz.com/2020/img/chinaz-logo.png" >
        </a>
      </td>
    </tr>
    <tr></tr>
  </tbody>
</table>
</div>

[See more user cases](https://github.com/apache/dubbo-go/issues/2)

## License

Apache Dubbo-go software is licensed under the Apache License Version 2.0. See the LICENSE file for details.
