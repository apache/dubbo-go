# go-for-apache-dubbo #
---
Apache Dubbo Golang Implementation.

## License

Apache License, Version 2.0

## Code design ##
Based on dubbo's layered code design (protocol layer,registry layer,cluster layer,config layer and so on),

About detail design please refer to [code layered design](https://github.com/dubbo/go-for-apache-dubbo/wiki/dubbo-go-V2.6-design)
## Feature list ##

+  Role: Consumer(√), Provider(√)

+  Transport: HTTP(√), TCP(√) Based on [getty](https://github.com/AlexStocks/getty)

+  Codec:  JsonRPC(√), Hessian(√) Based on [hession2](https://github.com/dubbogo/hessian2)

+  Registry: ZooKeeper(√)

+  Cluster Strategy: Failover(√)

+  Load Balance: Random(√)

+  Filter: Echo(√)

## Quick Start

The subdirectory examples shows how to use go-for-apache-dubbo. Please read the examples/readme.md carefully to learn how to dispose the configuration and compile the program.

## Benchmark

Benchmark project please refer to [go-for-apache-dubbo-benchmark](https://github.com/dubbogo/go-for-apache-dubbo-benchmark)

About go-for-apache-dubbo benchmarking report, please refer to [dubbo benchmarking report](https://github.com/dubbo/go-for-apache-dubbo/wiki/pressure-test-report-for-dubbo) & [jsonrpc benchmarking report](https://github.com/dubbo/go-for-apache-dubbo/wiki/pressure-test-report-for-jsonrpc)


## Todo list

Implement more extention:

 * cluster strategy : Failfast/Failsafe/Failback/Forking/Broadcast

 * load balance strategy: RoundRobin/LeastActive/ConsistentHash

 * standard filter in dubbo: TokenFilter/AccessLogFilter/CountFilter/ActiveLimitFilter/ExecuteLimitFilter/GenericFilter/TpsLimitFilter

 * registry impl: consul/etcd/k8s
 
Compatible with dubbo v2.7.x and not finished function in dubbo v2.6.x:
 
 * routing rule (dubbo v2.6.x)
 
 * monitoring (dubbo v2.6.x)
 
 * metrics (dubbo v2.6.x)
 
 * dynamic configuration (dubbo v2.7.x)

About the roadmap please refer to [roadmap](https://github.com/dubbo/go-for-apache-dubbo/wiki/Roadmap)
