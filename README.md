# Go for Apache Dubbo [中文](./README_CN.md) #

[![Build Status](https://travis-ci.com/dubbo/go-for-apache-dubbo.svg?branch=master)](https://travis-ci.com/dubbo/go-for-apache-dubbo)
[![codecov](https://codecov.io/gh/dubbo/go-for-apache-dubbo/branch/master/graph/badge.svg)](https://codecov.io/gh/dubbo/go-for-apache-dubbo)

---
Apache Dubbo Go Implementation.

## License

Apache License, Version 2.0

## Release note ##

[v1.0.0 - May 29, 2019 compatible with dubbo v2.6.5](https://github.com/dubbo/go-for-apache-dubbo/releases/tag/v1.0.0)

## Project Architecture ##

Extension module and layered code design based on dubbo (include protocol layer,registry layer,cluster layer,config layer and so on), Our goal is: you can implement these layered interfaces in a new way, and override the default implementation of dubbo-go[same go-for-apache-dubbo] by calling 'extension.SetXXX' of extension, and complete your special needs without modifying the source code. At the same time, you are welcome to contribute implementation of useful expansion to the community.

![frame design](https://raw.githubusercontent.com/wiki/dubbo/dubbo-go/dubbo-go%E4%BB%A3%E7%A0%81%E5%88%86%E5%B1%82%E8%AE%BE%E8%AE%A1.png)

About detail design please refer to [code layered design](https://github.com/dubbo/go-for-apache-dubbo/wiki/dubbo-go-V1.0-design)

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
- metrics (dubbo v2.7.x) waiting dubbo's quota
- dynamic configuration center & metadata center (dubbo v2.7.x)
- tracing (dubbo ecosystem)

You can know more about dubbo-go by its [roadmap](https://github.com/dubbo/go-for-apache-dubbo/wiki/Roadmap).

## Quick Start

The subdirectory examples shows how to use dubbo-go. Please read the [examples/README.md](https://github.com/dubbo/go-for-apache-dubbo/blob/develop/examples/README.md) carefully to learn how to dispose the configuration and compile the program.

## Benchmark

Benchmark project please refer to [go-for-apache-dubbo-benchmark](https://github.com/dubbogo/go-for-apache-dubbo-benchmark)

About dubbo-go benchmarking report, please refer to [dubbo benchmarking report](https://github.com/dubbo/go-for-apache-dubbo/wiki/pressure-test-report-for-dubbo) & [jsonrpc benchmarking report](https://github.com/dubbo/go-for-apache-dubbo/wiki/pressure-test-report-for-jsonrpc)

## [User List](https://github.com/dubbo/go-for-apache-dubbo/issues/2)

![ctrip](https://pic.c-ctrip.com/common/c_logo2013.png)

## Stargazers

[![Stargazers over time](https://starchart.cc/dubbo/dubbo-go.svg)](https://starchart.cc/dubbo/dubbo-go)

