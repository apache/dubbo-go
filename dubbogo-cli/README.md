# dubbogo-cli tool

## 1. Installation

dubbogo-cli is a sub-project of the Apach/dubbo-go ecosystem. It provides developers with convenient application template creation, tool installation, interface debugging and other functions to improve users' R&D efficiency.

Execute the following command to install dubbogo-cli to $GOPATH/bin

```
go install dubbo.apache.org/dubbo-go/v3/dubbogo-cli@latest
```

## 2. Feature Overview

dubbogo-cli supports the following capabilities

- Application Template Creation

  ```
  dubbogo-cli newApp .
  ```

- Create an application template in the current directory

  ```
  dubbogo-cli newDemo .
  ```

  Create an RPC example in the current directory, including a client and a server

- Compile and debug tool installation

  ```
  dubbogo-cli install all
  ```

  One-click install the following tools to $GOPATH/bin

  - protoc-gen-go-triple

    For triple protocol interface compilation

  - imports-formatter

    Used to tidy up code import blocks.

    [import-formatter README](https://github.com/dubbogo/tools#imports-formatter)

- View dubbo-go application registration information

  - View the registration information on Zookeeper to get a list of interfaces and methods

    ```bash
    $ dubbogo-cli show --r zookeeper --h 127.0.0.1:2181
    interface: com.dubbogo.pixiu.UserService
    methods: [CreateUser,GetUserByCode,GetUserByName,GetUserByNameAndAge,GetUserTimeout,UpdateUser,UpdateUserByName]
    ```

  - Check the registration information on Nacos [ in development ]

  - View Istio's registration information [ in development ]

- Debug Dubbo protocol application

- Debug Triple protocol application

## 3. Feature Details

### 3.1 Demo App Introduction

#### 3.1.1 Create Demo

```
dubbogo-cli newDemo .
```

Create a demo in the current directory, including the client and the server. The demo shows a set of interfaces to complete an RPC call.

The Demo uses the direct connection mode, without relying on the registration center, the server side exposes the service to the local port 20000, and the client initiates the call.

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

#### 3.1.2 Run The Demo

Run Server

```
$ cd go-server/cmd
$ go run .
```

Run client in another terminate

```
$ go mod tidy
$ cd go-client/cmd
$ go run .

```

See the logs

```
INFO    cmd/client.go:49        client response result: name:"Hello laurence" id:"12345" age:21
```

### 3.2 Application Template Introduction

#### 3.2.1 Create Application Template

```
dubbogo-cli newApp .
```

Create application template under './'

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

#### 3.2.2 Application Template Introduction

The build project includes several directories:

- api: place interface files: proto file and generated .pb.go file

- build: place mirror build related files

- chart: Place the chart warehouse for release, the basic environment chart warehouse: nacos, mesh (under development)

- cmd: program entry

- conf: dubbogo configuration

- pkg/service: RPC service implementation

- Makefile:

  - Image, helm deployment name:

    - IMAGE = $(your_repo)/$(namespace)/$(image_name)
      TAG = 1.0.0
    - HELM_INSTALL_NAME = dubbo-go-app, helm installation name, for helm install/uninstall commands.

  - Provide make scripts such as:

    - make build # Package the image and push it

    - make buildx-publish # The arm architecture packages the amd64 image locally and pushes it, depending on docker buildx

    - make deploy # Publish the application via helm

    - make remove # remove the published helm application

    - make proto-gen # generate pb.go file under api


Development Process Using App Templates:

> rely env：make、go、helm、kubectl、docker

1. Generate templates through dubbogo-cli
2. Modify api/api.proto
3. make proto-gen
4. Development interface
5. Modify the IMAGE image name and the HELM_INSTALL_NAME release name in the makefile
6. Mirror and push
7. Modify the deployment-related value configuration in chart/app/values, focusing on the mirroring part.

```
image:
  repository: $(your_repo)/$(namespace)/$(image_name)
  pullPolicy: Always
  tag: "1.0.0"
```

8. run make deploy, to deploy app by helm.

### 3.3 Debug dubbo-go application with gRPC/Triple protocol

#### 3.3.1 Introduction

The grpc_cli tool is a tool used by the gRPC ecosystem to debug services. On the premise that the [reflection service] (https://github.com/grpc/grpc/blob/master/doc/server-reflection.md) is enabled on the server, you can obtain To the proto file of the service, the service name, the method name, the parameter list, and to initiate a gRPC call.

The Triple protocol is compatible with the gRPC ecosystem, and the gRPC reflection service is enabled by default, so you can directly use grpc_cli to debug triple services.

#### 3.3.2 Install grpc_cli

> It will be installed by dubbogo-cli in the future, currently it needs to be installed manually by the user

Refer to [grpc_cli official documentation](https://github.com/grpc/grpc/blob/master/doc/command_line_tool.md)

#### 3.3.3 Debugging Triple service with grpc_cli

1. View the interface definition of triple service

```shell
$ grpc_cli ls localhost:20001 -l
filename: helloworld.proto
package: org.apache.dubbo.quickstart.samples;
service UserProvider {
  rpc SayHello(org.apache.dubbo.quickstart.samples.HelloRequest) returns (org.apache.dubbo.quickstart.samples.User) {}
  rpc SayHelloStream(stream org.apache.dubbo.quickstart.samples.HelloRequest) returns (stream org.apache.dubbo.quickstart.samples.User) {}
}
```

2. View the request parameter type

For example, if a developer wishes to test the SayHello method of the above port and tries to obtain the specific definition of HelloRequest, he needs to execute the following command to view the definition of the corresponding parameter.

```shell
$ grpc_cli type localhost:20001 org.apache.dubbo.quickstart.samples.HelloRequest
message HelloRequest {
  string name = 1 [json_name = "name"];
}
````

3. Call provider

Now that you know the specific types of request parameters, you can initiate a call to test the corresponding service. Check if the return value is as expected

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

### 3.4 Debug dubbo-go application with Dubbo protocol

#### 3.4.1 Open Dubbo server

Example: user.go:

````
func (u *UserProvider) GetUser(ctx context.Context, userStruct *CallUserStruct) (*User, error) {
fmt.Printf("========================\nreq:%#v\n", userStruct)
rsp := User{"A002", "Alex Stocks", 18, userStruct.SubInfo}
fmt.Printf("========================\nrsp:%#v\n", rsp)
return &rsp, nil
}
````

The server starts a service named GetUser, passes in a CallUserStruct parameter, and returns a User parameter
CallUserStruct parameter definition:

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

User POJO definition：

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

Run server：

```
cd server
source builddev.sh
go run .
```

#### 3.4.2 Define the request body (adapted to the serialization protocol)

The request body is defined as a json file, and the agreed key value is string
The key corresponds to the go language struct field name such as "ID", "Name", and the value corresponds to "type@val"
Among them, type supports string int bool time, val is initialized with string, and if only type is filled in, it will be initialized with zero value. It is agreed that each struct must have a JavaClassName field, which must correspond strictly to the server side

See userCall.json:

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

userCall.json defines the structure and substructure SubInfo of the parameter CallUserStruct, and assigns values to the request parameters.

Similarly, user.json does not need to be assigned an initial value as the return value, but the JavaClassName field must strictly correspond to the server side

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

#### 3.4.3 Debug server

```
./dubbo-go-cli -h=localhost -p=20001 -proto=dubbo -i=com.ikurento.user.UserProvider -method=GetUser -sendObj="./userCall.json" -recvObj="./user.json"
```

Print result：

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

You can see the detailed assignment of the request body, as well as the return results and time-consuming. Nested structures are supported

server side print result

````
=========================
req:&main.CallUserStruct{ID:"A000", Male:true, SubInfo:main.SubInfo{SubID:"A001", SubMale:false, SubAge:18}}
=========================
````

It can be seen that the data from the cli has been received
