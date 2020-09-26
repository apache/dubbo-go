# [快速开始](https://dubbogo.github.io/dubbo-go-website/zh-cn/docs/user/quick-start.html)

通过一个 `hellowworld` 例子带领大家快速上手Dubbo-go框架。

协议：Dubbo  
编码：Hessian2  
注册中心：Zookeeper

## 环境

*   Go编程环境
*   启动zookeeper服务，也可以使用远程实例

## 从服务端开始

### 第一步：编写 `Provider` 结构体和提供服务的方法

> [https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-server/app/user.go](https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-server/app/user.go)

1.  编写需要被编码的结构体，由于使用 `Hessian2` 作为编码协议，`User` 需要实现 `JavaClassName` 方法，它的返回值在dubbo中对应User类的类名。

```go
type User struct {
	Id   string
	Name string
	Age  int32
	Time time.Time
}

func (u User) JavaClassName() string {
	return "com.ikurento.user.User"
}
```

2.  编写业务逻辑，`UserProvider` 相当于dubbo中的一个服务实现。需要实现 `Reference` 方法，返回值是这个服务的唯一标识，对应dubbo的 `beans` 和 `path` 字段。

```go
type UserProvider struct {
}

func (u *UserProvider) GetUser(ctx context.Context, req []interface{}) (*User, error) {
	println("req:%#v", req)
	rsp := User{"A001", "hellowworld", 18, time.Now()}
	println("rsp:%#v", rsp)
	return &rsp, nil
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}
```

3.  注册服务和对象

```go
func init() {
	config.SetProviderService(new(UserProvider))
	// ------for hessian2------
	hessian.RegisterPOJO(&User{})
}
```

### 第二步：编写主程序

> [https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-server/app/server.go](https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-server/app/server.go)

1.  引入必需的dubbo-go包

```go
import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go/config"
	_ "github.com/apache/dubbo-go/registry/protocol"
	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	_ "github.com/apache/dubbo-go/filter/impl"
	_ "github.com/apache/dubbo-go/cluster/cluster_impl"
	_ "github.com/apache/dubbo-go/cluster/loadbalance"
	_ "github.com/apache/dubbo-go/registry/zookeeper"

	_ "github.com/apache/dubbo-go/protocol/dubbo"
)
```

2.  main 函数

```go
func main() {
	config.Load()
}
```

### 第三步：编写配置文件并配置环境变量

1.  参考 [log](https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-server/profiles/release/log.yml) 和 [server](https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-server/profiles/release/server.yml) 编辑配置文件。

主要编辑以下部分：

*   `registries` 结点下需要配置zk的数量和地址
    
*   `services` 结点下配置服务的具体信息，需要配置 `interface` 配置，修改为对应服务的接口名，服务的key对应第一步中 `Provider` 的 `Reference` 返回值
    

2.  把上面的两个配置文件分别配置为环境变量

```shell
export CONF_PROVIDER_FILE_PATH="xxx"
export APP_LOG_CONF_FILE="xxx"
```

## 接着是客户端

### 第一步：编写客户端 `Provider`

> [https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-client/app/user.go](https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-client/app/user.go)

1.  参考服务端第一步的第一点。
    
2.  与服务端不同的是，提供服务的方法作为结构体的参数，不需要编写具体业务逻辑。另外，`Provider` 不对应dubbo中的接口，而是对应一个实现。
    

```go
type UserProvider struct {
	GetUser func(ctx context.Context, req []interface{}, rsp *User) error
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}
```

3.  注册服务和对象

```go
func init() {
	config.SetConsumerService(userProvider)
	hessian.RegisterPOJO(&User{})
}
```

### 第二步：编写客户端主程序

> [https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-client/app/client.go](https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-client/app/client.go)

1.  引入必需的dubbo-go包

```go
import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go/config"
	_ "github.com/apache/dubbo-go/registry/protocol"
	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	_ "github.com/apache/dubbo-go/filter/impl"
	_ "github.com/apache/dubbo-go/cluster/cluster_impl"
	_ "github.com/apache/dubbo-go/cluster/loadbalance"
	_ "github.com/apache/dubbo-go/registry/zookeeper"

	_ "github.com/apache/dubbo-go/protocol/dubbo"
)
```

2.  main 函数

```go
func main() {
	config.Load()
	time.Sleep(3e9)

	println("\n\n\nstart to test dubbo")
	user := &User{}
	err := userProvider.GetUser(context.TODO(), []interface{}{"A001"}, user)
	if err != nil {
		panic(err)
	}
	println("response result: %v\n", user)
}
func println(format string, args ...interface{}) {
	fmt.Printf("\033[32;40m"+format+"\033[0m\n", args...)
}
```

### 第三步：编写配置文件并配置环境变量

1.  参考 [log](https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-client/profiles/release/log.yml) 和 [client](https://github.com/dubbogo/dubbo-samples/blob/master/golang/helloworld/dubbo/go-client/profiles/release/client.yml) 编辑配置文件。

主要编辑以下部分：

*   `registries` 结点下需要配置zk的数量和地址
    
*   `references` 结点下配置服务的具体信息，需要配置 `interface` 配置，修改为对应服务的接口名，服务的key对应第一步中 `Provider` 的 `Reference` 返回值
    

2.  把上面的两个配置文件费别配置为环境变量，为防止log的环境变量和服务端的log环境变量冲突，建议所有的环境变量不要做全局配置，在当前起效即可。

```shell
export CONF_CONSUMER_FILE_PATH="xxx"
export APP_LOG_CONF_FILE="xxx"
```