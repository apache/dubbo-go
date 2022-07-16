# Release Notes
---
## 3.0.2

### Features

- [Support hessian java method java.lang param](https://github.com/apache/dubbo-go/pull/1783)
- [Support istio xds](https://github.com/apache/dubbo-go/pull/1804)
- [Support istio xds ring hash](https://github.com/apache/dubbo-go/pull/1828)
- [Support otel trace](https://github.com/apache/dubbo-go/pull/1886)

### Bugfixs

- [Fix: where limitation not updates](https://github.com/apache/dubbo-go/pull/1784)
- [Fix: rootConfig and getty-session-param](https://github.com/apache/dubbo-go/pull/1802)
- [Fix: xds adsz empty metadata](https://github.com/apache/dubbo-go/pull/1813)
- [Fix: decode net stream bytes as getty rule](https://github.com/apache/dubbo-go/pull/1820)
- [Fix: ip register issue](https://github.com/apache/dubbo-go/pull/1821)
- [Fix: getty unit test](https://github.com/apache/dubbo-go/pull/1829)
- [Fix: remove HEADER_LENGTH in decode because of discard](https://github.com/apache/dubbo-go/pull/1831)
- [Fix: xds log use dubbogo logger](https://github.com/apache/dubbo-go/pull/1846)
- [Fix: limit rpc package data size by user's config rather than DEFAULT_LEN](https://github.com/apache/dubbo-go/pull/1848)
- [Fix: complete grpc based protocol panic recover handle](https://github.com/apache/dubbo-go/pull/1866)

### Enhancements

- [Add req.Data to OnMessage panic error log](https://github.com/apache/dubbo-go/pull/1847)
- [Add nacos updateCacheWhenEmpty options](https://github.com/apache/dubbo-go/pull/1852)
- [Xds enhancement](https://github.com/apache/dubbo-go/pull/1853)
- [Mock etcd and nacos in ut](https://github.com/apache/dubbo-go/pull/1873)
- [Use summary type to observe p99](https://github.com/apache/dubbo-go/pull/1875)
- [Remove gomonkey](https://github.com/apache/dubbo-go/pull/1881)
- [Optimized load balancing algorithm](https://github.com/apache/dubbo-go/pull/1884)
- [RandomLoadBalance code optimization](https://github.com/apache/dubbo-go/pull/1899)
- [Export hessian type api](https://github.com/apache/dubbo-go/pull/1911)
- [Export method getArgsTypeList for extension](https://github.com/apache/dubbo-go/pull/1913)


## 3.0.1

### Features

- [Support Graceful Shutdown](https://github.com/apache/dubbo-go/pull/1675)
- [Support `$invokeAsync` for generic service](https://github.com/apache/dubbo-go/pull/1674)
- [Support config the Nacos context path](https://github.com/apache/dubbo-go/pull/1656)

### Bugfixs

- [Fix: JSON-RPC request timeout time dynamically](https://github.com/apache/dubbo-go/pull/1713)
- [Fix: the heartbeat of polaris cannot be reported](https://github.com/apache/dubbo-go/pull/1688)
- [Fix: triple protocol request doesn't carry attachment such as group or version](https://github.com/apache/dubbo-go/pull/1700)
- [Fix: location trim space](https://github.com/apache/dubbo-go/pull/1670)
- [Fix: add category key to the Consumer for diff with Provider](https://github.com/apache/dubbo-go/pull/1672)

### Enhancements

- [Disabled Adaptive Service if set disabled by the Client](https://github.com/apache/dubbo-go/pull/1748)
- [Don't load Adaptive Service Filter if not set ](https://github.com/apache/dubbo-go/pull/1735)
- [Log print the caller ](https://github.com/apache/dubbo-go/pull/1732)
- [Refactor invocation interface](https://github.com/apache/dubbo-go/pull/1702)
- [Refactor Nacos listener check-healthy ](https://github.com/apache/dubbo-go/pull/1729)
- [Refactor Zookeeper dynamic configuration listener](https://github.com/apache/dubbo-go/pull/1665)


## 3.0.0

### Features

- [Adaptive service support](https://github.com/apache/dubbo-go/pull/1649)
- [User defined configuration of dubbogo.yml](https://github.com/apache/dubbo-go/pull/1640)
- [Extend configuration file format](https://github.com/apache/dubbo-go/pull/1626)
- [Polaris-Mesh reigstry support](https://github.com/apache/dubbo-go/pull/1620)
- [Triple proto reflection support](https://github.com/apache/dubbo-go/pull/1603)
- [Triple pb with jaeger tracing support](https://github.com/apache/dubbo-go/pull/1596)

### Bugfixs

- [Validate nacos's user and password configuration](https://github.com/apache/dubbo-go/pull/1645)
- [Fix bug of service configuration exported field](https://github.com/apache/dubbo-go/pull/1639/files)
- [Fix bug of attachments' nil or empty string](https://github.com/apache/dubbo-go/pull/1631)
- [Fix bug of zookeeper cpu idling](https://github.com/apache/dubbo-go/pull/1629)
- [Fix bug of  register url without group/version key if value is empty, instead of "" value.](https://github.com/apache/dubbo-go/pull/1612)
- [Fix Hessian encode attachments return error](https://github.com/apache/dubbo-go/pull/1588)
- [Fix consumer zk registry path bug](https://github.com/apache/dubbo-go/pull/1586)
- [Fix bug of graceful shutdown filter](https://github.com/apache/dubbo-go/pull/1585)
- [Validate nacos service-discovery's group and namespace](https://github.com/apache/dubbo-go/pull/1581)
- [Validate consumer config's 'check' key](https://github.com/apache/dubbo-go/pull/1568)
- [Set rootConfig to global rootConfig pointer in init function](https://github.com/apache/dubbo-go/pull/1564)
- [Fix bug of register of app-level-service-discovery not use metadata's configuration](https://github.com/apache/dubbo-go/pull/1565)

### Enhancements

- [Add triple max size configuration](https://github.com/apache/dubbo-go/pull/1654)
- [Hessian2 supports setting the type of Java method parameters](https://github.com/apache/dubbo-go/pull/1625)
- [Logger of dubbo-go internal codes are changed to the framework's logger](https://github.com/apache/dubbo-go/pull/1617)
- [Nacos suppports unregister and unsubscribe](https://github.com/apache/dubbo-go/pull/1616)
- [Register service mapping using registry config's address](https://github.com/apache/dubbo-go/pull/1611)
- [Add nacos registry url's 'methods' key](https://github.com/apache/dubbo-go/pull/1608)
- [Filter with single instance](https://github.com/apache/dubbo-go/pull/1591)
- [Refactor of listen dir event](https://github.com/apache/dubbo-go/pull/1589)
- [Load reference before service](https://github.com/apache/dubbo-go/pull/1571)
- [Change triple's http2 implementation from net/http to grpc](https://github.com/apache/dubbo-go/pull/1566)
- [Adjuest the startup process of configcenter](https://github.com/apache/dubbo-go/pull/1560)

## 3.0.0-rc3

### Feature: 

- Triple features

  - [New triple pb generation tool](https://github.com/dubbogo/tools/pull/17)

    Refer to [QuickStart of dubbo-go 3.0](https://dubbogo.github.io/zh-cn/docs/user/quickstart/3.0/quickstart.html), and [Dubbo-go-samples](https://github.com/apache/dubbo-go-samples) to try protoc-gen-go-triple pb generator plugin. 

  - [Response exception from client](https://github.com/dubbogo/triple/pull/30)

    Triple client can get the error stacks from server, pointing to real error occurs position.

  - [Multi params support](https://github.com/apache/dubbo-go/pull/1344)

- [New metrics support](https://github.com/apache/dubbo-go/pull/1540)

  Refer to [Docs of dubbo-go 3.0 Metrics](https://dubbogo.github.io/zh-cn/docs/user/samples/metrics.html), and [dubbo-go-samples/metrics](https://github.com/apache/dubbo-go-samples/tree/master/metrics)] to try new metrics support.

- [Remove sleep to wait for service discovery logic](https://github.com/apache/dubbo-go/pull/1531)

  Set reference check default to true and remove extra time.Sleep logic in client side during service-discovery [remove time.Sleep](https://github.com/apache/dubbo-go-samples/pull/276/files)

  - [Registry Waittime configurable](https://github.com/apache/dubbo-go/pull/1516/files)

- [Dynamic Route Config](https://github.com/apache/dubbo-go/pull/1519)

  Refer to [Docs of dubbo-go 3.0 Router](https://dubbogo.github.io/zh-cn/docs/user/samples/mesh_router.html), and [dubbo-go-samples/meshrouter](https://github.com/apache/dubbo-go-samples/tree/master/route/meshroute) to try dynamic mesh router support.

- [New config API support](https://github.com/apache/dubbo-go/pull/1499) [New root config API builder](https://github.com/apache/dubbo-go/pull/1491)

  Refer to [Docs of dubbo-go 3.0 Configuration](https://dubbogo.github.io/zh-cn/docs/user/concept/configuration.html), and [dubbo-go-samples/config-api](https://github.com/apache/dubbo-go-samples/tree/master/config-api) to try new config API.

- [Support custom registry group name on Nacos](https://github.com/apache/dubbo-go/pull/1353)

- [GRPC supports multi pb](https://github.com/apache/dubbo-go/pull/1361)

- [New logger support](https://github.com/apache/dubbo-go/pull/1335)

  Refer to [Docs of dubbo-go 3.0 logger](https://dubbogo.github.io/zh-cn/docs/user/samples/custom-logger.html), and [dubbo-go-samples/logger](https://github.com/apache/dubbo-go-samples/tree/master/logger) to try new logger support.

- [Generic invocation supports](https://github.com/apache/dubbo-go/pull/1315)

  Refer to [Docs of dubbo-go 3.0 generic](https://dubbogo.github.io/zh-cn/docs/user/samples/generic.html), and [dubbo-go-samples/generic](https://github.com/apache/dubbo-go-samples/tree/master/generic) to try new generic support.

- [Support key generate function in service event](https://github.com/apache/dubbo-go/pull/1286)

  

### Enhancement:

- [Configuration Enhancement](https://github.com/apache/dubbo-go/commit/1397e8bba97f14b7656a5afc9bc92530bb693092)

  One of the biggest update in this release is the configuration optimization. We discarded the old configuration and introduced a new one. Currently, the new configuration has been updated to [the master branch of samples](https://github.com/apache/dubbo-go-samples) (corresponding to latest dubbo-go 3.0), and there are introductions and detailed examples on the our website [dubbogo.github](https://dubbogo.github.io).

- Triple: 

  - [Triple CPU usage optimization](https://github.com/dubbogo/triple/pull/32/files)

     Set default tcp read buffer to 4k to decrease gc, and descrease CPU usage by 60% of 3.0.0-rc2

  - [Add Triple Debug Log](https://github.com/dubbogo/triple/pull/29)

- [Support Apollo secret](https://github.com/apache/dubbo-go/pull/1533)

- [Use class Name as the default reference name ](https://github.com/apache/dubbo-go/pull/1339)

  You can set service/refernce key to provider/consumer's struct name: [Config samples](https://github.com/apache/dubbo-go-samples/blob/master/helloworld/go-server/conf/dubbogo.yml#L15) & [Target provider struct](https://github.com/apache/dubbo-go-samples/blob/master/helloworld/go-server/cmd/server.go#L45), 

  And there is no needs to define (p*Provider) Reference() method from now on.

- [Set default logger level to info](https://github.com/apache/dubbo-go/pull/1549/files#diff-d5ab135265094924568957f56eaef061c7948d2664daa995fbe0de4c7ab2d272R82)
- [Refactor of filter package sturcture](https://github.com/apache/dubbo-go/pull/1299)
- [Refactor of cluster package structure](https://github.com/apache/dubbo-go/pull/1507)



### Bugfix:

- [Heartbeat's timeout will modify consumer's timeout](https://github.com/apache/dubbo-go/pull/1532)

- [Remove zk test to ensure ut could be run locally](https://github.com/apache/dubbo-go/pull/1357)

- [Add Application Registry](https://github.com/apache/dubbo-go/pull/1493)

- [Change serviceName like java style on nacos](https://github.com/apache/dubbo-go/pull/1352)

- [Url serialization bug](https://github.com/apache/dubbo-go/pull/1292)

- [Change the key of a mock echo filter](https://github.com/apache/dubbo-go/pull/1381)

- [Fix: fix the exception when tcp timeout is less than 1s for 3.0 #1362](https://github.com/apache/dubbo-go/pull/1380)

- [Registry timeout not pars](https://github.com/apache/dubbo-go/pull/1392)

- [Fix Isprovider check](https://github.com/apache/dubbo-go/pull/1500)

- [Delete zk registry when set defualt consumer/provider config](https://github.com/apache/dubbo-go/pull/1324/files)

  
## 3.0.0-rc2

### New Features

- [Add Triple Msgpack Codec](https://github.com/apache/dubbo-go/pull/1242)
- [Add Triple user defined serializer support](https://github.com/apache/dubbo-go/pull/1242)
- [Add gRPC provider reference in codes generated by protoc-gen-dubbo3](https://github.com/apache/dubbo-go/pull/1240)
- [Add integration tests using dubbo-go-samples](https://github.com/apache/dubbo-go/pull/1223)
- [Add service discovery support etcd remote reporter](https://github.com/apache/dubbo-go/pull/1221)
- [Add service discovery support nacos remote reporter](https://github.com/apache/dubbo-go/pull/1218)
- [Add grpc server reflection register logic](https://github.com/apache/dubbo-go/pull/1216)

### Enhancement

- [Make remote metadata center configurable](https://github.com/apache/dubbo-go/pull/1258)
- [Enhance nacos connection](https://github.com/apache/dubbo-go/pull/1255)
- [Add unit tests for zk metadata report](https://github.com/apache/dubbo-go/pull/1229)
- [Restructuring remoting metadata service](https://github.com/apache/dubbo-go/pull/1227)
- [Dependency prompting for unit tests](https://github.com/apache/dubbo-go/pull/1212)
- [Make cluster interceptor a chain](https://github.com/apache/dubbo-go/pull/1211)
- [Improve etcd version and change create to put](https://github.com/apache/dubbo-go/pull/1203)
- [Remove reflect in grpc server](https://github.com/apache/dubbo-go/pull/1200)
- [Change lb hash logic](https://github.com/apache/dubbo-go/pull/1267)

### Bugfixes

- [Fix: zk invoker refer check fail,and service will be added in cache invokers fail problem](https://github.com/apache/dubbo-go/pull/1249)
- [Fix: app level service discovery local mod URL serialize fail problem](https://github.com/apache/dubbo-go/pull/1238)
- [Fix: m1 cpu exec fail problem](https://github.com/apache/dubbo-go/pull/1236)
- [Fix: metadata info struct contains unsupported field](https://github.com/apache/dubbo-go/pull/1234)
- [Fix: go race in directory](https://github.com/apache/dubbo-go/pull/1222)
- [Fix: zk name changes from default to conn location](https://github.com/apache/dubbo-go/pull/1263)

### Dependencies

- [bump actions/cache from 2.1.5 to 2.1.6](https://github.com/apache/dubbo-go/pull/1230)

Milestone:

- [https://github.com/apache/dubbo-go/milestone/12](https://github.com/apache/dubbo-go/milestone/12?closed=1)




## 3.0.0-rc1

### New Features
- [Add triple protocol](https://github.com/apache/dubbo-go/pull/1071)
- [Add dubbo3 router](https://github.com/apache/dubbo-go/pull/1187)
- [Add app-level remote service discovery](https://github.com/apache/dubbo-go/pull/1161)
- [Add default config](https://github.com/apache/dubbo-go/pull/1073)
- [Add pass through proxy factory](https://github.com/apache/dubbo-go/pull/1081)

### Enhancement
- [Replace default config string with const value](https://github.com/apache/dubbo-go/pull/1182)
- [Add retry times in zookeeper starting](https://github.com/apache/dubbo-go/pull/1179)
- [Client pool enhance](https://github.com/apache/dubbo-go/pull/1119)
- [Impl the way of load configure file](https://github.com/apache/dubbo-go/pull/1099)

### Bugfixes
- [Fix: get failback error](https://github.com/apache/dubbo-go/pull/1177)
- [Fix: delete a service provider when using k8s hpa](https://github.com/apache/dubbo-go/pull/1157)
- [Fix: fix reExporter bug when config-center {applicationName}.configurator data change](https://github.com/apache/dubbo-go/pull/1144)
- [Fix: stop the provider app panic error](https://github.com/apache/dubbo-go/pull/1129)

### Dependencies
- [bump github.com/RoaringBitmap/roaring from 0.6.0 to 0.6.1](https://github.com/apache/dubbo-go/pull/1195)
- [bump github.com/RoaringBitmap/roaring from 0.5.5 to 0.6.0](https://github.com/apache/dubbo-go/pull/1175)
- [bump github.com/RoaringBitmap/roaring from 0.5.5 to 0.6.0](https://github.com/apache/dubbo-go/pull/1163)
- [Bump github.com/magiconair/properties from 1.8.4 to 1.8.5](https://github.com/apache/dubbo-go/pull/1115)

Milestone:
- [https://github.com/apache/dubbo-go/milestone/9](https://github.com/apache/dubbo-go/milestone/9?closed=1)

## 1.5.6

### New Features
- [Add dubbo-go-cli telnet tool](https://github.com/apache/dubbo-go/pull/818)
- [Add Prox ImplementFunc to allow override impl](https://github.com/apache/dubbo-go/pull/1019)
- [Add read configuration path from the command line when start](https://github.com/apache/dubbo-go/pull/1039)
- [Add use invoker with same ip as client first](https://github.com/apache/dubbo-go/pull/1023)
- [Add an "api way" to set general configure](https://github.com/apache/dubbo-go/pull/1020)
- [Add registry ip:port set from enviroment variable](https://github.com/apache/dubbo-go/pull/1036)

### Enhancement
- [introduce ConfigPostProcessor extension](https://github.com/apache/dubbo-go/pull/943)
- [Impl extension of two urls comparison](https://github.com/apache/dubbo-go/pull/854)
- [using event-driven to let router send signal to notify channel](https://github.com/apache/dubbo-go/pull/976)
- [lint codes](https://github.com/apache/dubbo-go/pull/941)
- [Imp: destroy invoker smoothly](https://github.com/apache/dubbo-go/pull/1045)
- [Improve config center](https://github.com/apache/dubbo-go/pull/1030)

### Bugfixes
- [Fix: generic struct2MapAll key of map keep type](https://github.com/apache/dubbo-go/pull/928)
- [Fix: when events empty, delete all the invokers](https://github.com/apache/dubbo-go/pull/758)
- [Fix: file service discovery run in windows](https://github.com/apache/dubbo-go/pull/932)
- [Fix: make metadata report work without serviceDiscovery](https://github.com/apache/dubbo-go/pull/948)
- [Fix: consumer invoker cache set nil after the ZK connection is lost](https://github.com/apache/dubbo-go/pull/985)
- [Fix: integration test in Github action](https://github.com/apache/dubbo-go/pull/1012)
- [Fix: etcd exit panic](https://github.com/apache/dubbo-go/pull/1013)
- [Fix: when connect to provider fail, will occur panic](https://github.com/apache/dubbo-go/pull/1021)
- [Fix: support getty send Length, when the data transfer failed](https://github.com/apache/dubbo-go/pull/1028)
- [Fix: RPCInvocation.ServiceKey use PATH_KEY instead of INTERFACE_KEY ](https://github.com/apache/dubbo-go/pull/1078/files)
- [Fix: zk too many tcp conn](https://github.com/apache/dubbo-go/pull/1010)
- [Fix: fix zk listener func pathToKey](https://github.com/apache/dubbo-go/pull/1066)
- [Fix: graceful shutdown](https://github.com/apache/dubbo-go/pull/1007)
- [Fix: nacos service provider does not require subscribe](https://github.com/apache/dubbo-go/pull/1056)
- [Fix: key of generic map convert is more general](https://github.com/apache/dubbo-go/pull/1041)
- [Fix: body buffer too short](https://github.com/apache/dubbo-go/pull/1090)

### Dependencies
- [Bump dubbo-go-hessian2 from v1.9.0-rc1 to v1.9.1](https://github.com/apache/dubbo-go/pull/1088/files)
- [Bump github.com/nacos-group/nacos-sdk-go from 1.0.5 to v1.0.7](https://github.com/apache/dubbo-go/pull/1106)

Milestone: 
- [https://github.com/apache/dubbo-go/milestone/7](https://github.com/apache/dubbo-go/milestone/7?closed=1)
- [https://github.com/apache/dubbo-go/milestone/10](https://github.com/apache/dubbo-go/milestone/10?closed=1)

## 1.5.5

### New Features
- [Add Address notification batch mode](https://github.com/apache/dubbo-go/pull/741) 
- [Add dubbo-gen stream support](https://github.com/apache/dubbo-go/pull/794) 
- [Add Change verify to Makefile](https://github.com/apache/dubbo-go/pull/831) 
- [Add more automatic components](https://github.com/apache/dubbo-go/pull/832) 
- [Add grpc max message size config](https://github.com/apache/dubbo-go/pull/824) 

### Enhancement
- [when it need local ip, it will get it every time. We can get local ip once, and reused it](https://github.com/apache/dubbo-go/pull/807) 
- [enhance client's connectivity](https://github.com/apache/dubbo-go/pull/800) 
- [Imp: get local ip once and reused it](https://github.com/apache/dubbo-go/pull/808) 
- [Remove unmeaning logic](https://github.com/apache/dubbo-go/pull/855) 

### Bugfixes
- [Fix: nacos registry can not get namespaceId](https://github.com/apache/dubbo-go/pull/778) [@peaman](https://github.com/peaman)       
- [Fix: url encode](https://github.com/apache/dubbo-go/pull/802)       
- [Fix: try to fix too many files open error](https://github.com/apache/dubbo-go/pull/797)       
- [Fix: refact heartbeat](https://github.com/apache/dubbo-go/pull/889)       
- [Fix: router_config add &url to url](https://github.com/apache/dubbo-go/pull/910)       
- [Fix: Router chain can not build immediately when started](https://github.com/apache/dubbo-go/pull/927)       
- [Fix: client block until timeout when provider return with PackageResponse_Exception](https://github.com/apache/dubbo-go/pull/926)      
- [Fix: URL.String() data race panic](https://github.com/apache/dubbo-go/pull/944)
- [Fix: generic "encode hessian.Object"](https://github.com/apache/dubbo-go/pull/945)

### Dependencies
- [Bump github.com/mitchellh/mapstructure from 1.2.3 to 1.3.3](https://github.com/apache/dubbo-go/pull/838)
- [Bump github.com/go-resty/resty/v2 from 2.1.0 to 2.3.0](https://github.com/apache/dubbo-go/pull/837)
- [Bump github.com/opentracing/opentracing-go from 1.1.0 to 1.2.0](https://github.com/apache/dubbo-go/pull/836)
- [Bump github.com/creasty/defaults from 1.3.0 to 1.5.1](https://github.com/apache/dubbo-go/pull/835)
- [Bump github.com/dubbogo/gost from 1.9.1 to 1.9.2](https://github.com/apache/dubbo-go/pull/834)
- [Bump github.com/zouyx/agollo/v3 from 3.4.4 to 3.4.5](https://github.com/apache/dubbo-go/pull/845)
- [Bump github.com/golang/mock from 1.3.1 to 1.4.4](https://github.com/apache/dubbo-go/pull/844)
- [Bump github.com/nacos-group/nacos-sdk-go from 1.0.0 to 1.0.1](https://github.com/apache/dubbo-go/pull/843)
- [Bump github.com/magiconair/properties from 1.8.1 to 1.8.4](https://github.com/apache/dubbo-go/pull/861)
- [Bump github.com/prometheus/client_golang from 1.1.0 to 1.8.0 ](https://github.com/apache/dubbo-go/pull/860)
- [Bump go.uber.org/atomic from 1.6.0 to 1.7.0](https://github.com/apache/dubbo-go/pull/859)
- [](https://github.com/apache/dubbo-go/pull/843)

Milestone: [https://github.com/apache/dubbo-go/milestone/5](https://github.com/apache/dubbo-go/milestone/5?closed=1)

## 1.4.5

### Bugfixes
- [Fix too many files open error](https://github.com/apache/dubbo-go/pull/828)  [@wenxuwan](https://github.com/wenxuwan) Milestone: [https://github.com/apache/dubbo-go/milestone/6](https://github.com/apache/dubbo-go/milestone/6?closed=1)

## 1.5.4

### Bugfixes
- [Fix etcd cluster reconnect](https://github.com/apache/dubbo-go/pull/828)
- [Fix zookeeper deadlock problem](https://github.com/apache/dubbo-go/pull/826)
- [Fix generic struct2MapAll](https://github.com/apache/dubbo-go/pull/822)
- [Fix Consumer panic when restart provider](https://github.com/apache/dubbo-go/pull/803)
- [Fix etcd can not registry](https://github.com/apache/dubbo-go/pull/819) [@lin-jianjun](https://github.com/lin-jianjun)
- [Fix cannot call go provider service when used by java dubbo 2.7.7 version](https://github.com/apache/dubbo-go/pull/815) [@jack15083](https://github.com/jack15083)
- [Fix go client quit abnormally when it connects java server](https://github.com/apache/dubbo-go/pull/820) [@wenxuwan](https://github.com/wenxuwan)
- [Fix sentinel windows issue](https://github.com/apache/dubbo-go/pull/821) [@louyuting](https://github.com/louyuting)
- [Fix metadata default port](https://github.com/apache/dubbo-go/pull/821) [@sanxun0325](https://github.com/sanxun0325)
- [Fix consul can not destroy](https://github.com/apache/dubbo-go/pull/788) [@LaurenceLiZhixin](https://github.com/LaurenceLiZhixin)

Milestone: [https://github.com/apache/dubbo-go/milestone/6](https://github.com/apache/dubbo-go/milestone/6?closed=1)

## 1.5.3

### New Features
- [Add consul service discovery](https://github.com/apache/dubbo-go/pull/701) [@zhangshen023](https://github.com/zhangshen023)
- [Add File system service discovery](https://github.com/apache/dubbo-go/pull/732) [@DogBaoBao](https://github.com/DogBaoBao)
- [Migrate travis Ci to Github Actions](https://github.com/apache/dubbo-go/pull/752) [@sdttttt](https://github.com/sdttttt)
- [Add sentinel-golang flow control/circuit breaker](https://github.com/apache/dubbo-go/pull/748) [@louyuting](https://github.com/louyuting)
- [Add dubbo-go docs and blog into doc directory](https://github.com/apache/dubbo-go/pull/767) [@oaoit](https://github.com/oaoit)

### Enhancement
- [Add address notification batch mode](https://github.com/apache/dubbo-go/pull/741) [@beiwei30](https://github.com/beiwei30)
- [Refactor network and codec model](https://github.com/apache/dubbo-go/pull/673) [@fangyincheng](https://github.com/fangyincheng) [@georgehao](https://github.com/georgehao)
- [Remove unnecessary return and judgement](https://github.com/apache/dubbo-go/pull/730) [@YongHaoWu](https://github.com/YongHaoWu)
- [Improve exporter append method](https://github.com/apache/dubbo-go/pull/722) [@gaoxinge](https://github.com/gaoxinge)
- [Refactor for proxyInvoker cannot be extended](https://github.com/apache/dubbo-go/pull/747) [@cvictory](https://github.com/cvictory)
- [Refactor attachment type from map\[string\]stiring to map\[string\]interface{}](https://github.com/apache/dubbo-go/pull/713) [@cvictory](https://github.com/cvictory)
- [Improve map access concurrency](https://github.com/apache/dubbo-go/pull/739) [@skyao](https://github.com/skyao)
- [Improve code quantity](https://github.com/apache/dubbo-go/pull/763) [@gaoxinge](https://github.com/gaoxinge)

### Bugfixes
- [Fix etcdv3 lease](https://github.com/apache/dubbo-go/pull/738) [@zhangshen023](https://github.com/zhangshen023)
- [Fix rename SethealthChecker to SetHealthChecker](https://github.com/apache/dubbo-go/pull/746) [@watermelo](https://github.com/watermelo)
- [Fix init config problem in HystrixFilter](https://github.com/apache/dubbo-go/pull/731) [@YGrylls](https://github.com/YGrylls)
- [Fix zookeeper listener report error after started](https://github.com/apache/dubbo-go/pull/735) [@wenxuwan](https://github.com/wenxuwan)

Milestone: [https://github.com/apache/dubbo-go/milestone/4](https://github.com/apache/dubbo-go/milestone/4?closed=1)

Project: [https://github.com/apache/dubbo-go/projects/10](https://github.com/apache/dubbo-go/projects/10)

## 1.5.4

### Bugfixes
- [Fix etcd cluster reconnect](https://github.com/apache/dubbo-go/pull/828)  
- [Fix zookeeper deadlock problem](https://github.com/apache/dubbo-go/pull/826)      
- [Fix generic struct2MapAll](https://github.com/apache/dubbo-go/pull/822)     
- [Fix Consumer panic when restart provider](https://github.com/apache/dubbo-go/pull/803)     
- [Fix etcd can not registry](https://github.com/apache/dubbo-go/pull/819) [@lin-jianjun](https://github.com/lin-jianjun)
- [Fix cannot call go provider service when used by java dubbo 2.7.7 version](https://github.com/apache/dubbo-go/pull/815) [@jack15083](https://github.com/jack15083) 
- [Fix go client quit abnormally when it connects java server](https://github.com/apache/dubbo-go/pull/820) [@wenxuwan](https://github.com/wenxuwan)      
- [Fix sentinel windows issue](https://github.com/apache/dubbo-go/pull/821) [@louyuting](https://github.com/louyuting)      
- [Fix metadata default port](https://github.com/apache/dubbo-go/pull/821) [@sanxun0325](https://github.com/sanxun0325)      
- [Fix consul can not destroy](https://github.com/apache/dubbo-go/pull/788) [@LaurenceLiZhixin](https://github.com/LaurenceLiZhixin)      

Milestone: [https://github.com/apache/dubbo-go/milestone/6](https://github.com/apache/dubbo-go/milestone/6?closed=1)

## 1.5.3

### New Features
- [Add consul service discovery](https://github.com/apache/dubbo-go/pull/701) [@zhangshen023](https://github.com/zhangshen023)
- [Add File system service discovery](https://github.com/apache/dubbo-go/pull/732) [@DogBaoBao](https://github.com/DogBaoBao)
- [Migrate travis Ci to Github Actions](https://github.com/apache/dubbo-go/pull/752) [@sdttttt](https://github.com/sdttttt)
- [Add sentinel-golang flow control/circuit breaker](https://github.com/apache/dubbo-go/pull/748) [@louyuting](https://github.com/louyuting)
- [Add dubbo-go docs and blog into doc directory](https://github.com/apache/dubbo-go/pull/767) [@oaoit](https://github.com/oaoit) 

### Enhancement
- [Add address notification batch mode](https://github.com/apache/dubbo-go/pull/741) [@beiwei30](https://github.com/beiwei30)
- [Refactor network and codec model](https://github.com/apache/dubbo-go/pull/673) [@fangyincheng](https://github.com/fangyincheng) [@georgehao](https://github.com/georgehao)
- [Remove unnecessary return and judgement](https://github.com/apache/dubbo-go/pull/730) [@YongHaoWu](https://github.com/YongHaoWu)
- [Improve exporter append method](https://github.com/apache/dubbo-go/pull/722) [@gaoxinge](https://github.com/gaoxinge)
- [Refactor for proxyInvoker cannot be extended](https://github.com/apache/dubbo-go/pull/747) [@cvictory](https://github.com/cvictory) 
- [Refactor attachment type from map\[string\]stiring to map\[string\]interface{}](https://github.com/apache/dubbo-go/pull/713) [@cvictory](https://github.com/cvictory) 
- [Improve map access concurrency](https://github.com/apache/dubbo-go/pull/739) [@skyao](https://github.com/skyao) 
- [Improve code quantity](https://github.com/apache/dubbo-go/pull/763) [@gaoxinge](https://github.com/gaoxinge) 

### Bugfixes
- [Fix etcdv3 lease](https://github.com/apache/dubbo-go/pull/738) [@zhangshen023](https://github.com/zhangshen023) 
- [Fix rename SethealthChecker to SetHealthChecker](https://github.com/apache/dubbo-go/pull/746) [@watermelo](https://github.com/watermelo)  
- [Fix init config problem in HystrixFilter](https://github.com/apache/dubbo-go/pull/731) [@YGrylls](https://github.com/YGrylls)   
- [Fix zookeeper listener report error after started](https://github.com/apache/dubbo-go/pull/735) [@wenxuwan](https://github.com/wenxuwan)      

Milestone: [https://github.com/apache/dubbo-go/milestone/4](https://github.com/apache/dubbo-go/milestone/4?closed=1)

Project: [https://github.com/apache/dubbo-go/projects/10](https://github.com/apache/dubbo-go/projects/10)

## 1.5.1

### New Features
- [Add dynamic tag router](https://github.com/apache/dubbo-go/pull/703)
- [Add TLS support](https://github.com/apache/dubbo-go/pull/685)
- [Add Nearest first for multiple registry](https://github.com/apache/dubbo-go/pull/659)
- [Add application and service level router](https://github.com/apache/dubbo-go/pull/662)
- [Add dynamic tag router](https://github.com/apache/dubbo-go/pull/665)

### Enhancement
- [Avoid init the log twice](https://github.com/apache/dubbo-go/pull/719)
- [Correct words and format codes](https://github.com/apache/dubbo-go/pull/704)
- [Change log stack level from warn to error](https://github.com/apache/dubbo-go/pull/702)
- [Optimize remotes configuration](https://github.com/apache/dubbo-go/pull/687)

### Bugfixes
- [Fix register service instance after provider config load](https://github.com/apache/dubbo-go/pull/694)
- [Fix call subscribe function asynchronously](https://github.com/apache/dubbo-go/pull/721)
- [Fix tag router rule copy](https://github.com/apache/dubbo-go/pull/721)
- [Fix nacos unit test failed](https://github.com/apache/dubbo-go/pull/705)
- [Fix can not inovke nacos destroy when graceful shutdown](https://github.com/apache/dubbo-go/pull/689)
- [Fix zk lost event](https://github.com/apache/dubbo-go/pull/692)
- [Fix k8s ut bug](https://github.com/apache/dubbo-go/pull/693)

Milestone: [https://github.com/apache/dubbo-go/milestone/2?closed=1](https://github.com/apache/dubbo-go/milestone/2?closed=1)

Project: [https://github.com/apache/dubbo-go/projects/8](https://github.com/apache/dubbo-go/projects/8)

## 1.5.0

### New Features
- [Application-Level Registry Model](https://github.com/apache/dubbo-go/pull/604)
    - [DelegateMetadataReport & RemoteMetadataService](https://github.com/apache/dubbo-go/pull/505)
    - [Nacos MetadataReport implementation](https://github.com/apache/dubbo-go/pull/522)
    - [Nacos service discovery](https://github.com/apache/dubbo-go/blob/9a5990d9a9c3d5e6633c0d7d926c156416bcb931/registry/nacos/service_discovery.go)
    - [Zk metadata service](https://github.com/apache/dubbo-go/pull/633)
    - [Zk service discovery](https://github.com/apache/dubbo-go/blob/9a5990d9a9c3d5e6633c0d7d926c156416bcb931/registry/zookeeper/service_discovery.go)
    - [Etcd metadata report](https://github.com/apache/dubbo-go/blob/9a5990d9a9c3d5e6633c0d7d926c156416bcb931/metadata/report/etcd/report.go)
    - [Etcd metadata service discovery](https://github.com/apache/dubbo-go/blob/9a5990d9a9c3d5e6633c0d7d926c156416bcb931/registry/etcdv3/service_discovery.go)
- [Support grpc json protocol](https://github.com/apache/dubbo-go/pull/582)
- [Ftr: using different labels btw provider and consumer, k8s service discovery across namespaces](https://github.com/apache/dubbo-go/pull/577 )

### Enhancement
- [Optimize err handling ](https://github.com/apache/dubbo-go/pull/536/)
- [Add attribute method into Invocation and RpcInvocation](https://github.com/apache/dubbo-go/pull/537)
- [Optimize lock for zookeeper registry](https://github.com/apache/dubbo-go/pull/578)
- [Improve code coverage of zookeeper config center](https://github.com/apache/dubbo-go/pull/549)
- [Improve code coverage of nacos config center and configuration parser](https://github.com/apache/dubbo-go/pull/587)
- [Kubernetes as registry enhance](https://github.com/apache/dubbo-go/pull/577)
- [Optimize zk client's lock and tests](https://github.com/apache/dubbo-go/pull/601)
- [Add setInvoker method for invocation](https://github.com/apache/dubbo-go/pull/612)
- [Upgrade getty & hessian2](https://github.com/apache/dubbo-go/pull/626)
- [Optimize router design: Extract priority router](https://github.com/apache/dubbo-go/pull/630)
- [NamespaceId config for nacos](https://github.com/apache/dubbo-go/pull/641)


### Bugfixes
- [Fix Gitee problem](https://github.com/apache/dubbo-go/pull/590)
- [Gitee quality analyses -- common](https://github.com/apache/dubbo-go/issues/616)
- [Nacos client logDir path seperator for Windows](https://github.com/apache/dubbo-go/pull/591)
- [Fix various linter warnings](https://github.com/apache/dubbo-go/pull/624)
- [Fixed some issues in config folder that reported by sonar-qube](https://github.com/apache/dubbo-go/pull/634)
- [Zk disconnected, dubbo-go panic when subscribe](https://github.com/apache/dubbo-go/pull/613)
- [Enhancement cluster code analysis](https://github.com/apache/dubbo-go/pull/632)

### Document & Comment
- [Add comment for common directory](https://github.com/apache/dubbo-go/pull/530)
- [Add comments for config_center](https://github.com/apache/dubbo-go/pull/545)
- [Update the comments in metrics](https://github.com/apache/dubbo-go/pull/547)
- [Add comments for config](https://github.com/apache/dubbo-go/pull/579)
- [Updated the dubbo-go-ext image](https://github.com/apache/dubbo-go/pull/581)
- [Add comment for cluster](https://github.com/apache/dubbo-go/pull/584)
- [Update the comments in filter directory](https://github.com/apache/dubbo-go/pull/586)
- [Add comment for metadata](https://github.com/apache/dubbo-go/pull/588)
- [Update the comments in protocol directory](https://github.com/apache/dubbo-go/pull/602)
- [Add comments for remoting](https://github.com/apache/dubbo-go/pull/605)
- [Update the comments in registy directory](https://github.com/apache/dubbo-go/pull/589)

## 1.4.0
### New Features

- [Condition router](https://github.com/apache/dubbo-go/pull/294)
- [Context support](https://github.com/apache/dubbo-go/pull/330)
- [Opentracing & transfer context end to end for jsonrpc protocol](https://github.com/apache/dubbo-go/pull/335)
- [Opentracing & transfer context end to end for dubbo protocol](https://github.com/apache/dubbo-go/pull/344)
- [Grpc tracing for client and server](https://github.com/apache/dubbo-go/pull/397)
- [Nacos config center](https://github.com/apache/dubbo-go/pull/357)
- [Prometheus support](https://github.com/apache/dubbo-go/pull/342)
- [Support sign and auth for request](https://github.com/apache/dubbo-go/pull/323)
- [Healthy instance first router](https://github.com/apache/dubbo-go/pull/389)
- [User can add attachments for dubbo protocol](https://github.com/apache/dubbo-go/pull/398)
- [K8s as registry](https://github.com/apache/dubbo-go/pull/400)
- [Rest protocol](https://github.com/apache/dubbo-go/pull/352)

### Enhancement

- [Reduce the scope of lock in zk listener](https://github.com/apache/dubbo-go/pull/346)
- [Trace error of getGettyRpcClient](https://github.com/apache/dubbo-go/pull/384)
- [Refactor to add base_registry](https://github.com/apache/dubbo-go/pull/348)
- [Do not listen to directory event if zkPath ends with providers/ or consumers/](https://github.com/apache/dubbo-go/pull/359)

### Bugfixes

- [Handle the panic when invoker was destroyed](https://github.com/apache/dubbo-go/pull/358)
- [HessianCodec failed to check package header length](https://github.com/apache/dubbo-go/pull/381)


## 1.3.0

### New Features

- [Add apollo config center support](https://github.com/apache/dubbo-go/pull/250)
- [Gracefully shutdown](https://github.com/apache/dubbo-go/pull/255)
- [Add consistent hash load balance support](https://github.com/apache/dubbo-go/pull/261)
- [Add sticky connection support](https://github.com/apache/dubbo-go/pull/270)
- [Add async call for dubbo protocol](https://github.com/apache/dubbo-go/pull/272)
- [Add generic implement](https://github.com/apache/dubbo-go/pull/291)
- [Add request timeout for method](https://github.com/apache/dubbo-go/pull/284)
- [Add grpc protocol](https://github.com/apache/dubbo-go/pull/311)

### Enhancement

- [The SIGSYS and SIGSTOP are not supported in windows platform](https://github.com/apache/dubbo-go/pull/262)
- [Error should be returned when `NewURL` failed](https://github.com/apache/dubbo-go/pull/266)
- [Split config center GetConfig method](https://github.com/apache/dubbo-go/pull/267)
- [Modify closing method for dubbo protocol](https://github.com/apache/dubbo-go/pull/268)
- [Add SetLoggerLevel method](https://github.com/apache/dubbo-go/pull/271)
- [Change the position of the lock](https://github.com/apache/dubbo-go/pull/286)
- [Change zk version and add base_registry](https://github.com/apache/dubbo-go/pull/355)

### Bugfixes

- [Fix negative wait group count](https://github.com/apache/dubbo-go/pull/253)
- [After disconnection with ZK registry, cosumer can't listen to provider changes](https://github.com/apache/dubbo-go/pull/258)
- [The generic filter and default reference filters lack ','](https://github.com/apache/dubbo-go/pull/260)
- [Url encode zkpath](https://github.com/apache/dubbo-go/pull/283)
- [Fix jsonrpc about HTTP/1.1](https://github.com/apache/dubbo-go/pull/327)
- [Fix zk bug](https://github.com/apache/dubbo-go/pull/346)
- [HessianCodec failed to check package header length](https://github.com/apache/dubbo-go/pull/381)

## 1.2.0

### New Features

- [Add etcdv3 registry support](https://github.com/apache/dubbo-go/pull/148)
- [Add nacos registry support](https://github.com/apache/dubbo-go/pull/151)
- [Add fail fast cluster support](https://github.com/apache/dubbo-go/pull/140)
- [Add available cluster support](https://github.com/apache/dubbo-go/pull/155)
- [Add broadcast cluster support](https://github.com/apache/dubbo-go/pull/158)
- [Add forking cluster support](https://github.com/apache/dubbo-go/pull/161)
- [Add service token authorization support](https://github.com/apache/dubbo-go/pull/202)
- [Add accessLog filter support](https://github.com/apache/dubbo-go/pull/214)
- [Add tps limit support](https://github.com/apache/dubbo-go/pull/237)
- [Add execute limit support](https://github.com/apache/dubbo-go/pull/246)
- [Move callService to invoker & support attachments](https://github.com/apache/dubbo-go/pull/193)
- [Move example in dubbo-go project away](https://github.com/apache/dubbo-go/pull/228)
- [Support dynamic config center which compatible with dubbo 2.6.x & 2.7.x and commit the zookeeper impl](https://github.com/apache/dubbo-go/pull/194)

### Enhancement

- [Split gettyRPCClient.close and gettyRPCClientPool.remove in protocol/dubbo/pool.go](https://github.com/apache/dubbo-go/pull/186)
- [Remove client from pool before closing it](https://github.com/apache/dubbo-go/pull/190)
- [Enhance the logic for fetching the local address](https://github.com/apache/dubbo-go/pull/209)
- [Add protocol_conf default values](https://github.com/apache/dubbo-go/pull/221)
- [Add task pool for getty](https://github.com/apache/dubbo-go/pull/141)
- [Update getty: remove read queue](https://github.com/apache/dubbo-go/pull/137)
- [Clean heartbeat from PendingResponse](https://github.com/apache/dubbo-go/pull/166)

### Bugfixes

- [GettyRPCClientPool remove deadlock](https://github.com/apache/dubbo-go/pull/183/files)
- [Fix failover cluster bug and url parameter retries change int to string type](https://github.com/apache/dubbo-go/pull/195)
- [Fix url params unsafe map](https://github.com/apache/dubbo-go/pull/201)
- [Read protocol config by map key in config yaml instead of protocol name](https://github.com/apache/dubbo-go/pull/218)
- *Fix dubbo group issues #238* [pr #243](https://github.com/apache/dubbo-go/pull/243) and [pr #244](https://github.com/apache/dubbo-go/pull/244)
- [Fix bug in reference_config](https://github.com/apache/dubbo-go/pull/157)
- [Fix high memory bug in zookeeper listener](https://github.com/apache/dubbo-go/pull/168)

## 1.1.0

### New Features

- [Support Java bigdecimal](https://github.com/apache/dubbo-go/pull/126)
- [Support all JDK exceptions](https://github.com/apache/dubbo-go/pull/120)
- [Support multi-version of service](https://github.com/apache/dubbo-go/pull/119)
- [Allow user set custom params for registry](https://github.com/apache/dubbo-go/pull/117)
- [Support zookeeper config center](https://github.com/apache/dubbo-go/pull/99)
- [Failsafe/Failback  Cluster Strategy](https://github.com/apache/dubbo-go/pull/136)

### Enhancement

- [Use time wheel instead of time.After to defeat timer object memory leakage](https://github.com/apache/dubbo-go/pull/130)

### Bugfixes

- [Preventing dead loop when got zookeeper unregister event](https://github.com/apache/dubbo-go/pull/129)
- [Delete ineffassign](https://github.com/apache/dubbo-go/pull/127)
- [Add wg.Done() for mockDataListener](https://github.com/apache/dubbo-go/pull/118)
- [Delete wrong spelling words](https://github.com/apache/dubbo-go/pull/107)
- [Use sync.Map to defeat from gettyClientPool deadlock](https://github.com/apache/dubbo-go/pull/106)
- [Handle panic when function args list is empty](https://github.com/apache/dubbo-go/pull/98)
- [url.Values is not safe map](https://github.com/apache/dubbo-go/pull/172)
