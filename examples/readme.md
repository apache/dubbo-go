## dubbogo examples
---
    ***a golang apache dubbo code example. just using jsonrpc 2.0 protocol over http now.***

### Introduction

examples是基于dubbogo的实现的代码示例，目前提供user-info一个例子。examples/client 是一个 go client，借鉴java的编译思路，提供了区别于一般的go程序的而类似于java的独特的编译脚本系统。

### Code Example

#### 1: user-info 

*从这个程序可以看出dubbogo client程序(user-info/client) 如何调用 java(dubbo) server的程序( user-info/java-server)的。*

**Attention**: 测试的时候一定注意修改配置文件中服务端和zk的ip&port。

+ 1 部署zookeeper服务；
+ 2 请编译并部署 examples/java-server，注意修改zk地址(conf/dubbo.properties:line6:"dubbo.registry.address")和监听端口(conf/dubbo.properties:line6:"dubbo.protocol.port", 不建议修改port), 然后执行"bin/start.sh"启动java服务端；
+ 3 修改 examples/client/profiles/test/client.toml:line 31，写入正确的zk地址；
+ 4 examples/client/ 下执行 sh assembly/mac/dev.sh命令(linux下请执行sh assembly/linux/dev.sh)

	target/darwin 下即放置好了编译好的程序以及打包结果，在 client/target/darwin/user_info_client-0.2.0-20180808-1258-dev 下执行sh bin/load.sh start命令即可客户端程序；

### QA List

#### 1 Unsupported protocol jsonrpc

问题描述: 用java写的dubbo provider端程序如果遇到下面的错误：

	 java.lang.IllegalStateException: Unsupported protocol jsonrpc in notified url:   
	
	 jsonrpc://116.211.15.189:8080/im.ikurento.user.UserProvider?anyhost=true&application=token-dubbo-p&default.threads=100&dubbo=2.5.3…
	
	 from registry 116.211.15.190:2181 to consumer 10.241.19.54, supported protocol: [dubbo, http, injvm, mock, redis, registry, rmi, thrift]

错误原因：provider端声明使用了jsonrpc，所以所有的协议都默认支持了jsonrpc协议。

解决方法：服务需要在dubbo.provider.xml中明确支持dubbo协议，请在reference中添加protocol="dubbo"，如：

    <dubbo:protocol name="dubbo" port="28881" />
    <dubbo:protocol name="jsonrpc" port="38881" server="jetty" />
    <dubbo:service id="userService" interface="im.ikurento.user.UserProvider" check="false" timeout="5000" protocol="dubbo"/>

 	<dubbo:service id="userService" interface="im.ikurento.user.UserProvider" check="false" timeout="5000" protocol="jsonrpc"/>

与本问题无关，补充一些消费端的配置：

    <dubbo:reference id="userService" interface="im.ikurento.user.UserService" connections="2" check="false">
    	<dubbo:method name="GetUser" async="true" return="false" />
 	</dubbo:reference>

#### 2 配置文件

dubbogo client端根据其配置文件client.toml的serviceList来watch zookeeper上相关服务，否则当相关服务的zk provider path下的node发生增删的时候，因为关注的service不正确而导致不能收到相应的通知。

所以务必注意这个配置文件中的serviceList与实际代码中相关类的Service函数提供的Service一致。

#### 3 其他注意事项
- a. dubbo 可以配置多个 zk 地址；
- b. 消费者在 dubbo 的配置文件中相关interface配置务必指定protocol, 如protocol="dubbo";
- c. java dubbo provider提供服务的method如果要提供给dubbogo client服务，则method的首字母必须大写;