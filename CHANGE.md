# Release Notes
---
## 1.5.6

### New Features
- [Add dubbo-go-cli telnet tool](https://github.com/apache/dubbo-go/pull/818)
- [Add Prox ImplementFunc to allow override impl](https://github.com/apache/dubbo-go/pull/1019)
- [Add read configuration path from the command line when start](https://github.com/apache/dubbo-go/pull/1039)

### Enhancement
- [introduce ConfigPostProcessor extension](https://github.com/apache/dubbo-go/pull/943)
- [Impl extension of two urls comparison](https://github.com/apache/dubbo-go/pull/854)
- [using event-driven to let router send signal to notify channel](https://github.com/apache/dubbo-go/pull/976)
- [lint codes](https://github.com/apache/dubbo-go/pull/941)

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

Milestone: [https://github.com/apache/dubbo-go/milestone/7](https://github.com/apache/dubbo-go/milestone/7?closed=1)

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
- [Fix consul can not destory](https://github.com/apache/dubbo-go/pull/788) [@LaurenceLiZhixin](https://github.com/LaurenceLiZhixin)

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
- [Fix consul can not destory](https://github.com/apache/dubbo-go/pull/788) [@LaurenceLiZhixin](https://github.com/LaurenceLiZhixin)      

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
