# Apache Dubbo-go

[![Build Status](https://github.com/apache/dubbo-go/workflows/CI/badge.svg)](https://travis-ci.org/apache/dubbo-go)
[![codecov](https://codecov.io/gh/apache/dubbo-go/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-go)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/apache/dubbo-go?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-go)](https://goreportcard.com/report/github.com/apache/dubbo-go)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

---

[中文 🇨🇳](./README_CN.md)

Apache Dubbo-go, a Dubbo implementation written in Golang, is born to bridge the gap between Java/Dubbo and Go/X. Please visit our [Dubbo official website](https://dubbo.apache.org/zh/docs/languages/golang/) for the quick start and documentation.

## RPC Invocation

![](https://dubbogo.github.io/img/dubbogo-3.0-invocation.png)

Dubbo-go has supported many RPC protocol, like Triple, Dubbo JSONRPC, gRPC, HTTP, HTTP2.

- Triple is the supported protocol of Dubbo3 ecology, and is gRPC extended protocol based on HTTP2, which is compatible with gRPC service.In other words, **on the basis of gRPC's reliable invocation, it adds Dubbo's service governance capability.**
- Dubbo protocol  is tradition Dubbo ecology protocol, which is capitable with dubbo 2.x, and is a good choice for cross-language invocation between GO and Java old service.
- HTTP support：As you can see in the figure above, you can invoke Triple/Dubbo service using HTTP protocol through [dubbo-go-pixiu](https://github.com/apache/dubbo-go-pixiu) gateway.

## Service Governance Capability.

![](https://dubbogo.github.io/img/devops.png)

- **Registry**: Nacos, Zookeeper, ETCD, Polaris-mesh, Consul.
- **ConfigCenter**: Nacos, Zookeeper
- **Cluster Strategy**: Failover, [Failfast](https://github.com/apache/dubbo-go/pull/140), [Failsafe/Failback](https://github.com/apache/dubbo-go/pull/136), [Available](https://github.com/apache/dubbo-go/pull/155), [Broadcast](https://github.com/apache/dubbo-go/pull/158), [Forking](https://github.com/apache/dubbo-go/pull/161)
- **Load Balance**: [AdaptiveService](https://github.com/apache/dubbo-go/pull/1649), Random, [RoundRobin](https://github.com/apache/dubbo-go/pull/66), [LeastActive](https://github.com/apache/dubbo-go/pull/65), [ConsistentHash](https://github.com/apache/dubbo-go/pull/261)
- [**Filter**](./filter): Echo, Hystrix, Token, AccessLog, TpsLimiter, ExecuteLimit, Generic, Auth/Sign, Metrics, Tracing, Active, Seata, Sentinel
- **[Generic Invoke](https://github.com/apache/dubbo-go/pull/122)**
- **Monitor**:  [Prometheus](https://github.com/apache/dubbo-go/pull/342)
- **Tracing**:  Jaeger, Zipkin
- **Router**: [Dubbo3 Router](https://github.com/apache/dubbo-go/pull/1187)

## Getting started

- Dubbo-go Quick Start: [中文 🇨🇳](https://dubbogo.github.io/zh-cn/docs/user/quickstart/3.0/quickstart_triple.html), [English 🇺🇸](https://dubbogo.github.io/en-us/docs/user/quickstart/3.0/quickstart_triple.html)
- [Dubbo-go Samples](https://github.com/apache/dubbo-go-samples): The project gives a series of samples that show each feature available for Dubbo-go and help you know how to integrate Dubbo-go with your system.
- [Dubbo-go Benchmark](https://github.com/dubbogo/dubbo-go-benchmark)
- [Dubbo-go Wiki](https://github.com/apache/dubbo-go/wiki)

## tools

  * [imports-formatter](https://github.com/dubbogo/tools/blob/master/cmd/imports-formatter/main.go) formatting dubbo-go project import code block.
  * [dubbo-go-cli](https://github.com/dubbogo/tools/blob/master/cmd/dubbogo-cli/main.go) dubbo-go command line tools, by which you can define your own request pkg and gets rsp struct of your server, test your service as telnet and generate hessian.POJO register method body.
  * [dubbo-go-cli-v2](https://github.com/dubbogo/tools/blob/master/cmd/dubbogo-cli-v2/main.go) new dubbo-go line tools, you can get services from register center, create a demo and has the same features with [dubbo-go-cli](https://github.com/dubbogo/tools/blob/master/cmd/dubbogo-cli/main.go).
  * [protoc-gen-go-triple](https://github.com/dubbogo/tools/blob/master/cmd/protoc-gen-go-triple/main.go) tripe protocol pb file generation tool.
  * [protoc-gen-dubbo3grpc](https://github.com/dubbogo/tools/blob/master/cmd/protoc-gen-dubbo3grpc/main.go) dubbo3 grpc pb file generation tool.


If you want to know more about dubbogo tools, please visit https://github.com/dubbogo/tools and read its readme carefully.

## Intellij Plugin

* Windows: File > Settings > Plugins > Browse repositories... > Search for "Dubbo Go" > Install Plugin
* MacOS: Preferences > Settings > Plugins > Browse repositories... > Search for "Dubbo Go" > Install Plugin
* Manually:
    * Download the [latest release](https://plugins.jetbrains.com/plugin/18581-dubbo-go) and install it manually using Preferences > Plugins > Install plugin from disk...
    * From official jetbrains store from download


### Feature
|      Feature       | IDEA | GoLand |
|:------------------:|:----:|:------:|
| Hessian2 Generator |  ✅️  |   ✅️   |
| New Project/Module |  ✅️  |   ✅️   |

#### Project/Module Template
| Project/Module Template | Progress |
|:-----------------------:|:--------:|
|         Sample          |    ✅️    |
|      Empty Project      |    ✅️    |

##### Empty Project Template Middleware
|  Middleware  |                Module                 | Support |
|:------------:|:-------------------------------------:|:-------:|
| Web Service  |    [Gin](github.com/gin-gonic/gin)    |   ✅️    |
| Memory Cache | [Redis](github.com/go-redis/redis/v8) |   ✅️    |
|   Database   |         [Gorm](gorm.io/gorm)          |   ✅️    |


If you want to know more about dubbogo Intellij Plugin, please visit [https://gitee.com/changeden/intellij-plugin-dubbo-go-generator](https://gitee.com/changeden/intellij-plugin-dubbo-go-generator) and read its readme carefully.

## Dubbo-go ecosystem

* [Dubbo Ecosystem Entry](https://github.com/apache?utf8=%E2%9C%93&q=dubbo&type=&language=) - A GitHub group `dubbo` to gather all Dubbo relevant projects not appropriate in [apache](https://github.com/apache) group yet.
* [dubbo-go-pixiu](https://github.com/apache/dubbo-go-pixiu) - A dynamic, high-performance API gateway solution for Dubbo and Http services.
* [dubbo-go-samples](https://github.com/apache/dubbo-go-samples) - Samples for Apache Dubbo-go.
* [dubbo-getty](https://github.com/apache/dubbo-getty) - A netty like asynchronous network I/O library which supports tcp/udp/websocket network protocol.
* [triple](https://github.com/dubbogo/triple) - A golang network package that based on http2, used by Dubbo-go 3.0.
* [dubbo-go-hessian2](https://github.com/apache/dubbo-go-hessian2) - A golang hessian library used by Apache/dubbo-go.
* [gost](https://github.com/dubbogo/gost) - A go sdk for Apache Dubbo-go.


## Contributing

Please visit [CONTRIBUTING](./CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## Reporting bugs

Please use the [bug report template](issues/new?template=bug-report.md) to report bugs, use the [enhancement template](issues/new?template=enhancement.md) to provide suggestions for improvement.

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
          <img width="222px" src="https://github.com/seven-tan/static/blob/main/logo.png?raw=true" >
        </a>
      </td>
    </tr>
    <tr></tr>
  </tbody>
</table>
</div>

[See more user cases](https://github.com/apache/dubbo-go/issues/2)

## License

Apache Dubbo-go software is licenced under the Apache License Version 2.0. See the LICENSE file for details.
