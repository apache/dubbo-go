# Go for Apache Dubbo [English](./README.md) #

[![Build Status](https://travis-ci.com/dubbo/go-for-apache-dubbo.svg?branch=master)](https://travis-ci.com/dubbo/go-for-apache-dubbo)
[![codecov](https://codecov.io/gh/dubbo/go-for-apache-dubbo/branch/master/graph/badge.svg)](https://codecov.io/gh/dubbo/go-for-apache-dubbo)

---
Apache Dubbo Go 语言实现

## 证书 ##

Apache License, Version 2.0

## 发布日志 ##

[v1.0.0 - 2019年5月29日 兼容dubbo v2.6.5 版本](https://github.com/dubbo/go-for-apache-dubbo/releases/tag/v1.0.0)

## 工程架构 ##

基于dubbo的extension模块和分层的代码设计(包括 protocol layer, registry layer, cluster layer, config 等等)。我们的目标是：你可以对这些分层接口进行新的实现，并通过调用 extension 模块的“ extension.SetXXX ”方法来覆盖 dubbo-go [同 go-for-apache-dubbo ]的默认实现，以完成自己的特殊需求而无需修改源代码。同时，欢迎你为社区贡献有用的拓展实现。

![框架设计](https://raw.githubusercontent.com/wiki/dubbo/dubbo-go/dubbo-go%E4%BB%A3%E7%A0%81%E5%88%86%E5%B1%82%E8%AE%BE%E8%AE%A1.png)

关于详细设计请阅读 [code layered design](https://github.com/dubbo/go-for-apache-dubbo/wiki/dubbo-go-V1.0-design)

## 功能列表 ##

实现列表:

- Role: Consumer(√), Provider(√)
- Transport: HTTP(√), TCP(√)
- Codec: JsonRPC v2(√), Hessian v2(√)
- Registry: ZooKeeper(√)
- Cluster Strategy: Failover(√)
- Load Balance: Random(√)
- Filter: Echo Health Check(√)

开发中列表:

- Cluster Strategy: Failfast/Failsafe/Failback/Forking
- Load Balance: RoundRobin/LeastActive/ConsistentHash
- Filter: TokenFilter/AccessLogFilter/CountFilter/ActiveLimitFilter/ExecuteLimitFilter/GenericFilter/TpsLimitFilter
- Registry: etcd/k8s/consul

任务列表:

- routing rule (dubbo v2.6.x)
- metrics (dubbo v2.7.x) waiting dubbo's quota
- dynamic configuration center & metadata center (dubbo v2.7.x)
- tracing (dubbo ecosystem)

你可以通过访问 [roadmap](https://github.com/dubbo/go-for-apache-dubbo/wiki/Roadmap) 知道更多关于 dubbo-go 的信息

## 快速开始 ##

这个子目录下的例子展示了如何使用 dubbo-go 。请仔细阅读 [examples/README.md](https://github.com/dubbo/go-for-apache-dubbo/blob/develop/examples/README.md) 学习如何处理配置并编译程序。

## 性能测试 ##

性能测试项目是 [go-for-apache-dubbo-benchmark](https://github.com/dubbogo/go-for-apache-dubbo-benchmark)

关于 dubbo-go 性能测试报告，请阅读 [dubbo benchmarking report](https://github.com/dubbo/go-for-apache-dubbo/wiki/pressure-test-report-for-dubbo) & [jsonrpc benchmarking report](https://github.com/dubbo/go-for-apache-dubbo/wiki/pressure-test-report-for-jsonrpc)

## [User List](https://github.com/dubbo/go-for-apache-dubbo/issues/2)

![ctrip](https://pic.c-ctrip.com/common/c_logo2013.png)

## Stargazers

[![Stargazers over time](https://starchart.cc/dubbo/dubbo-go.svg)](https://starchart.cc/dubbo/dubbo-go)

