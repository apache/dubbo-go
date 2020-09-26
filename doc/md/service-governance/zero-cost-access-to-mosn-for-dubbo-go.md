# [Dubbo/Dubbo-go 应用零成本接入 MOSN](https://mosn.io/docs/dev/dubbo-integrate/)

## Dubbo 介绍

Dubbo 最初是 **Java 开发的一套 RPC 框架**，随着社区的发展。当前 dubbo 也渐渐成为一套跨语言的解决方案。**除了 Java 以外，还有相应的 Go 实现**。有规律的版本发布节奏，社区较为活跃。

## Dubbo 服务 mesh 化

接入 service mesh 的应用，其服务发现应该由相应的 mesh 模块接管。一般由控制面将相应的服务发现配置进行订阅和下发。但这里存在几个问题：

如果公司是第一次接入 service mesh，不希望一次引入太多模块，这样会增加整体的运维负担。如果可以渐进地迁移到 service mesh 架构，例如先接入数据面，再接入控制面。那么就可以随时以较低的成本进行回滚。也不会给运维造成太大的压力。

每个公司都有自己的发展规划，并不是每个公司都完整地拥抱了云原生。大部分公司可能存在部分上云，部分未上云的情况，在迁移到 service mesh 时，也存在部分应用接入了 service mesh，而另一部分未接入的情况。需要考虑跨架构互通。

我们这里提出的方案希望能够解决这些问题。

### 服务发现接入

#### 配置工作

在配置文件中，我们配置了两个 listener：

一个是 serverListener，负责拦截外部进入的流量，转发给本地模块，这个方向的请求不需要做特殊处理，只要使用 xprotocol 转发给本机即可。

一个是 clientListener，负责拦截本机向外发起的请求，因为外部集群根据服务注册中心下发的 endpoint 列表动态变化，所以该 listener 对应的也是一个 特殊的 router 名 “dubbo”。，这里务必注意。

```json
    "listeners": [
        {
            "name": "serverListener",
            "address": "127.0.0.1:2046",
            "bind_port": true,
            "log_path": "stdout",
            "filter_chains": [
                {
                    "tls_context": {},
                    "filters": [
                        {
                            "type": "proxy",
                            "config": {
                                "downstream_protocol": "X",
                                "upstream_protocol": "X",
                                "router_config_name": "server_router",
                                "extend_config": {
                                    "sub_protocol": "dubbo"
                                }
                            }
                        }
                    ]
                }
            ]
        },
        {
            "name": "clientListener",
            "address": "0.0.0.0:2045",
            "bind_port": true,
            "log_path": "stdout",
            "filter_chains": [
                {
                    "tls_context": {},
                    "filters": [
                        {
                            "type": "proxy",
                            "config": {
                                "downstream_protocol": "X",
                                "upstream_protocol": "X",
                                "router_config_name": "dubbo",
                                "extend_config": {
                                    "sub_protocol": "dubbo"
                                }
                            }
                        }
                    ]
                }
            ]
        }
    ]
```

#### 开发工作

第一步，在 MOSN 配置中增加 dubbo_registry 扩展选项：

```json
"extend": [
    {
        "type": "dubbo_registry",
        "config": {
            "enable": true,
            "server_port": 20080,
            "api_port": 22222,
            "log_path": "/tmp"
        }
    }
]
```

该配置与 tracing、admin 等为平级配置。

第二步，针对接入的服务，需要简单修改 sdk 中的 pub、sub 环节代码：

pub 时，如果当前环境为接入 MOSN 环境(可通过配置系统下发的开关来判断)，则调用 MOSN 的 pub 接口，而非直接去注册中心 pub。

sub 时，如果当前环境为接入 MOSN 环境，则调用 MOSN 的 sub 接口，不去注册中心 sub。

第三步，应用退出时，需要将所有 pub、sub 的服务执行反向操作，即 unpub、unsub。

在本文中使用 httpie 来发送 http 请求。使用 dubbo-go 中的样例程序作为我们的服务的 client 和 server。

接下来我们使用 httpie 来模拟各种情况下的 pub、sub 流程。

直连 client 与正常的 dubbo service 互通
例子路径

Service 是正常的 dubbo service，所以会自动注册到 zk 中去，不需要我们帮它 pub，这里只要 sub 就可以了，所以执行流程为：

第一步，修改 MOSN 配置，增加 dubbo_registry 的 extend 扩展。

第二步，mosn start。

第三步，start server。

第四步，subscribe service。
```sh
http --json post localhost:22222/sub registry:='{"type":"zookeeper", "addr" : "127.0.0.1:2181"}' service:='{"interface" : "com.ikurento.user.UserProvider", "methods" :["GetUser"], "group" : "", "version" : ""}' --verbose
```
第五步，start client。

在 client 中正确看到返回结果的话，说明请求成功了。

直连 client 与直连 dubbo service 互通
例子路径

直连的服务不会主动对自身进行发布，直连的 client 不会主动进行订阅。因此此例子中，pub 和 sub 都是由我们来辅助进行的。

第一步，修改 MOSN 配置，增加 dubbo_registry 的 extend 扩展。

第二步，mosn start

第三步，start server

第四步，subscribe service
```sh
http --json post localhost:22222/sub registry:='{"type":"zookeeper", "addr" : "127.0.0.1:2181"}' service:='{"interface" : "com.ikurento.user.UserProvider", "methods" :["GetUser"], "group" : "", "version" : ""}' --verbose
```
第五步，publish service

http --json post localhost:22222/pub registry:='{"type":"zookeeper", "addr" : "127.0.0.1:2181"}' service:='{"interface" : "com.ikurento.user.UserProvider", "methods" :["GetUser"], "group" : "", "version" : ""}' --verbose
第六步，start client

此时应该能看到 client 侧的响应。

正常的 client 与直连 dubbo service 互通
例子路径

Client 是正常 client，因此 client 会自己去 subscribe。我们只要正常地把服务 pub 出去即可：

第一步，修改 MOSN 配置，增加 dubbo_registry 的 extend 扩展。

第二步，mosn start

第三步，start server

第四步，publish service
```sh
http --json post localhost:22222/sub registry:='{"type":"zookeeper", "addr" : "127.0.0.1:2181"}' service:='{"interface" : "com.ikurento.user.UserProvider", "methods" :["GetUser"], "group" : "", "version" : ""}' --verbose
```
第五步，start client

此时应该能看到 client 侧的响应。
