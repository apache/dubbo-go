

## dubbogo examples ##
---
    a golang apache dubbo code example. just using jsonrpc 2.0 protocol over http now.

examples是基于dubbogo的实现的代码示例，目前提供user-info一个例子。examples/client 是一个 go client，借鉴java的编译思路，提供了区别于一般的go程序的而类似于java的独特的编译脚本系统。

###  example1: user-info ###
---
*从这个程序可以看出dubbogo client程序(user-info/client) 如何调用 java(dubbo) server的程序( user-info/java-server)的。*

**Attention**: 测试的时候一定注意修改配置文件中服务端和zk的ip&port。

+ 1 部署zookeeper服务；
+ 2 请编译并部署 examples/java-server，注意修改zk地址(conf/dubbo.properties:line6:"dubbo.registry.address")和监听端口(conf/dubbo.properties:line6:"dubbo.protocol.port", 不建议修改port), 然后执行"bin/start.sh"启动java服务端；
+ 3 修改 examples/client/profiles/test/client.toml:line 31，写入正确的zk地址；
+ 4 examples/client/ 下执行 sh assembly/mac/dev.sh命令(linux下请执行sh assembly/linux/dev.sh)

	target/darwin 下即放置好了编译好的程序以及打包结果，在 client/target/darwin/user_info_client-0.2.0-20180808-1258-dev 下执行sh bin/load.sh start命令即可客户端程序；

