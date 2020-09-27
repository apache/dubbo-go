# [快速上手 dubbo-go](https://studygolang.com/articles/29457)

## 前言

每次技术调研总会发现自己学不动了怎么办？用已有的知识来拓展要学习的新知识就好了~ by LinkinStar

最近需要调研使用 dubbo，之前完全是 0 基础，对于 dubbo 只存在于听说，今天上手实战一把，告诉你如何快速用 go 上手 dubbo

PS：以下的学习方式适用于很多新技术

## 基本概念

首先学习一个技术首先要看看它的整体架构和基本概念，每个技术都有着自己的名词解释和实现方式，如果文档齐全就简单很多。

http://dubbo.apache.org/zh-cn/docs/user/preface/background.html

大致浏览了背景、需求、架构之后基本上有一个大致概念

image.png
其实整体架构和很多微服务的架构都是类似的，就是有一个注册中心管理所有的服务列表，服务提供方先向注册中心注册，而消费方向注册中心请求服务列表，通过服务列表调用最终的服务。总的来说 dubbo 将整个过程封装在了里面，而作为使用者的我们来说更加关心业务实现，它帮我们做好了治理的工作。

然后我抓住了几个我想要知道的重点：

注册中心可替换，官方推荐的是 zk
如果有变更，注册中心将基于长连接推送变更数据给消费者，注册中心，服务提供者，服务消费者三者之间均为长连接
基于软负载均衡算法，选一台提供者进行调用，如果调用失败，再选另一台调用
消费者在本地缓存了提供者列表

## 实际上手

官网文档中也给出如果使用 golang 开发，那么有 https://github.com/apache/dubbo-go 可以用，那么废话不多说，上手实践一把先。因为你看再多，都比不上实践一把来学的快。

### 环境搭建

大多数教程就会跟你说，一个 helloWorld 需要 zookeeper 环境，但是不告诉你如何搭建，因为这对于他们来说太简单了，而我不一样，我是 0 基础，那如何快速搭建一个需要调研项目的环境呢？最好的方式就是 docker。

```yml
version: "3"
services:
  zookeeper:
    image: zookeeper
    ports:
      - 2181:2181
  admin:
    image: apache/dubbo-admin
    depends_on:
      - zookeeper
    ports:
      - 8080:8080
    environment:
      - admin.registry.address=zookeeper://zookeeper:2181
      - admin.config-center=zookeeper://zookeeper:2181
      - admin.metadata-report.address=zookeeper://zookeeper:2181
```

```yml
version: "3"
services:
  zookeeper:
    image: zookeeper
    ports:
      - 2181:2181
  admin:
    image: chenchuxin/dubbo-admin
    depends_on:
      - zookeeper
    ports:
      - 8080:8080
    environment:
      - dubbo.registry.address=zookeeper://zookeeper:2181
      - dubbo.admin.root.password=root
      - dubbo.admin.guest.password=guest
```

上面两个 docker-compose 文件一个是官方提供的管理工具，一个包含的是个人修改之后的管理工具，记住这里有个用户名密码是 root-root，看你喜欢

废话不多说，直接创建 docker-compose.yaml 然后 `docker-compose up` 你就得到了一个环境，棒 👍

### 样例下载

有的技术会给出官方的实验样例，dubbo-go 也不例外

https://github.com/apache/dubbo-samples

里面包含了 go 和 java 的样例，有了它你就能快速上手了，把它下载到本地

### 样例运行

首先要做的第一步就是把 helloworld 给跑起来，进入 golang 目录，里面有个 README.md 看一下。然后开搞。

打开一个终端运行服务方
```sh
export ARCH=mac
export ENV=dev
cd helloworld/dubbo/go-server
sh ./assembly/$ARCH/$ENV.sh
cd ./target/linux/user_info_server-0.3.1-20190517-0930-release
sh ./bin/load.sh start
```
打开另一个终端运行客户端

```sh
export ARCH=mac
export ENV=dev
cd helloworld/dubbo/go-client
sh ./assembly/$ARCH/$ENV.sh
cd ./target/linux/user_info_client-0.3.1-20190517-0921-release
sh ./bin/load_user_info_client.sh start
```
启动过程中会出现一些警告，问题不大，如果成功，那么客户端会有一个调用服务端的请求，并在控制台中以白色底色进行打印

image.png
java 的服务也有相对应的启动方式，按照 README 中所说明的也可以进行注册和调用，并且 java 和 go 之间是可以互相调用的

### 查看服务

因为我们部署的时候有一个 dubbo-admin 用于管理 zk 上注册的服务，我们可以访问本地的 8080 端口看到对应的服务情况

image.png
至此你应该已经对于整体的链路调用有一个大致的认识，结合之前官网文档的中的架构图，应该也清晰了。

## 如何使用

那么现在你已经运行起来了，那么势必就要来看看具体是如何进行使用的了。

### 服务端

服务端，也就是服务提供者；

位置在：dubbo-samples/golang/helloworld/dubbo/go-server/app

```go
// 将服务进行注册
config.SetProviderService(new(UserProvider))
// 注册对象的hessian序列化
hessian.RegisterPOJO(&User{})
```

是不是看起来其实很简单，其实重点就是上面两句代码了
对于服务本身来说

```go
type UserProvider struct {
}

func (u *UserProvider) GetUser(ctx context.Context, req []interface{}) (*User, error) {
    println("req:%#v", req)
    rsp := User{"A001", "Alex Stocks", 18, time.Now()}
    println("rsp:%#v", rsp)
    return &rsp, nil
}

func (u *UserProvider) Reference() string {
    return "UserProvider"
}
```

其实就是需要实现下面那个接口就可以了

```go
// rpc service interface
type RPCService interface {
    Reference() string // rpc service id or reference id
}
```

其中有一步骤不要忘记了是 config.Load()，会加载配置文件的相关配置，不然你以为注册中心的地址等等是在哪里配置的呢？

客户端
看了服务端，其实客户端也就很简单了

```go
config.SetConsumerService(userProvider)
hessian.RegisterPOJO(&User{})
```

其实也是差不多的，也需要注册一个消费服务，并将序列化方式给注册上去

```go
user := &User{}
err := userProvider.GetUser(context.TODO(), []interface{}{"A001"}, user)
```

使用也就很简单了，同样的也需要实现 Reference 接口

```go
type UserProvider struct {
    GetUser func(ctx context.Context, req []interface{}, rsp *User) error
}

func (u *UserProvider) Reference() string {
    return "UserProvider"
}
```

回头想想
当我们看完了代码的上的使用，再回头想想 dubbo 的作用，你就会发现其实 dubbo 帮你做的事情真的屏蔽了很多。

- 你不需要关心服务是怎么注册的
- 你不需要关心服务是怎么获取的
- 你不需要关系服务是怎么调用的
- 甚至序列化的过程都是基本透明的
- 想对比来说，如果让你自己去是实现微服务，那是不是说，你需要实现服务的拉取，服务的负载均衡，服务的发现，序列化.....

这下你应该明白了 dubbo 是做什么的也就明白了它为什么会出现了

## 其他能力

当你学习了一个技术的基本使用之后，要学习其他的能力，以便在使用的过程中不会出现自己重复造轮子或者说有轮子没用到的情况，https://github.com/apache/dubbo-samples 不止有 helloworld 还要很多别的案例供你参考，你可以继续看看并进行试验。

### 支持扩展

https://github.com/apache/dubbo-go

在 Feature list 中列举了 dubbo-go 所支持的相关功能，比如序列化，比如注册中心，在比如过滤器。

也就是说，在使用 dubbo-go 的时候相关功能都是插件化的，想用什么就看你自己了，比如注册中心我想用 etcd，比如调用的协议我想换成 grpc 都可以。

## 相关问题

对于一个技术调研，那么肯定会有相关问题等着你去发现去考虑。

下面是我能想到的一些问题：

当前 dubbo-go 的版本最高在 1.4，所对应的 dubbo 版本应该是 2.6.x，如果调用更高版本的服务是否会有问题
java 和 go 之间互相调用，各种类型转换之间是否存在问题，是否容易出现无法正确反序列化的问题


## 后续学习

那么上面只是说能让你上手 dubbo-go，但是实际使用可能存在距离。为什么这么说呢？如果你在不动里面任何的原理情况下，那么如果遇到问题，你很可能就束手无策了。比如如果线上服务无法正常发现，你应该如何排查？调用过程中出现问题如何定位？

所以后续你需要做的是：

看看整体设计架构和思路，明白整条链路调用过程和规则
学习它的接口设计，为什么别人设计的接口能兼容那么多的注册中心？如果让你来设计你怎么设计呢？
性能也很重要，网上说性能很不错，为什么呢？什么地方做了优化，负载均衡的算法是怎么样的，你能自定义吗？

## 总结

总的来说，对于 dubbo-go 整体的使用上还是非常好上手的，自己想了一下，如果当前项目想要接入的话，主要是服务的暴露、序列化方式、鉴权调整等存在开发工作。

上面砖头也抛的差不多了，对于你快速上手应该没有问题了，剩下的就要靠你自己了。
