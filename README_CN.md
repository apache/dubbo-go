# Apache Dubbo-go

[![Build Status](https://github.com/apache/dubbo-go/workflows/CI/badge.svg)](https://travis-ci.org/apache/dubbo-go)
[![codecov](https://codecov.io/gh/apache/dubbo-go/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-go)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/apache/dubbo-go?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-go)](https://goreportcard.com/report/github.com/apache/dubbo-go)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

---

[English 🇺🇸](./README.md)

Apache Dubbo Go 语言实现，架起 Java 和 Golang 之间的桥梁，与 gRPC/Dubbo/SpringCloud 生态互联互通，带领 Java 生态享受云原生时代的技术红利。请访问[Dubbo 官网](https://dubbo.apache.org/zh/docs/languages/golang/)查看快速开始和文档。

## RPC 调用

![](https://dubbogo.github.io/img/dubbogo-3.0-invocation.png)

Dubbo-go 生态覆盖多种网络协议：Triple、Dubbo、JSONRPC、gRPC、HTTP、HTTP2 等。

- Triple 协议是 Dubbo3 生态主推的协议，是基于 gRPC 的扩展协议，底层为HTTP2，可与 gRPC 服务互通。**相当于在 gRPC 可靠的传输基础上，增加了 Dubbo 的服务治理能力。**
- Dubbo 协议是 Dubbo 生态的传统协议，dubbo-go 支持的 dubbo 协议与dubbo2.x 版本兼容，是 Go 语言和旧版本 Dubbo 服务互通的不错选择。
- 我们支持通过[貔貅](https://github.com/apache/dubbo-go-pixiu)网关暴露 Triple/Dubbo 协议到集群外部，调用者可以直接通过HTTP 协议调用 Dubbo-go 服务。

## 服务治理

![](https://dubbogo.github.io/img/devops.png)

- **注册中心**:

  支持 Nacos（阿里开源） 、Zookeeper、ETCD、Consul、Polaris-mesh（腾讯开源） 等服务注册中间件，并拥有可扩展能力。我们也会根据用户使用情况，进一步扩展出用户需要的实现。

- **配置中心**

  开发者可以使用Nacos、Zookeeper 进行框架/用户的配置的发布和拉取。

- **集群策略**: Failover, [Failfast](https://github.com/apache/dubbo-go/pull/140), [Failsafe/Failback](https://github.com/apache/dubbo-go/pull/136), [Available](https://github.com/apache/dubbo-go/pull/155), [Broadcast](https://github.com/apache/dubbo-go/pull/158), [Forking](https://github.com/apache/dubbo-go/pull/161) 等

- **负载均衡策略**: [柔性服务](https://github.com/apache/dubbo-go/pull/1649), Random, [RoundRobin](https://github.com/apache/dubbo-go/pull/66), [LeastActive](https://github.com/apache/dubbo-go/pull/65), [ConsistentHash](https://github.com/apache/dubbo-go/pull/261) 等

- [**过滤器**](./filter): Echo, Hystrix, Token, AccessLog, TpsLimiter, ExecuteLimit, Generic, Auth/Sign, Metrics, Tracing, Active, Seata, Sentinel 等

- **泛化调用**

- **监控**: [Prometheus](https://github.com/apache/dubbo-go/pull/342)

- **链路追踪**: Jaeger, Zipkin

- **路由器**: [Dubbo3 Router](https://github.com/apache/dubbo-go/pull/1187)

## 快速开始

- Dubbo-go 快速开始: [中文 🇨🇳](https://dubbogo.github.io/zh-cn/docs/user/quickstart/3.0/quickstart_triple.html), [English 🇺🇸](https://dubbogo.github.io/en-us/docs/user/quickstart/3.0/quickstart_triple.html)
- [Dubbo-go 样例](https://github.com/apache/dubbo-go-samples): 该项目提供了一系列的样例，以展示Dubbo-go的每一项特性以及帮助你将Dubbo-go集成到你的系统中。
- [Dubbo-go 百科](https://github.com/apache/dubbo-go/wiki)

## 工具

  * [imports-formatter](https://github.com/dubbogo/tools/blob/master/cmd/imports-formatter/main.go) dubbo-go 工程 import 代码块格式化工具
  * [dubbo-go-cli](https://github.com/dubbogo/tools/blob/master/cmd/dubbogo-cli/main.go) dubbo-go 命令行工具、支持展示服务、发起服务调用、定义 dubbogo 服务 struct 等功能、生成 hessian.POJO 方法体
  * [dubbo-go-cli-v2](https://github.com/dubbogo/tools/blob/master/cmd/dubbogo-cli-v2/main.go) 新 dubbo-go 命令行工具, 支持创建 demo、从注册中心获取服务列表以及兼容 [dubbo-go-cli](https://github.com/dubbogo/tools/blob/master/cmd/dubbogo-cli/main.go) 的所有功能
  * [protoc-gen-go-triple](https://github.com/dubbogo/tools/blob/master/cmd/protoc-gen-go-triple/main.go) tripe 协议 pb 生成工具
  * [protoc-gen-dubbo3grpc](https://github.com/dubbogo/tools/blob/master/cmd/protoc-gen-dubbo3grpc/main.go) dubbo3 grpc 生成工具

如果想知道 dubbogo 工具集更多更详细的使用细节，请访问 https://github.com/dubbogo/tools 并仔细阅读其 raedme。

## Intellij 插件

* Windows: File > Settings > Plugins > Browse repositories... > 搜索 "Dubbo Go" > Install Plugin
* MacOS: Preferences > Settings > Plugins > Browse repositories... > 搜索 "Dubbo Go" > Install Plugin
* 手动安装:
  * 下载[最新版插件](https://plugins.jetbrains.com/plugin/18581-dubbo-go) 并且尝试手动安装， Preferences > Plugins > Install plugin from disk...
  * 插件市场[https://plugins.jetbrains.com/plugin/18581-dubbo-go](https://plugins.jetbrains.com/plugin/18581-dubbo-go)


### 功能特性
|      特性      | IDEA | GoLand |
|:------------:|:----:|:------:|
| Hessian2 生成器 |  ✅️  |   ✅️   |
|   创建项目/模块    |  ✅️  |   ✅️   |

#### 创建新项目
| 项目/模块模板 | 进度  |
|:-------:|:---:|
|  官方例子   | ✅️  |
|   空项目   | ✅️  |

##### 空项目模板中间件
| 中间件类型 |                 可选模块                  | 是否支持 |
|:-----:|:-------------------------------------:|:----:|
| 网络服务  |    [Gin](github.com/gin-gonic/gin)    |  ✅️  |
| 内存缓存  | [Redis](github.com/go-redis/redis/v8) |  ✅️  |
|  数据库  |         [Gorm](gorm.io/gorm)          |  ✅️  |

如果想知道 dubbogo 工具集更多更详细的使用细节，请访问 [https://gitee.com/changeden/intellij-plugin-dubbo-go-generator](https://gitee.com/changeden/intellij-plugin-dubbo-go-generator) 并仔细阅读其 raedme。

## 生态

* [Dubbo Ecosystem Entry](https://github.com/apache?utf8=%E2%9C%93&q=dubbo&type=&language=) - Apache Dubbo 群组的相关开源项目
* [dubbo-go-pixiu](https://github.com/apache/dubbo-go-pixiu) - 动态高性能 API 网关，支持 Dubbo 和 Http 等多种协议
* [dubbo-go-samples](https://github.com/apache/dubbo-go-samples) - Dubbo-go 项目案例
* [dubbo-getty](https://github.com/apache/dubbo-getty) - Netty 风格的异步网络 IO 库，支持 tcp、udp 和 websocket 等协议
* [triple](https://github.com/dubbogo/triple) - 基于 HTTP2 的 Dubbo-go 3.0 协议网络库
* [dubbo-go-hessian2](https://github.com/apache/dubbo-go-hessian2) - 供 Dubbo-go 使用的 hessian2 库
* [gost](https://github.com/dubbogo/gost) - 供 Dubbo-go 使用的基础代码库

## 如何贡献

请访问[CONTRIBUTING](./CONTRIBUTING.md)来了解如何提交更新以及贡献工作流。

## 报告问题

请使用[bug report 模板](issues/new?template=bug-report.md)报告错误，使用[enhancement 模版](issues/new?template=enhancement.md)提交改进建议。

## 联系

- [钉钉群](https://www.dingtalk.com/): 23331795

## [用户列表](https://github.com/apache/dubbo-go/issues/2)

若你正在使用 [apache/dubbo-go](https://github.com/apache/dubbo-go) 且认为其有用或者想对其做改进，请添列贵司信息于 [用户列表](https://github.com/apache/dubbo-go/issues/2)，以便我们知晓。

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
[查看更多用户示例](https://github.com/apache/dubbo-go/issues/2)

## 许可证

Apache Dubbo-go使用Apache许可证2.0版本，请参阅LICENSE文件了解更多。
