# dubbogo-cli 工具

## 1. 安装

dubbogo-cli 是 Apach/dubbo-go 生态的子项目，为开发者提供便利的应用模板创建、工具安装、接口调试等功能，以提高用户的研发效率。

执行以下指令安装dubbogo-cli 至 $GOPATH/bin

```
go install dubbo.apache.org/dubbo-go/v3/dubbogo-cli@latest
```

## 2. 功能概览

dubbogo-cli 支持以下能力

- 应用模板创建

  ```
  dubbogo-cli newApp .
  ```

  在当前目录下创建应用模板

- Demo 创建

  ```
  dubbogo-cli newDemo .
  ```

  在当前目录下创建 RPC 示例，包含一个客户端和一个服务端

- 编译、调试工具安装

  ```
  dubbogo-cli install all
  ```

  一键安装以下等工具至 $GOPATH/bin

  - protoc-gen-go-triple

    用于 triple 协议接口编译

  - imports-formatter

    用于整理代码 import 块。

    [import-formatte README](https://github.com/dubbogo/tools#imports-formatter)



- 查看 dubbo-go 应用注册信息

  - 查看 Zookeeper 上面的注册信息, 获取接口及方法列表

    ```bash
    $ dubbogo-cli show --r zookeeper --h 127.0.0.1:2181
    interface: com.dubbogo.pixiu.UserService
    methods: [CreateUser,GetUserByCode,GetUserByName,GetUserByNameAndAge,GetUserTimeout,UpdateUser,UpdateUserByName]
    ```

  - 查看 Nacos 上面的注册信息 【功能开发中】

  - 查看 Istio 的注册信息【功能开发中】

- 调试 Dubbo 协议接口

- 调试 Triple 协议接口

## 3. 功能详解

### 3.1 Demo 应用介绍

#### 3.1.1 Demo 创建

```
dubbogo-cli newDemo .
```

在当前目录下创建Demo, 包含客户端和服务端，该 Demo 展示了基于一套接口，完成一次 RPC 调用。

该Demo 使用直连模式，无需依赖注册中心，server端暴露服务到本地20000端口，客户端发起调用。

```shell
.
├── api
│   ├── samples_api.pb.go
│   ├── samples_api.proto
│   └── samples_api_triple.pb.go
├── go-client
│   ├── cmd
│   │   └── client.go
│   └── conf
│       └── dubbogo.yaml
├── go-server
│   ├── cmd
│   │   └── server.go
│   └── conf
│       └── dubbogo.yaml
└── go.mod
```

#### 3.1.2 运行Demo

开启服务端

```
$ cd go-server/cmd
$ go run .
```

另一个终端开启客户端

```
$ go mod tidy
$ cd go-client/cmd
$ go run .

```

可看到打印日志

```
INFO    cmd/client.go:49        client response result: name:"Hello laurence" id:"12345" age:21
```

### 3.2 应用模板介绍

#### 3.2.1 应用模板创建

```
dubbogo-cli newApp .
```

在当前目录下创建应用模板:

```
.
├── Makefile
├── api
│   ├── api.pb.go
│   ├── api.proto
│   └── api_triple.pb.go
├── build
│   └── Dockerfile
├── chart
│   ├── app
│   │   ├── Chart.yaml
│   │   ├── templates
│   │   │   ├── _helpers.tpl
│   │   │   ├── deployment.yaml
│   │   │   ├── service.yaml
│   │   │   └── serviceaccount.yaml
│   │   └── values.yaml
│   └── nacos_env
│       ├── Chart.yaml
│       ├── templates
│       │   ├── _helpers.tpl
│       │   ├── deployment.yaml
│       │   └── service.yaml
│       └── values.yaml
├── cmd
│   └── app.go
├── conf
│   └── dubbogo.yaml
├── go.mod
├── go.sum
└── pkg
    └── service
        └── service.go

```

#### 3.2.2 应用模板介绍

生成项目包括几个目录：

- api：放置接口文件：proto文件和生成的.pb.go文件
- build：放置镜像构建相关文件
- chart：放置发布用 chart 仓库、基础环境chart 仓库：nacos、mesh（开发中）
- cmd：程序入口
- conf：框架配置
- pkg/service：RPC 服务实现
- Makefile：

- - 镜像、helm部署名：

- - - IMAGE = $(your_repo)/$(namespace)/$(image_name)
      TAG = 1.0.0
    - HELM_INSTALL_NAME = dubbo-go-app，helm 安装名，用于 helm install/uninstall 命令。

- - 提供脚本，例如：

- - - make build # 打包镜像并推送
    - make buildx-publish # arm架构本地打包amd64镜像并推送，依赖 docker buildx
    - make deploy  # 通过 helm 发布应用
    - make remove  # 删除已经发布的 helm 应用
    - make proto-gen # api下生成 pb.go 文件

  -

使用应用模板的开发流程

> 依赖环境：make、go、helm、kubectl、docker

1. 通过 dubbogo-cli 生成模板
2. 修改api/api.proto
3. make proto-gen
4. 开发接口
5. 修改 makefile 内 IMAGE 镜像名和 HELM_INSTALL_NAME 发布名
6. 打镜像并推送
7. 修改chart/app/values 内与部署相关的value配置, 重点关注镜像部分。

```
image:
  repository: $(your_repo)/$(namespace)/$(image_name)
  pullPolicy: Always
  tag: "1.0.0"
```

8. make deploy, 使用 helm 发布应用。

### 3.3 以 gRPC 协议调试 dubbo-go 应用

#### 3.3.1 简介

grpc_cli 工具是 gRPC 生态用于调试服务的工具，在 server 开启[反射服务](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md)的前提下，可以获取到服务的 proto 文件、服务名、方法名、参数列表，以及发起 gRPC 调用。

Triple 协议兼容 gRPC 生态，并默认开启 gRPC 反射服务，因此可以直接使用 grpc_cli 调试 triple 服务。

#### 3.3.2 安装grpc_cli

> 后续将由 dubbogo-cli 安装，目前需要用户手动安装

参考[grpc_cli 官方文档](https://github.com/grpc/grpc/blob/master/doc/command_line_tool.md)

#### 3.3.3 使用 grpc_cli 对 Triple 服务进行调试

1. 查看 triple 服务的接口定义

```shell
$ grpc_cli ls localhost:20001 -l
filename: helloworld.proto
package: org.apache.dubbo.quickstart.samples;
service UserProvider {
  rpc SayHello(org.apache.dubbo.quickstart.samples.HelloRequest) returns (org.apache.dubbo.quickstart.samples.User) {}
  rpc SayHelloStream(stream org.apache.dubbo.quickstart.samples.HelloRequest) returns (stream org.apache.dubbo.quickstart.samples.User) {}
}
```

2. 查看请求参数类型

例如开发者期望测试上述端口的 SayHello 方法，尝试获取HelloRequest的具体定义，需要执行r如下指令，可查看到对应参数的定义。

```shell
$ grpc_cli type localhost:20001 org.apache.dubbo.quickstart.samples.HelloRequest
message HelloRequest {
  string name = 1 [json_name = "name"];
}
```

3. 请求接口

已经知道了请求参数的具体类型，可以发起调用来测试对应服务。查看返回值是否符合预期。

```shell
$ grpc_cli call localhost:20001 SayHello "name: 'laurence'"
connecting to localhost:20001
name: "Hello laurence"
id: "12345"
age: 21
Received trailing metadata from server:
accept-encoding : identity,gzip
adaptive-service.inflight : 0
adaptive-service.remaining : 50
grpc-accept-encoding : identity,deflate,gzip
Rpc succeeded with OK status
```

### 3.4 以 Dubbo 协议调试dubbo-go 应用

#### 3.4.1 开启 Dubbo 服务端

示例：user.go:

```
func (u *UserProvider) GetUser(ctx context.Context, userStruct *CallUserStruct) (*User, error) {
	fmt.Printf("=======================\nreq:%#v\n", userStruct)
	rsp := User{"A002", "Alex Stocks", 18, userStruct.SubInfo}
	fmt.Printf("=======================\nrsp:%#v\n", rsp)
	return &rsp, nil
}
```

服务端开启一个服务，名为GetUser，传入一个CallUserStruct的参数，返回一个User参数
CallUserStruct参数定义：

```
type CallUserStruct struct {
	ID      string
	Male    bool
	SubInfo SubInfo // 嵌套子结构
}
func (cs CallUserStruct) JavaClassName() string {
	return "com.ikurento.user.CallUserStruct"
}

type SubInfo struct {
	SubID   string
	SubMale bool
	SubAge  int
}

func (s SubInfo) JavaClassName() string {
	return "com.ikurento.user.SubInfo"
}
```

User结构定义：

```
type User struct {
	Id      string
	Name    string
	Age     int32
	SubInfo SubInfo // 嵌套上述子结构SubInfo
}

func (u *User) JavaClassName() string {
	return "com.ikurento.user.User"
}
```

开启服务：

```
cd server`
`source builddev.sh`
`go run .
```

#### 3.4.2 定义请求体 (适配于序列化协议)

请求体定义为json文件，约定键值均为string
键对应go语言struct字段名例如"ID"、"Name" ，值对应"type@val"
其中type支持string int bool time，val使用string 来初始化，如果只填写type则初始化为零值。 约定每个struct必须有JavaClassName字段，务必与server端严格对应

见userCall.json:

```
{
  "ID": "string@A000",
  "Male": "bool@true",
  "SubInfo": {
    "SubID": "string@A001",
    "SubMale": "bool@false",
    "SubAge": "int@18",
    "JavaClassName":"string@com.ikurento.user.SubInfo"
  },
  "JavaClassName": "string@com.ikurento.user.CallUserStruct"
}
```

userCall.json将参数CallUserStruct的结构及子结构SubInfo都定义了出来，并且给请求参数赋值。

user.json 同理，作为返回值不需要赋初始值，但JavaClassName字段一定与server端严格对应

```
{
  "ID": "string",
  "Name": "string",
  "Age": "int",
  "JavaClassName":  "string@com.ikurento.user.User",
  "SubInfo": {
    "SubID": "string",
    "SubMale": "bool",
    "SubAge": "int",
    "JavaClassName":"string@com.ikurento.user.SubInfo"
  }
}
```

#### 3.4.3 调试端口

```
./dubbo-go-cli -h=localhost -p=20001 -proto=dubbo -i=com.ikurento.user.UserProvider -method=GetUser -sendObj="./userCall.json" -recvObj="./user.json"
```

打印结果：

```
2020/10/26 20:47:45 Created pkg:
2020/10/26 20:47:45 &{ID:A000 Male:true SubInfo:0xc00006ea20 JavaClassName:com.ikurento.user.CallUserStruct}
2020/10/26 20:47:45 SubInfo:
2020/10/26 20:47:45 &{SubID:A001 SubMale:false SubAge:18 JavaClassName:com.ikurento.user.SubInfo}


2020/10/26 20:47:45 Created pkg:
2020/10/26 20:47:45 &{ID: Name: Age:0 JavaClassName:com.ikurento.user.User SubInfo:0xc00006ec90}
2020/10/26 20:47:45 SubInfo:
2020/10/26 20:47:45 &{SubID: SubMale:false SubAge:0 JavaClassName:com.ikurento.user.SubInfo}


2020/10/26 20:47:45 connected to localhost:20001!
2020/10/26 20:47:45 try calling interface:com.ikurento.user.UserProvider.GetUser
2020/10/26 20:47:45 with protocol:dubbo

2020/10/26 20:47:45 After 3ms , Got Rsp:
2020/10/26 20:47:45 &{ID:A002 Name:Alex Stocks Age:18 JavaClassName: SubInfo:0xc0001241b0}
2020/10/26 20:47:45 SubInfo:
2020/10/26 20:47:45 &{SubID:A001 SubMale:false SubAge:18 JavaClassName:}```
```

可看到详细的请求体赋值情况，以及返回结果和耗时。支持嵌套结构

server端打印结果

```
=======================
req:&main.CallUserStruct{ID:"A000", Male:true, SubInfo:main.SubInfo{SubID:"A001", SubMale:false, SubAge:18}}
=======================
```

可见接收到了来自cli的数据
