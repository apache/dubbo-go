# Apache Dubbo-go [中文](./README_CN.md) #

[![Build Status](https://travis-ci.org/apache/dubbo-go.svg?branch=master)](https://travis-ci.org/apache/dubbo-go)
[![codecov](https://codecov.io/gh/apache/dubbo-go/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-go)

---
Apache Dubbo Go Implementation.

## License

Apache License, Version 2.0

## Release note ##

[v1.0.0 - May 29, 2019 compatible with dubbo v2.6.5](https://github.com/apache/dubbo-go/releases/tag/v1.0.0)

## Project Architecture ##

Both extension module and layered project architecture is according to Apache Dubbo (including protocol layer, registry layer, cluster layer, config layer and so on), the advantage of this arch is as following: you can implement these layered interfaces in your own way, override the default implementation of dubbo-go by calling 'extension.SetXXX' of extension, complete your special needs without modifying the source code. At the same time, you are welcome to contribute implementation of useful extension to the community.

![frame design](https://raw.githubusercontent.com/wiki/dubbo/dubbo-go/dubbo-go%E4%BB%A3%E7%A0%81%E5%88%86%E5%B1%82%E8%AE%BE%E8%AE%A1.png)

If you wanna know more about dubbo-go, please visit this reference [Project Architeture design](https://github.com/apache/dubbo-go/wiki/dubbo-go-V1.0-design)

## Feature list ##

Finished List:

- Role: Consumer(√), Provider(√)
- Transport: HTTP(√), TCP(√)
- Codec: JsonRPC v2(√), Hessian v2(√)
- Registry: ZooKeeper(√)
- Cluster Strategy: Failover(√)
- Load Balance: Random(√), RoundRobin(√), LeastActive(√)
- Filter: Echo Health Check(√)

Working List:

- Cluster Strategy: Failfast/Failsafe/Failback/Forking
- Load Balance: ConsistentHash
- Filter: TokenFilter/AccessLogFilter/CountFilter/ActiveLimitFilter/ExecuteLimitFilter/GenericFilter/TpsLimitFilter
- Registry: etcd/k8s/consul
- Routing Rule (dubbo v2.6.x)
- Dynamic Configuration Center & Metadata Center (dubbo v2.7.x)
- Metrics (dubbo v2.7.x)

Todo List:

- tracing (dubbo ecosystem)

You can know more about dubbo-go by its [roadmap](https://github.com/apache/dubbo-go/wiki/Roadmap).

## Quick Start

The subdirectory examples shows how to use dubbo-go. Please read the [examples/README.md](https://github.com/apache/dubbo-go/blob/develop/examples/README.md) carefully to learn how to dispose the configuration and compile the program.

## Benchmark

Benchmark project please refer to [go-for-apache-dubbo-benchmark](https://github.com/dubbogo/go-for-apache-dubbo-benchmark)

About dubbo-go benchmarking report, please refer to [dubbo benchmarking report](https://github.com/apache/dubbo-go/wiki/pressure-test-report-for-dubbo) & [jsonrpc benchmarking report](https://github.com/apache/dubbo-go/wiki/pressure-test-report-for-jsonrpc)

## [User List](https://github.com/apache/dubbo-go/issues/2)

If you are using [apache/dubbo-go](github.com/apache/dubbo-go) and think that it helps you or want do some contributions to it, please add your company to to [the user list](https://github.com/apache/dubbo-go/issues/2) to let us know your needs.

![ctrip](https://pic.c-ctrip.com/common/c_logo2013.png)

## Stargazers

[![Stargazers over time](https://starchart.cc/apache/dubbo-go.svg)](https://starchart.cc/apache/dubbo-go)
If you are using , please add your company here to tell us your scenario to make Nacos better.
