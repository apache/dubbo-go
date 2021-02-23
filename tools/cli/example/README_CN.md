# dubbo-go-cli 使用示例

### 1. 开启服务端
见 server/main.go server/user.go

示例：user.go:
```go
func (u *UserProvider) GetUser(ctx context.Context, userStruct *CallUserStruct) (*User, error) {
	fmt.Printf("=======================\nreq:%#v\n", userStruct)
	rsp := User{"A002", "Alex Stocks", 18, userStruct.SubInfo}
	fmt.Printf("=======================\nrsp:%#v\n", rsp)
	return &rsp, nil
}

```
服务端开启一个服务，名为GetUser，传入一个CallUserStruct的参数，返回一个User参数\
CallUserStruct参数定义：
```go
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
```go
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

`cd server `\
`source builddev.sh`\
`go run .`

### 2. 定义请求体(打解包协议)

请求体定义为json文件，约定键值均为string\
键对应go语言struct字段名例如"ID"、"Name" ，值对应"type@val"\
其中type支持string int bool time，val使用string 来初始化，如果只填写type则初始化为零值。
约定每个struct必须有JavaClassName字段，务必与server端严格对应

见userCall.json:
```json
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
```go
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

### 3. 执行请求
`./dubbo-go-cli -h=localhost -p=20001 -proto=dubbo -i=com.ikurento.user.UserProvider -method=GetUser -sendObj="./userCall.json" -recvObj="./user.json"`

cli端打印结果：
```log
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