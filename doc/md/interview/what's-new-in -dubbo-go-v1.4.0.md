# [dubbo-go 1.4.0 版本发布，支持 K8s 注册中心、rest 协议](https://blog.csdn.net/weixin_45583158/article/details/105132322)


2020-03-26 09:30:00

得益于社区活跃的支持，2020 年 3 月 25 日 我们发布了一个让人兴奋的版本——dubbo-go v1.4.0。除了继续支持已有的 Dubbo 的一些特性外， dubbo-go 开始了一些自己的创新尝试。

这个版本，最大的意义在于，做了一些支持云原生的准备工作。比如说，社区在探讨了很久的 k8s 落地之后，终于拿出来了使用 k8s 作为注册中心的解决方案。

其次一个比较大的改进是--我们在可观测性上也迈出了重要的一步。在这之前，dubbo-go只提供了日志这么一个单一手段，内部的信息比较不透明，这个版本将有很大的改善。

最后一个令人心动的改进是，我们支持了 REST 协议。

## 1\. K8s 注册中心

dubbo-go 注册中心的本质为K/V型的数据存储。当前版本实现了以 Endpoint 为维度在 k8s API Server 进行服务注册和发现的方案【下文简称 Endpoint 方案】，架构图如下。

![](../../pic/interview/what's-new-in-dubbo-go-v1.4.0-a.png "dubbo-go-k8s.png")

Endpoint 方案，首先将每个 dubbo-go 进程自身服务信息序列化后，通过 Kubernetes 提供的 Patch 的接口写入在自身 Pod 对象的 Annotation 中。其次，通过 Kubernetes 的 Watch 接口观察集群中本 Namespace 内带有某些固定lable \[见上图\] Pod 的Annotation 信息的更新，处理服务健康检查、服务上下线等情况并实时更新本地缓存。整体流程仅使用 Kubernetes 原生 API 完成将 Kubernetes 作为注册中心的功能特性。

这个方案非常简洁，不需要实现额外的第三方模块，也不需要对 Dubbo 业务作出改动，仅仅把 k8s 当做部署平台，依赖其容器管理能力，没有使用其 label selector 和 service 等服务治理特性。如果站在 k8s Operator 的角度来看，Operator 方案的优点即 Endpoint 方案的缺点，Endpoint 方案无法使用 k8s 的健康检查能力，亦没有使用 k8s service 的事件监听能力，每个 consumer 冗余监听一些不必要监听的事件，当 Endpoint 过多时会加大 API Server 的网络压力。

目前 dubbo-go 社区其实已经有了 operator 版本注册中心的技术方案， 后续版本【计划版本是 v1.6】的 dubbo-go 会给出其实现。相比当前实现，operator 方案开发和线上维护成本当然上升很多。二者如同硬币的两面，社区会让两种方式会共存，以满足不同 level 的使用者。

注意: 因 Pod 被调度而 IP 发生变化时，当前版本的 configuration 以及 router config 模块暂时无法动态更新。这有待于我们进一步解决。

参考范例\[1\].

## 2\. tracing 和 metric

可观测性是微服务重要的一环，也是我们1.4版本着力支持的部分。在1.4版本中，我们主要在 tracing 和 metric 两个方向提供了支持。

为了支持 tracing 和 metric，关键的一点是支持context在整个调用过程中传递。为此我们解决了context跨端传递的问题。目前用户可以在接口中声明 context 并且设置值，dubbo-go 在底层完成 context 内容从 client 传递到 server 的任务。

![](../../pic/interview/what's-new-in-dubbo-go-v1.4.0-b.png "image.png")

在 metric 方面，dubbo-go 开始支持 Prometheus 采集数据了。目前支持 Prometheus中 的 Histogram 和 Summary。用户也可以通过扩展 Reporter 接口来自定义数据采集。

在 tracing 方面，目前 dubbo-go 的设计是采用 opentracing 作为统一的 API，在该 API 的基础上，通过在 client 和 server 之中传递 context，从而将整个链路串起来。用户可以采用任何支持 opentracing API 的监控框架来作为实现，例如 zipkin，jaeger 等。

## 3\. rest协议支持

Dubbo 生态的应用与其他生态的应用互联互通，一直是 dubbo-go 社区追求的目标。dubbo-go v1.3 版本已经实现了 dubbo-go 与 grpc 生态应用的互联互通，若想与其他生态如 Spring 生态互联互通，借助 rest 协议无疑是一个很好的技术手段。

Rest 协议是一个很强大并且社区呼声很高的特性，它能够有效解决 open API，前端通信，异构系统通信等问题。比如，如果你的公司里面有一些陈年代码是通过 http 接口来提供服务的，那么使用我们的 rest 协议就可以无缝集成了。

通过在 dubbo-go 中发布 RESTful 的接口的应用可以调用任意的 RESTful 的接口，也可以被任何客户端以 http 的形式调用，框架图如下：

  
![](../../pic/interview/what's-new-in-dubbo-go-v1.4.0-c.png "dubbo-go-rest.png")

在设计过程中，考虑到不同的公司内部使用的 web 框架并不相同，所以我们允许用户扩展自己 rest server （ web 框架在 dubbo-go的封装）的实现，当然，与 rest server 相关的，诸如 filter 等，都可以在自己的 rest server 实现内部扩展。

## 4\. 路由功能增强

路由规则在发起一次 RPC 调用前起到过滤目标服务器地址的作用，过滤后的地址列表，将作为消费端最终发起 RPC 调用的备选地址。v1.4 版本的 dubbo-go 实现了 Condition Router 和 Health Instance First Router，将在后面版本中陆续给出诸如 Tag Router 等剩余 Router 的实现。

### 4.1 条件路由

条件路由，是 dubbo-go 中第一个支持的路由规则，允许用户通过配置文件及远端配置中心管理路由规则。

与之相似的一个概念是 dubbo-go 里面的 group 概念，但是条件路由提供了更加细粒度的控制手段和更加丰富的表达语义。比较典型的使用场景是黑白名单设置，灰度以及测试等。

参考范例\[2\]。

### 4.2 健康实例优先路由

在 RPC 调用中，我们希望尽可能地将请求命中到那些处理能力快、处于健康状态的实例，该路由的功能就是通过某种策略断定某个实例不健康，并将其排除在候选调用列表，优先调用那些健康的实例。这里的"健康"可以是我们自己定义的状态，默认实现即当错误比例到达某一个阈值时或者请求活跃数大于上限则认为其不健康，允许用户扩展健康检测策略。

在我们服务治理里面，核心的问题其实就在于如何判断一个实例是否可用。无论是负载均衡、

熔断还是限流，都是对这个问题的解答。所以，这个 feature 是一个很好的尝试。因为我们接下来计划提供的特性，基于规则的限流以及动态限流，都是要解决“如何断定一个实例是否可用”这么一个问题。

所以欢迎大家使用这个特性，并向社区反馈各自设定的健康指标。这对我们接下来的工作会有很大的帮助。

## 5\. hessian 协议增强

相较于 dubbo 的 Java 语言以及其他多语言版本，dubbo-go 社区比较自豪的地方之一就是：无论底层网络引擎还是原生使用的 hessian2 协议，以及整体服务治理框架，都由 dubbo-go 社区从零开发并维护。v1.4 版本的 dubbo-go 对 hessian2 协议又带来了诸多新 feature。

### 5.1 支持 dubbo 协议的 attachments

在 dubbo-go中，attachments 机制用于传递业务参数之外的附加信息，是在客户端和服务端之间传递非业务参数信息的重要方式。

hessian 编码协议将之编码在 body 内容的后面进行传输，dubbo-go-hessian2 之前并不支持读/写 attachments，在多个使用方【如蚂蚁金服】的要求下，dubbo-go-hessian2 以兼容已有的使用方式为前提，支持了 attachments 的读/写。

Request 和 Response 的 struct 中定义了 attachments 的 map，当需要使用 attachments，需要由使用方构造这两种类型的参数或者返回对象。否则，将无法在hessian的传输流中获取和写入attachments。

另外，利用 dubbo-go 调用链中传输 context 的功能，用户已经可以在服务方法中通过 context 添加 attachments了。

### 5.2 支持忽略非注册 pojo 的解析方式

由于 hessian 编码协议与 Java 的类型高度耦合，在 golang 的实现中会相对比较麻烦，需要有指明的对应类型。dubbo-go-hessian2 的实现方式是：定义 POJO 接口，要求实现 JavaClassName 方法来供程序获取 Java 对应的类名。这导致了接收到包含未注册类的请求时，将会无法解析而报错，这个问题以前是无法解决的。

但是，有一些使用场景如网关或者 service mesh 的 sidecar，需要在不关心 Java 类的具体定义的情况下，像 http读取 header 信息一样仅仅读取 dubbo 请求的附加信息，将 dubbo/dubbo-go 请求转发。通过该 feature，网关/sidecar 并不关注请求的具体内容，可以在解析请求的数据流时跳过无法解析的具体类型，直接读取 attachments 的内容。

该实现通过在 Decoder 中添加的 skip 字段，对每一个 object 做出特殊处理。

### 5.3 支持 java.math.BigInteger 和 java.math.BigDecimal

在 Java 服务中，java.math.BigInteger 和 java.math.BigDecimal 是被频繁使用的数字类型，hessian 库将它们映射为 github.com/dubbogo/gost/math/big 下的对应类型。

### 5.4 支持 ‘继承’ 和忽略冗余字段

由于 go 没有继承的概念，所以在之前的版本，Java 父类的字段不被 dubbo-go-hessian2 所支持。新版本中，dubbo-go-hessian2 将Java来自父类的字段用匿名结构体对应，如：

```cpp
type Dog struct {
    Animal
    Gender  string
    DogName string `hessian:"-"`
}
```

同时，就像 json 编码中通过 `immediately` 可以在序列化中忽略该字段，同理，通过 `hessian:"-"` 用户也可以让冗余字段不参与 hessian 序列化。  

目前，上述四个特性已被某 Go 版本的 sidecar 集成到其商业版本中提供商业服务。

## 6\. Nacos 配置中心

配置中心是现代微服务架构里面的核心组件，现在 dubbo-go 提供了对配置中心的支持。

![](../../pic/interview/what's-new-in-dubbo-go-v1.4.0-d.png "image.png")

Nacos 作为一个易于构建云原生应用的动态服务发现、配置管理和服务管理平台，在该版本终于作为配置中心而得到了支持。

参考范例\[3\].

## 7\. 接口级签名认证

Dubbo 鉴权认证是为了避免敏感接口被匿名用户调用而在 SDK 层面提供的额外保障。用户可以在接口级别进行定义是否允许匿名调用，并对调用方进行验签操作，对于验签不通过的消费端，禁止调用。

![](../../pic/interview/what's-new-in-dubbo-go-v1.4.0-e.png "image.png")

如上图，总体实现基于 AK/SK 机制，应用通过 HTTPS 通信，启动时向鉴权服务拉取，定期更新。且允许用户自定义获取 AK/SK 的源，在 RPC 层面保障安全性。

## 8\. 回顾与展望

目前 dubbo-go 已经到了一个比较稳定成熟的状态。在接下来的版本里面，我们将集中精力在云原生上。下一个版本，我们将首先实现应用维度的服务注册，这是一个和现有注册模型完全不同的新的注册模型。也是我们朝着云原生努力的一个关键版本。

在可观测性上，我们计划在整个 dubbo-go 的框架内，引入更多的埋点，收集更加多的内部状态。这需要实际生产环境用户的使用反馈，从而知道该如何埋点，收集何种数据。

在限流和熔断上，可以进一步扩展。当下的限流算法，是一种静态的算法--限流参数并没有实时根据当前服务器的状态来推断是否应该限流，它可能仅仅是用户的经验值。其缺点在于，用户难以把握应该如何配置，例如 TPS 究竟应该设置在多大。所以计划引入一种基于规则的限流和熔断。这种基于规则的限流和熔断，将允许用户设置一些系统状态的状态，如 CPU 使用率，磁盘 IO，网络 IO 等。当系统状态符合用户规则时，将触发熔断。

目前这些规划的 任务清单\[4\]，都已经放入在 dubbo-go 项目的 issue 里面，欢迎感兴趣的朋友认领参与开发。dubbo-go 社区在 **钉钉群 23331795** 欢迎你的加入。

**文中链接：**

\[1\] https://github.com/apache/dubbo-samples/tree/master/golang/registry/kubernetes

\[2\] https://github.com/dubbogo/dubbo-samples/tree/master/golang/router/condition

\[3\] https://github.com/dubbogo/dubbo-samples/tree/master/golang/configcenter/nacos

\[4\] https://github.com/apache/dubbo-go/milestone/1

**参考阅读：**

*   [从lstio的角度谈微服务的一些误区](https://blog.csdn.net/weixin_45583158/article/details/105085686)  
    
*   [Go语言如何实现stop the world？](https://blog.csdn.net/weixin_45583158/article/details/104912555)
    
*   [关于Golang GC的一些误解--真的比Java算法更领先吗？](https://blog.csdn.net/weixin_45583158/article/details/100143593)
    
*   [Swift程序员对Rust印象：内存管理](https://blog.csdn.net/weixin_45583158/article/details/104853360)
    
*   [JDK 14发布，空指针错误改进正式落地](https://blog.csdn.net/weixin_45583158/article/details/104981073)  