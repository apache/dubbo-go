# Go for Apache Dubbo [中文](./README_CN.md) #

[![Build Status](https://travis-ci.com/dubbo/go-for-apache-dubbo.svg?branch=master)](https://travis-ci.com/dubbo/go-for-apache-dubbo)
[![codecov](https://codecov.io/gh/dubbo/go-for-apache-dubbo/branch/master/graph/badge.svg)](https://codecov.io/gh/dubbo/go-for-apache-dubbo)

---
Apache Dubbo Go Implementation.

## License

Apache License, Version 2.0

## Code design ##
Based on dubbo's layered code design (protocol layer,registry layer,cluster layer,config layer and so on), you can achieve your needs by invoking 'extension.SetXXX' to extend these layered interfaces instead of modifying dubbo-go[same go-for-apache-dubbo]'s source code. And welcome to contribute your awesome extension.

![frame design](https://raw.githubusercontent.com/wiki/dubbo/dubbo-go/dubbo-go%E9%87%8D%E6%9E%84-%E6%A1%86%E6%9E%B6%E8%AE%BE%E8%AE%A1.jpg)

About detail design please refer to [code layered design](https://github.com/dubbo/go-for-apache-dubbo/wiki/dubbo-go-V2.6-design)

## Feature list ##

Finished List:

- Role: Consumer(√), Provider(√)
- Transport: HTTP(√), TCP(√)
- Codec: JsonRPC v2(√), Hessian v2(√)
- Registry: ZooKeeper(√)
- Cluster Strategy: Failover(√)
- Load Balance: Random(√)
- Filter: Echo Health Check(√)

Working List:

- Cluster Strategy: Failfast/Failsafe/Failback/Forking
- Load Balance: RoundRobin/LeastActive/ConsistentHash
- Filter: TokenFilter/AccessLogFilter/CountFilter/ActiveLimitFilter/ExecuteLimitFilter/GenericFilter/TpsLimitFilter
- Registry: etcd/k8s/consul

Todo List:

- routing rule (dubbo v2.6.x)
- monitoring (dubbo v2.6.x)
- dynamic configuration (dubbo v2.7.x)
- metrics (dubbo v2.7.x) waiting dubbo's quota

You can know more about dubbo-go by its [roadmap](https://github.com/dubbo/go-for-apache-dubbo/wiki/Roadmap).

## Quick Start

The subdirectory examples shows how to use dubbo-go. Please read the [examples/README.md](https://github.com/dubbo/go-for-apache-dubbo/blob/develop/examples/README.md) carefully to learn how to dispose the configuration and compile the program.

## Benchmark

Benchmark project please refer to [go-for-apache-dubbo-benchmark](https://github.com/dubbogo/go-for-apache-dubbo-benchmark)

About dubbo-go benchmarking report, please refer to [dubbo benchmarking report](https://github.com/dubbo/go-for-apache-dubbo/wiki/pressure-test-report-for-dubbo) & [jsonrpc benchmarking report](https://github.com/dubbo/go-for-apache-dubbo/wiki/pressure-test-report-for-jsonrpc)

## [User List](https://github.com/dubbo/go-for-apache-dubbo/issues/2)

![ctrip](https://pic.c-ctrip.com/common/c_logo2013.png)
