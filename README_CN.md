# Go for Apache Dubbo [English](./README.md) #

[![Build Status](https://travis-ci.com/dubbo/go-for-apache-dubbo.svg?branch=master)](https://travis-ci.com/dubbo/go-for-apache-dubbo)
[![codecov](https://codecov.io/gh/dubbo/go-for-apache-dubbo/branch/master/graph/badge.svg)](https://codecov.io/gh/dubbo/go-for-apache-dubbo)

---
Apache Dubbo Go 语言实现

## 证书 ##

Apache License, Version 2.0

## 代码设计 ##
基于dubbo的分层代码设计( protocol layer , registry layer , cluster layer , config 等等)，你可以通过调用“ extension.SetXXX ”拓展这些分层替换 dubbo 的默认拓展来实现你的需求。欢迎贡献你觉得好的拓展。

![框架设计](https://raw.githubusercontent.com/wiki/dubbo/dubbo-go/dubbo-go%E9%87%8D%E6%9E%84-%E6%A1%86%E6%9E%B6%E8%AE%BE%E8%AE%A1.jpg)

关于详细设计请阅读 [code layered design](https://github.com/dubbo/go-for-apache-dubbo/wiki/dubbo-go-V2.6-design)

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
- monitoring (dubbo v2.6.x)
- dynamic configuration (dubbo v2.7.x)
- metrics (dubbo v2.7.x) waiting dubbo's quota

你可以通过访问 [roadmap](https://github.com/dubbo/go-for-apache-dubbo/wiki/Roadmap) 知道更多关于dubbo-go【同 go-for-apache-dubbo 】的信息

## 快速开始 ##

这个子目录下的例子展示了如何使用 dubbo-go 。请仔细阅读 [examples/README.md](https://github.com/dubbo/go-for-apache-dubbo/blob/develop/examples/README.md) 学习如何处理配置并编译程序。

## 性能测试 ##

性能测试项目是 [go-for-apache-dubbo-benchmark](https://github.com/dubbogo/go-for-apache-dubbo-benchmark)

关于 dubbo-go 性能测试报告，请阅读 [dubbo benchmarking report](https://github.com/dubbo/go-for-apache-dubbo/wiki/pressure-test-report-for-dubbo) & [jsonrpc benchmarking report](https://github.com/dubbo/go-for-apache-dubbo/wiki/pressure-test-report-for-jsonrpc)

## [User List](https://github.com/dubbo/go-for-apache-dubbo/issues/2)

![ctrip](https://pic.c-ctrip.com/common/c_logo2013.png)
