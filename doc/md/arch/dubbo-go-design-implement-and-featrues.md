# [dubbo-go 的开发、设计与功能介绍](https://www.infoq.cn/article/7JIDIi7pfwDDk47EpaXZ)

## dubbo-go 的前世今生

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-a.png)

dubbo-go 是目前 Dubbo 多语言生态最火热的项目。dubbo-go 最早的版本应该要追溯到 2016 年，由社区于雨同学编写 dubbo-go 的初版。当时很多东西没有现成的轮子，如 Go 语言没有像 netty 一样的基于事件的网络处理引擎、 hessian2 协议没有 Go 语言版本实现，加上当时 Dubbo 也没有开始重新维护。所以从协议库到网络引擎，再到上层 dubbo-go ，其实都是从零开始写的。

在 2018 年，携程开始做 Go 语言的一些中间件以搭建内部的 Go 语言生态，需要有一个 Go 的服务框架可以与携程的现有 dubbo soa 生态互通。所以由我负责重构了 dubbo－go 并开源出这个版本。当时调研了很多开源的 Go 语言服务框架，当时能够支持 hessian2 协议的并跟 Dubbo 可以打通的仅找到了当时于雨写的 dubbo-go 早期版本。由于携程对社区版本的 Dubbo 做了挺多的扩展，源于对扩展性的需求我们 Go 语言版本需要一个更易于扩展的版本，加上当时这个版本本身的功能也比较简单，所以我们找到了作者合作重构了一个更好的版本。经过了大半年时间，在上图第三阶段 19 年 6 月的时候，基本上已经把 dubbo-go 重构了一遍，总体的思路是参考的 Dubbo 整体的代码架构，用 Go 语言完全重写了一个完整的具备服务端跟消费端的 Golang rpc/ 微服务框架。

后来我们将重构后的版本 dubbo-go 1.0 贡献给 Apache 基金会，到现在已经过去了两个多月的时间，近期社区发布了 1.1 版本。目前为止，已经有包括携程在内的公司已经在生产环境开始了试用和推广。

## Start dubbo-go

现在的 dubbo-go 已经能够跟 Java 版本做比较好的融合互通，同时 dubbo-go 自身也是一个完成的 Go 语言 rpc/ 微服务框架，它也可以脱离 java dubbo 来独立使用。

这边简单介绍一下用法，写一个 hello world 的例子。

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-b.png)

上图是一个简单的 java service ，注册为一个 Dubbo 服务，是一个简单的获取用户信息的例子。

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-c.png)

上图是 dubbo-go 的客户端，来订阅和调用这个 Java 的 Dubbo 服务。Go 语言客户端需要显式调用 SetConsumerService 来注册需要订阅的服务，然后通过调用 dubbo-go-hessian2 库的 registerPOJO 方法来注册 user 对象，做 Java 和 Go 语言之间的自定义 pojo 类型转换。具体的服务调用方法就是声明一个的 GetUser 闭包，便可直接调用。

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-d.png)

上图，同样的可以基于 dubbo-go 发布一个 GetUser 的服务端，使用方式类似，发布完后可以被 dubbo java 的客户端调用。

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-e.png)

如上图所示，现在已经做到了这样一个程度，同样一份 dubbo-go 客户端代码，可以去调用 dubbo-go 的服务端，也可以去调用 Dubbo Java 的服务端；同样一份 dubbo-go 的服务端代码，可以被 dubbo-go 客户端和 Java 客户端调用，所以基本上使用 Dubbo 作为 PPC 框架的 Go 语言应用跟 Java 应用是没有什么阻碍的，是完全的跨语言 RPC 调用。更重要的是 dubbo-go 继承了 Dubbo 的许多优点，如易于扩展、服务治理功能强大，大家在用 Go 语言开发应用的过程中，如果也遇到类似需要与 Dubbo Java 打通的需求，或者需要找一个服务治理功能完备的 Go 微服务框架，可以看下我们 dubbo-go 项目。

## dubbo-go 的组成项目

下面介绍一下 dubbo-go 的组成项目，为了方便可以被其他项目直接复用， dubbo-go 拆分成了多个项目，并全部以 Apache 协议开源。

**apache/dubbo-go**

dubbo-go 主项目， Dubbo 服务端、客户端完整 Go 语言实现。

**apache/dubbo-go-hession2**

目前应用最广泛，与 Java 版本兼容程度最高的 hessian2 协议 Go 语言实现，已经被多个 GolangRPC & Service Mesh 项目使用。

**dubbo-go/getty**

dubbo-go 异步网络 I/O 库，将网络处理层解耦。

**dubbo-go/gost**

基本类库，定义了 timeWheel、hashSet、taskPool 等。

**dubbo-go/dubbo-go-benchmark**

用于对 dubbo-go 进行简单的压力测试，性能测试。

**apache/dubbo-go-hessian2**

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-f.png)

先简单介绍一下 dubbo-go-hessian2 项目。该项目就是 hessian2 协议的 Go 语言实现，最基本的可以将 Java 的基本数据类型和复杂数据类型（如一些包装类和 list 接口实现类）与 golang 这边对应。

详情可以参考： [https://github.com/hessian-group/hessian-type-mapping](https://github.com/hessian-group/hessian-type-mapping)

另外 Dubbo Java 服务端可以不捕获异常，将异常类通过 hession2 协议序列化通过网络传输给消费端，消费端进行反序列化对该异常对象并进行捕获。我们经过一段时间的整理，目前已经支持在 Go 消费端定义对应 Java 的超过 40 种 exception 类，来实现对 Java 异常的捕获，即使用 dubbo-go 也可以做到直接捕获 Java 服务端抛出的异常。

另外对于 Java 端 BigDecimal 高精度计算类的支持。涉及到一些金融相关的计算会有类似的需求，所以也对这个类进行了支持。

其他的，还有映射 java 端的方法别名，主要的原因是 Go 这边语言的规约，需要被序列化的方法名必须是首字母大写。而 Java 这边没有这种规范，所以我们加了一个 hessian 标签的支持，可以允许用户手动映射 Java 端的方法名称。

基本上现在的 dubbo-go 已经满足绝大多数与 Java 的类型互通需求，我们近期也在实现对 Java 泛型的支持。

**dubbo-go/getty**

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-g.png)

Go 语言天生就是一个异步网络 I/O 模型，在 linux 上 Go 语言写的网络服务器也是采用的 epoll 作为最底层的数据收发驱动, 这跟 java 在 linux 的 nio 实现是一样的。所以 Go 语言的网络处理天生就是异步的。我们需要封装的其实是基于 Go 的异步网络读写以及之后的处理中间层。getty 将网络数据处理分为三层，入向方向分别经过对网络 i/o 封装的 streaming 层、根据不同协议对数据进行序列化反序列化的 codec 层，以及最后数据上升到需要上层消费的 handler 层。出向方向基本与入向经过的相反。每个链接的 IO 协程是成对出现的，比如读协程负责读取、 codec 逻辑然后数据到 listener 层，然后最后的事件由业务协程池来处理。

该项目目前是与 dubbo-go 解耦出来的，所以大家如果有类似需求可以直接拿来用，目前已经有对于 tcp/udp/websocket 的支持。

**Apache / dubbo-go**

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-h.png)

dubbo-go 主项目，我们重构的这一版主要是基于 Dubbo 的分层代码设计，上图是 dubbo-go 的代码分层。基本上与 Java 版本 Dubbo 现有的分层一致，所以 dubbo－go 也继承了 Dubbo 的一些优良特性，比如整洁的代码架构、易于扩展、完善的服务治理功能。

我们携程这边，使用的是自己的注册中心，可以在 dubbo-go 扩展机制的基础上灵活扩展而无需去改动 dubbo-go 的源代码。

## dubbo-go 的功能介绍

**dubbo-go 已实现功能**

目前 dubbo-go 已经实现了 Dubbo 的常用功能（如负责均衡、集群策略、服务多版本多实现、服务多注册中心多协议发布、泛化调用、服务降级熔断等），其中服务注册发现已经支持 zookeeper/etcd/consul/nacos 主流注册中心。这里不展开详细介绍，目前 dubbo-go 支持的功能可以查看项目 readme 中的 feature list ，详情参考： [https://github.com/apache/dubbo-go#feature-list](https://github.com/apache/dubbo-go#feature-list)

目前社区正在开发中的功能，主要是早期用户使用过程中提出的一些需求，也是生产落地一些必需的需求，如监控、调用链跟踪以及服务路由、动态配置中心等更高级的服务治理需求。

**dubbo-go 功能介绍之泛化调用**

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-i.png)

这里详细做几个重点功能的介绍。首先是泛化调用，如上图，这个也是社区同学提的需求。该同学公司内部有很多 Dubbo 服务，他们用 Go 做了一个 api gateway 网关，想要把 Dubbo 服务暴露成外网 http 接口。因为内部的 Dubbo 服务比较多，不可能每一个 Dubbo 服务都去做一个消费端接口去做适配，这样的话一旦服务端改动，客户端也要改。所以他这边的思路是做基于 dubbo-go 做泛化调用， api-gateway 解析出外网请求的地址，解析出想要调用的 Dubbo 服务的目标。基于 dubbo-go consumer 泛化调用指定 service、method ，以及调用参数。

具体的原理是， dubbo-go 这边作为消费端，实际会通过本地 genericService.invoke 方法做代理，参数里面包含了 service name，method name ，还包含被调用目标 service 需要的参数类型、值等数据，这些数据后面会通过 dubbo-go-hession2 做转换，会将内容转化成 map 类型，经过网络发送到对应的 Java 服务端，然后 Java 那边是接收的 map 类型的参数，会自动反序列化成自己的 pojo 类型。这样就实现了 dubbo-go 作为客户端，泛化调用 Dubbo 服务端的目的。

**dubbo-go 功能介绍之降级熔断**

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-j.png)

降级熔断这边是基于的是大家比较熟悉的 hystrix 的 Go 语言版本，基于 hystrix ，用户可以定义熔断规则和降级触发的代码段。降级熔断支持是作为一个独立的 dubbo-go filter ，可以灵活选择是否启用，如果不启用就可以在打包的时候不将依赖引入。Filter 层是 dubbo-go 中对于请求链路的一个责任链模式抽象，目前有许多功能都是基于动态扩展 filter 链来实现的，包括 trace、leastactive load balacne、log 等。降级熔断设计成一个服务调用端独立的 filter 可以灵活满足调用端视角对于微服务架构中“防雪崩“的服务治理需求。

**dubbo-go 功能介绍之动态配置**

关于动态配置中心， Dubbo 的 2.6 到 2.7 版本做了一个比较大的变化，从之前的 url 配置形式过渡到了支持配置中心 yaml 格式配置的形式，治理粒度也从单服务级别的配置支持到了应用级别的配置，不过在 2.7 版本中还是兼容 2.6 版本 url 形式进行服务配置的。dubbo-go 这边考虑到跟 Dubbo2.6 和 2.7 的互通性，同样支持 url 和配置文件方式的服务配置，同时兼容应用级别和服务级别的配置，跟 dubbo 保持一致，目前已经实现了 zookeeper 和 apollo 作为配置中心的支持。

## dubbo-go roadmap 2019-2020

![dubbo-go 的开发、设计与功能介绍](../../pic/arch/dubbo-go-design-implement-and-featrues-k.png)

最后是大家比较关注的，社区关于 dubbo-go 2019 年下半年的计划，目前来看主要还是现有功能的补齐和一些问题的修复，我们的目标就是首先做到 Java 和 Go 在运行时的兼容互通和功能的一致，其次是查漏补缺 dubbo-go 作为一个完整 Go 语言微服务框架在功能上的可以改进之处。

另外值得关注的一点是，预计今年年底， dubbo-go 会发布一个支持 kubernetes 作为注册中心的扩展，积极拥抱云原生生态。关于云原生的支持，社区前期做了积极的工作，包括讨论关于 dubbo-go 与 Service Mesh 的关系以及在其中的定位，可以肯定的是， dubbo-go 将会配合 Dubbo 社区在 Service Mesh 方向的规划并扮演重要角色，我们初步预计会在明年给出与 Service Mesh 开源社区项目集成的方案，请大家期待。

dubbo-go 社区目前属于快速健康成长状态，从捐赠给 Apache 后的不到 3 个月的时间里，吸引了大批量的活跃开发者和感兴趣的用户，欢迎各位同道在使用或者学习中遇到问题能够来社区讨论或者给予指正，也欢迎对 dubbo-go 有潜在需求或者对 dubbo-go 感兴趣的同道能加入到社区中。

**作者介绍**：

何鑫铭，目前就职于携程，基础中台研发部技术专家，dubbo-go 共同发起人、主要作者，Apache Dubbo committer，关注互联网中台以及中间件领域。