# [Dubbo-go v1.5.1 发布，Apache Dubbo 的 Go 实现](https://www.oschina.net/news/118469/dubbo-go-1-5-1-released)

Dubbo-go 团队近期发布了 Dubbo-go v1.5.1，Dubbo-go 是 Apache Dubbo 项目的 Go 实现。

根据团队的介绍，虽然 v1.5.1 是 v1.5 的一个子版本，但相比于 v1.5.0， 社区还是投入了很大人力添加了如下重大改进。

## 1 应用维度注册模型

在新模型 release 后，团队发现 Provider 每个 URL 发布元数据都会注册 ServiceInstance，影响性能需要优化。

优化方案是： 去除 ServiceDiscoveryRegistry 中注册 ServiceInstance 的代码，在 config_loader 中的 loadProviderConfig 方法的最后注册 ServiceInstance 具体步骤： 

1、获取所有注册的 Registry，过滤出 ServiceDiscoveryRegistry，拿取所有 ServiceDiscovery。 
2、创建 ServiceInstance。 
3、每个 ServiceDiscovery 注册 ServiceInstance。

保证 Provider 在注册成功之后，才暴露元数据信息。

## 2 支持基于 Seata 的事务

基于 Seata 扩展实现。通过增加过滤器，在服务端接收 xid 并结合 seata-golang 达到支持分布式事务的目的。 从而使 Dubbo-go 在分布式场景下，让用户有更多的选择，能适应更多的个性化场景。

开发团队在 dubbo-samples 中给出了 事务测试用例 。

## 3 多注册中心集群负载均衡

对于多注册中心订阅的场景，选址时的多了一层注册中心集群间的负载均衡：

在 Cluster Invoker 这一级，支持的选址策略有：

- 指定优先级
- 同 zone 优先
- 权重轮询

## 3 传输链路安全性

该版本在传输链路的安全性上做了尝试，对于内置的 Dubbo getty Server 提供了基于 TLS 的安全链路传输机制。

为尽可能保证应用启动的灵活性，TLS Cert 的指定通过配置文件方式，具体请参见 Dubbo-go 配置读取规则与 TLS 示例：

## 4 路由功能增强

本次路由功能重点支持了 动态标签路由 和 应用/服务级条件路由。

### 4.1 动态标签路由

标签路由通过将某一个或多个服务的提供者划分到同一个分组，约束流量只在指定分组中流转，从而实现流量隔离的目的，可以作为蓝绿发布、灰度发布等场景的能力基础。

标签主要是指对 Provider 端应用实例的分组，目前有两种方式可以完成实例分组，分别是动态规则打标和静态规则打标，其中动态规则相较于静态规则优先级更高，而当两种规则同时存在且出现冲突时，将以动态规则为准。

### 4.1.1 动态规则打标

可随时在服务治理控制台下发标签归组规则

```yml
# governance-tagrouter-provider应用增加了两个标签分组tag1和tag2
# tag1包含一个实例 127.0.0.1:20880
# tag2包含一个实例 127.0.0.1:20881
---
  force: false
  runtime: true
  enabled: true
  key: governance-tagrouter-provider
  tags:
    - name: tag1
      addresses: ["127.0.0.1:20880"]
    - name: tag2
      addresses: ["127.0.0.1:20881"]
 ...
```

### 4.1.2 静态规则打标

可以在 server 配置文件的 tag 字段里设置

```yml
services:
  "UserProvider":
    registry: "hangzhouzk"
    protocol: "dubbo"
    interface: "com.ikurento.user.UserProvider"
    loadbalance: "random"
    warmup: "100"
    tag: "beijing"
    cluster: "failover"
    methods:
      - name: "GetUser"
        retries: 1
        loadbalance: "random"
```

consumer 添加 tag 至 attachment 即可

```go
ctx := context.Background()
attachment := make(map[string]string)
attachment["dubbo.tag"] = "beijing"
ctx = context.WithValue(ctx, constant.AttachmentKey, attachment)
err := userProvider.GetUser(ctx, []interface{}{"A001"}, user)
```

请求标签的作用域为每一次 invocation，使用 attachment 来传递请求标签，注意保存在 attachment 中的值将会在一次完整的远程调用中持续传递，得益于这样的特性，只需要在起始调用时，通过一行代码的设置，达到标签的持续传递。

### 4.1.3 规则详解

格式
Key 明确规则体作用到哪个应用。必填。
enabled=true 当前路由规则是否生效，可不填，缺省生效。
force=false 当路由结果为空时，是否强制执行，如果不强制执行，路由结果为空的路由规则将自动失效，可不填，缺省为 false。
runtime=false 是否在每次调用时执行路由规则，否则只在提供者地址列表变更时预先执行并缓存结果，调用时直接从缓存中获取路由结果。如果用了参数路由，必须设为 true，需要注意设置会影响调用的性能，可不填，缺省为 false。
priority=1 路由规则的优先级，用于排序，优先级越大越靠前执行，可不填，缺省为 0。
tags 定义具体的标签分组内容，可定义任意 n（n>=1）个标签并为每个标签指定实例列表。必填
name， 标签名称
addresses， 当前标签包含的实例列表
降级约定
request.tag=tag1 时优先选择 标记了 tag=tag1 的 provider。若集群中不存在与请求标记对应的服务，默认将降级请求 tag 为空的 provider；如果要改变这种默认行为，即找不到匹配 tag1 的 provider 返回异常，需设置 request.tag.force=true。
request.tag 未设置时，只会匹配 tag 为空的 provider。即使集群中存在可用的服务，若 tag 不匹配也就无法调用，这与约定 1 不同，携带标签的请求可以降级访问到无标签的服务，但不携带标签/携带其他种类标签的请求永远无法访问到其他标签的服务。

## 4.2 应用/服务级条件路由

可以在路由规则配置中配置多个条件路由及其粒度

Sample:

```yml
# dubbo router yaml configure file
routerRules:
  - scope: application
    key: BDTService
    priority: 1
    enable: false
    force: true
    conditions: ["host = 192.168.199.208 => host = 192.168.199.208 "]
  - scope: service
    key: com.ikurento.user.UserProvider
    priority: 1
    force: true
    conditions: ["host = 192.168.199.208 => host = 192.168.199.208 "]
```

### 4.2.1 规则详解

#### 各字段含义

- scope 表示路由规则的作用粒度，scope 的取值会决定 key 的取值。必填。
  - service 服务粒度
  - application 应用粒度
- Key 明确规则体作用在哪个服务或应用。必填。
  - scope=service 时，key 取值为[{group}/]{service}[:{version}]的组合
  - scope=application 时，key 取值为 application 名称
- enabled=true 当前路由规则是否生效，可不填，缺省生效。
- force=false 当路由结果为空时，是否强制执行，如果不强制执行，路由结果为空的路由规则将自动失效，可不填，缺省为 false。
- runtime=false 是否在每次调用时执行路由规则，否则只在提供者地址列表变更时预先执行并缓存结果，调用时直接从缓存中获取路由结果。如果用了参数路由，必须设为 true，需要注意设置会影响调用的性能，可不填，缺省为 false。
- priority=1 路由规则的优先级，用于排序，优先级越大越靠前执行，可不填，缺省为 0。
- conditions 定义具体的路由规则内容。必填。

## 5 回顾与展望

Dubbo-go 处于一个比较稳定成熟的状态。目前新版本正处于往云原生方向的尝试，应用服务维度注册是首先推出的功能，这是一个和之前模型完全不一样的新注册模型。该版本是朝云原生迈进新一步的关键版本。除此之外，包含在该版本也有一些之前提到的优化。

下一个版本 v1.5.2，本次的关注重点以通信模型改进为主，除此之外，与 2.7.x 的兼容性、易用性及质量保证也是本次关注的信息。

在服务发现，会支持更加多的方式，如：文件、Consul。 从而使 Dubbo-go 在服务发现场景下，让用户有更多的选择，能适应更多的个性化场景。

另外 易用性及质量保证，主要关注的是 samples 与自动化构建部分。可降低用户上手 Dubbo-go 的难度，提高代码质量。

目前下一个版本正在紧锣密鼓的开发中，具体规划及任务清单，都已经在 Github 上体现。

更多信息：https://github.com/apache/dubbo-go/releases/tag/v1.5.1
