# [dubbo-go 可信 RPC 调用实现](https://mp.weixin.qq.com/s/30CjBKheCZClKaZCw1DRZA)

Apache Dubbo/Dubbo-Go 作为阿里巴巴开源的一款服务治理框架，因其适应 Java/Go 开发者面向接口的编程习惯、完全透明的调用方式、优越的性能以及强大的扩展性等优点，在国内使用非常广泛。除此之外，Dubbo 开源版本原生集成了很多开箱即用的服务治理功能，包括链路追踪，路由、负载均衡、服务注册发现、监控、认证等。

本文将讲解如何在 Dubbo/Dubbo-Go 中实现灵活、安全和高效的身份验证和授权方案。

## 定义可信

何为可信？可信的定义很广泛，依场景不同有不同的定义。在微服务架构中，每个服务都是无状态的，多个服务之间不可信，为了实现服务间更好地隔离，服务间应进行认证和鉴权。

如支付之类的业务场景，安全性敏感的服务会有限制匿名系统调用的需求，其他业务在接入该类敏感业务之前，需要通过审批方可正常调用，这就需要对这类敏感服务进行权限管控。尽管 Dubbo 开源版本中支持 Token 方式的鉴权实现，但是该实现方式总体来说安全性并不高，并且无法满足我们需要动态下发以及变更的灵活性需求。

针对于此，我们内部着重从巩固安全性和拓展灵活性层面重新设计了一套 Dubbo/Dubbo-Go 的服务间调用的鉴权认证功能。本文我们将主要从实现层面讲解其大致实现思路。

## 可信方案

总体而言，鉴权认证主要讨论以下两个问题：

1.身份认证：指验证应用的身份，每个应用在其生命周期内只有唯一身份，无法变更和伪造。

2.权限鉴定：根据身份信息鉴定权限是否满足调用。权限粒度可以进行控制。

我们通过 Access Key ID/Secret Access Key (后文简称为 AK/SK) 信息标识应用和应用之间的身份关系。例如上游 应用 A 依赖下游 服务 B 和 C，则 A 对 B 和 C 分别有一套 AK/SK，其相互独立没有任何关系，就算 A 服务 的 AK/SK 信息泄漏，也无法通过该 AK/SK 信息调用其他的服务。

在权限鉴定方面也借鉴了公有云开放 API 常用的 AK/SK 签名机制。在请求过程中使用 SK 签名生成 SigningKey，并通过 Dubbo 的 attachment 机制将额外的元数据信息以及 SigningKey 传输到服务端，交由服务端计算和验签，验签通过方能正常处理和响应。

签名过程主要通过如下三个方式进行加强 SigningKey 的可靠性和安全性。

1、验证请求者的身份，签名会通过对应应用的 SK 作为加密密钥对请求元数据(以及参数)进行加密，保证签名的唯一性和不可伪造性。
2、支持对参数进行计算签名，防止非法篡改，若请求参数在传输过程中遭到非法篡改，则收到请求后服务端验签匹配将失败，身份校验无法通过，从而防止请求参数被篡改。考虑到签名以及验签过程中加入请求参数的计算可能会影响性能，这个过程是可选的。
3、防止重放攻击，每一次请求生成的 SigningKey 都具有指定的有效时间，如请求被截获，该请求无法在有效时间外进行调用，一定程度避免了重放攻击。

同时为了去掉明文配置，防止 AK/SK 信息泄漏，我们通过鉴权系统分发和管理所有 AK/SK 信息。通过对接内部审批流程，达到流程化和规范化，需要鉴权的应用会通过启动获取的方式拉当前应用分发出去或者是已被授权的 AK/SK 信息。这种方式也带来了另一种好处，新增、吊销以及更新权限信息也无需重启应用。

## 可信流程

结合上面的这些需求和方案，整个接入和鉴权流程图如下所示：

![](../../pic/rpc/dubbo-go-trusted-RPC-call-implementation-1.png)

整体流程如下：

1、使用该功能的应用需要提前申请对应的证书，并向提供服务的应用提交申请访问工单，由双方负责人审批通过后，请求鉴权服务中心自动生成键值对。

2、开启鉴权认证的服务在应用启动之后，会运行一个后台线程，长轮询远鉴权服务中心，查询是否有新增权限变动信息，如果有则进行全量/增量的拉取。
3、上游应用在请求需要鉴权的服务时，会通过 SK 作为签名算法的 key，对本次请求的元数据信息甚至是参数信息进行计算得到签名，通过 Dubbo 协议 Attachment 字段传送到对端，除此之外还有请求时间戳、AK 信息等信息。
4、下游应用在处理鉴权服务时会对请求验签，验签通过则继续处理请求，否则直接返回异常。

其中需要说明的是第三步，使用鉴权服务的应用和鉴权服务中心的交互需通过 HTTPS 的双向认证，并在 TLS 信道上进行数据交互，保证 AK/SK 信息传输的安全性。

该方案目前已经有 Java/Go 实现，均已合并到 dubbo/dubbo-go。除了默认的 Hmac 签名算法实现之外，我们将签名和认证方法进行抽象，以 dubbo-go 中的实现为例：

```go
// Authenticator
type Authenticator interface {
    // Sign
    // give a sign to request
    Sign(protocol.Invocation, *common.URL) error
    // Authenticate
    // verify the signature of the request is valid or not
    Authenticate(protocol.Invocation, *common.URL) error
}
```

使用者可通过 SPI 机制定制签名和认证方式，以及适配公司内部基础设施的密钥服务下发 AK/SK。

## 示例

以 Helloworld 示例 中的代码接入当前社区版本中的默认鉴权认证功能实现为例：

### Helloworld 示例：

https://github.com/apache/dubbo-samples/tree/master/golang/helloworld/dubbo

在无需改变代码的情况下，只需要在配置上增加额外的相关鉴权配置即可，dubbo-go 服务端配置示例如下：

```yml
services:
  "UserProvider":
    # 可以指定多个registry，使用逗号隔开;不指定默认向所有注册中心注册
    registry: "hangzhouzk"
    protocol: "dubbo"
    # 相当于dubbo.xml中的interface
    interface: "com.ikurento.user.UserProvider"
    loadbalance: "random"
    # 本服务开启auth
    auth: "true"
    # 启用auth filter，对请求进行验签
    filter: "auth"
    # 默认实现通过配置文件配置AK、SK
    params:
      .accessKeyId: "SYD8-23DF"
      .secretAccessKey: "BSDY-FDF1"
    warmup: "100"
    cluster: "failover"
    methods:
      - name: "GetUser"
        retries: 1
        loadbalance: "random"
```

dubbo-go 客户端配置示例如下：

```yml
references:
  "UserProvider":
    # 可以指定多个registry，使用逗号隔开;不指定默认向所有注册中心注册
    registry: "hangzhouzk"
    protocol: "dubbo"
    interface: "com.ikurento.user.UserProvider"
    cluster: "failover"
    # 本服务开启sign filter，需要签名
    filter: "sign"
    # 默认实现通过配置文件配置AK、SK
    params:
      .accessKeyId: "SYD8-23DF"
      .secretAccessKey: "BSDY-FDF1"
    methods:
      - name: "GetUser"
        retries: 3
```

可以看到，dubbo-go 接入鉴权认证的功能也十分简单。需要补充说明的是，配置文件文件中 ak/sk 都加了特殊前缀 "."，是为了说明该字段是敏感信息，不能在发起网络请求时传输出去，相关代码可参阅 dubbo-go-pr-509。

dubbo-go-pr-509：
https://github.com/apache/dubbo-go/pull/509

## 总结

Apache Dubbo 作为一款老而弥新的服务治理框架，无论是其自身还是其生态都还在飞速进化中。本文描述的最新实现的可信服务调用，是为了避免敏感接口被匿名用户调用而在 SDK 层面提供的额外保障，在 RPC 层面保障安全性。

dubbo-go 作为 Dubbo 生态中发展最快的成员，目前基本上保持与 Dubbo 齐头并进的态势。dubbo-go 社区钉钉群号为 23331795， 欢迎你的加入。

- 作者信息：

  郑泽超，Apache Dubbo/Dubbo-Go committer，GithubID: CodingSinger，目前就职于上海爱奇艺科技有限公司，Java/Golang 开发工程师。
