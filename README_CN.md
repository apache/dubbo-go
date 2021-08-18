# Apache Dubbo-go

[![Build Status](https://github.com/apache/dubbo-go/workflows/CI/badge.svg)](https://travis-ci.org/apache/dubbo-go)
[![codecov](https://codecov.io/gh/apache/dubbo-go/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-go)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/apache/dubbo-go?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-go)](https://goreportcard.com/report/github.com/apache/dubbo-go)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

---

[English 🇺🇸](./README.md)

Apache Dubbo Go 语言实现，架起 Java 和 Golang 之间的桥梁，与 gRPC/Spring Cloud 生态互联互通，带领 Java 生态享受云原生时代的技术红利。请访问我们的[官方网站](https://dubbogo.github.io)查看快速开始和文档。

## 架构

![dubbo go extend](https://dubbogo.github.io/img/doc/dubbo-go3.0-arch.jpg)

Dubbo-go已经实现了Dubbo的大部分层级，包括协议层（protocol layer）、注册层（registry layer)）等等。在Dubbo-go中使用了拓展模块（extension module）以实现更灵活的系统架构，开发者可以根据层接口实现一个自定义的层，并在不改动源代码的前提下通过`extension.Set`方法将它应用到Dubbo-go中。

## 特性

Dubbo-go中已实现的特性：

- **角色**: Consumer, Provider
- **传输协议**: HTTP, TCP
- **序列化协议**: JsonRPC V2, Hessian V2, [Json for gRPC](https://github.com/apache/dubbo-go/pull/582), Protocol Buffers
- **协议**: Dubbo, [Triple](https://github.com/dubbogo/triple), JsonRPC V2, [gRPC](https://github.com/apache/dubbo-go/pull/311), [RESTful](https://github.com/apache/dubbo-go/pull/352)
- **路由器**: [Dubbo3 Router](https://github.com/apache/dubbo-go/pull/1187)
- **注册中心**: ZooKeeper, [etcd](https://github.com/apache/dubbo-go/pull/148), [Nacos](https://github.com/apache/dubbo-go/pull/151), [Consul](https://github.com/apache/dubbo-go/pull/121), [K8s](https://github.com/apache/dubbo-go/pull/400)
- **动态配置中心与服务治理配置器**: Zookeeper, [Apollo](https://github.com/apache/dubbo-go/pull/250), [Nacos](https://github.com/apache/dubbo-go/pull/357)
- **集群策略**: Failover, [Failfast](https://github.com/apache/dubbo-go/pull/140), [Failsafe/Failback](https://github.com/apache/dubbo-go/pull/136), [Available](https://github.com/apache/dubbo-go/pull/155), [Broadcast](https://github.com/apache/dubbo-go/pull/158), [Forking](https://github.com/apache/dubbo-go/pull/161)
- **负载均衡策略**: Random, [RoundRobin](https://github.com/apache/dubbo-go/pull/66), [LeastActive](https://github.com/apache/dubbo-go/pull/65), [ConsistentHash](https://github.com/apache/dubbo-go/pull/261)
- [**过滤器**](./filter): Echo, Hystrix, Token, AccessLog, TpsLimiter, ExecuteLimit, Generic, Auth/Sign, Metrics, Tracing, Active, Seata, Sentinel
- **调用**: [Generic Invoke](https://github.com/apache/dubbo-go/pull/122)
- **监控**: Opentracing API, [Prometheus](https://github.com/apache/dubbo-go/pull/342)
- **Tracing**: [For JsonRPC](https://github.com/apache/dubbo-go/pull/335), [For Dubbo](https://github.com/apache/dubbo-go/pull/344), [For gRPC](https://github.com/apache/dubbo-go/pull/397)
- **元数据中心**: [Nacos(Local)](https://github.com/apache/dubbo-go/pull/522), [ZooKeeper(Local)](https://github.com/apache/dubbo-go/pull/633), [etcd(Local)](https://github.com/apache/dubbo-go/blob/9a5990d9a9c3d5e6633c0d7d926c156416bcb931/metadata/report/etcd/report.go), [Consul(Local)](https://github.com/apache/dubbo-go/pull/633), [ZooKeeper(Remoting)](https://github.com/apache/dubbo-go/pull/1161)
- **工具**: [Dubbo-go-cli](https://github.com/dubbogo/tools)

## 开始

### 安装 Dubbo-go v3

```
go get dubbo.apache.org/dubbo-go/v3
```

### 下一步

- [Dubbo-go 样例](https://github.com/apache/dubbo-go-samples): 该项目提供了一系列的样例，以展示Dubbo-go的每一项特性以及帮助你将Dubbo-go集成到你的系统中。
- Dubbo-go 快速开始: [中文 🇨🇳](https://dubbogo.github.io/zh-cn/docs/user/quickstart/3.0/quickstart.html), [English 🇺🇸](https://dubbogo.github.io/en-us/docs/user/quick-start.html)
- [Dubbo-go 基准测试](https://github.com/dubbogo/dubbo-go-benchmark)
- [Dubbo-go 百科](https://github.com/apache/dubbo-go/wiki)

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
          <img width="222px"  src="https://oss.icsoc.net/icsoc-ekt-test-files/icsoc.png">
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
  </tbody>
</table>
</div>

[查看更多用户示例](https://github.com/apache/dubbo-go/issues/2)

## 许可证

Apache Dubbo-go使用Apache许可证2.0版本，请参阅LICENSE文件了解更多。
