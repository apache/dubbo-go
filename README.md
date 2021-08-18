# Apache Dubbo-go

[![Build Status](https://github.com/apache/dubbo-go/workflows/CI/badge.svg)](https://travis-ci.org/apache/dubbo-go)
[![codecov](https://codecov.io/gh/apache/dubbo-go/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-go)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/apache/dubbo-go?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-go)](https://goreportcard.com/report/github.com/apache/dubbo-go)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

---

[中文 🇨🇳](./README_CN.md)

Apache Dubbo-go, a Dubbo implementation written in Golang, is born to bridge the gap between Java and Golang. Please visit our [official website](https://dubbogo.github.io) for the quick start and documentation.

## Architecture

![dubbo go extend](https://dubbogo.github.io/img/doc/dubbo-go3.0-arch.jpg)

Dubbo-go has been implemented most layers of Dubbo, like protocol layer, registry layer, etc. An extension module is applied to Dubbo-go in order to achieve a more flexible architecture. Developers are allowed to implement a customized layer conformed to the layer interface and use them in Dubbo-go via `extension.Set` method without modifying the source code.

## Features

The features that are available for Dubbo-go are:

- **Role**: Consumer, Provider
- **Transport**: HTTP, TCP
- **Codec**: JsonRPC V2, Hessian V2, [Json for gRPC](https://github.com/apache/dubbo-go/pull/582), Protocol Buffers
- **Protocol**: Dubbo, [Triple](https://github.com/dubbogo/triple), JsonRPC V2, [gRPC](https://github.com/apache/dubbo-go/pull/311), [RESTful](https://github.com/apache/dubbo-go/pull/352)
- **Router**: [Dubbo3 Router](https://github.com/apache/dubbo-go/pull/1187)
- **Registry**: ZooKeeper, [etcd](https://github.com/apache/dubbo-go/pull/148), [Nacos](https://github.com/apache/dubbo-go/pull/151), [Consul](https://github.com/apache/dubbo-go/pull/121), [K8s](https://github.com/apache/dubbo-go/pull/400)
- **Dynamic Configure Center & Service Management Configurator**: Zookeeper, [Apollo](https://github.com/apache/dubbo-go/pull/250), [Nacos](https://github.com/apache/dubbo-go/pull/357)
- **Cluster Strategy**: Failover, [Failfast](https://github.com/apache/dubbo-go/pull/140), [Failsafe/Failback](https://github.com/apache/dubbo-go/pull/136), [Available](https://github.com/apache/dubbo-go/pull/155), [Broadcast](https://github.com/apache/dubbo-go/pull/158), [Forking](https://github.com/apache/dubbo-go/pull/161)
- **Load Balance**: Random, [RoundRobin](https://github.com/apache/dubbo-go/pull/66), [LeastActive](https://github.com/apache/dubbo-go/pull/65), [ConsistentHash](https://github.com/apache/dubbo-go/pull/261)
- [**Filter**](./filter): Echo, Hystrix, Token, AccessLog, TpsLimiter, ExecuteLimit, Generic, Auth/Sign, Metrics, Tracing, Active, Seata, Sentinel
- **Invoke**: [Generic Invoke](https://github.com/apache/dubbo-go/pull/122)
- **Monitor**: Opentracing API, [Prometheus](https://github.com/apache/dubbo-go/pull/342)
- **Tracing**: [For JsonRPC](https://github.com/apache/dubbo-go/pull/335), [For Dubbo](https://github.com/apache/dubbo-go/pull/344), [For gRPC](https://github.com/apache/dubbo-go/pull/397)
- **Metadata Center**: [Nacos(Local)](https://github.com/apache/dubbo-go/pull/522), [ZooKeeper(Local)](https://github.com/apache/dubbo-go/pull/633), [etcd(Local)](https://github.com/apache/dubbo-go/blob/9a5990d9a9c3d5e6633c0d7d926c156416bcb931/metadata/report/etcd/report.go), [Consul(Local)](https://github.com/apache/dubbo-go/pull/633), [ZooKeeper(Remoting)](https://github.com/apache/dubbo-go/pull/1161)
- **Tool**: [Dubbo-go-cli](https://github.com/dubbogo/tools)

## Getting started

### Install Dubbo-go v3

```
go get dubbo.apache.org/dubbo-go/v3
```

### Next steps

- [Dubbo-go Samples](https://github.com/apache/dubbo-go-samples): The project gives a series of samples that show each feature available for Dubbo-go and help you know how to integrate Dubbo-go with your system.
- Dubbo-go Quick Start: [中文 🇨🇳](https://dubbogo.github.io/zh-cn/docs/user/quickstart/3.0/quickstart.html), [English 🇺🇸](https://dubbogo.github.io/en-us/docs/user/quick-start.html)
- [Dubbo-go Benchmark](https://github.com/dubbogo/dubbo-go-benchmark)
- [Dubbo-go Wiki](https://github.com/apache/dubbo-go/wiki)

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

[See more user cases](https://github.com/apache/dubbo-go/issues/2)

## License

Apache Dubbo-go software is licenced under the Apache License Version 2.0. See the LICENSE file for details.
