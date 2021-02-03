# dubbo-go-cli example

### 1. Start dubbo-go server
before we use dubbo-go-cli to send a request, we start a dubbo-go server firstly.

example in file: server/main.go server/user.go

example：user.go:
```go
func (u *UserProvider) GetUser(ctx context.Context, userStruct *CallUserStruct) (*User, error) {
	fmt.Printf("=======================\nreq:%#v\n", userStruct)
	rsp := User{"A002", "Alex Stocks", 18, userStruct.SubInfo}
	fmt.Printf("=======================\nrsp:%#v\n", rsp)
	return &rsp, nil
}

```
as example shows above, server start a function named GetUser, which has a param @CallUserStruct and return User struct.

param @CallUserStruct defination:
```go
type CallUserStruct struct {
	ID      string
	Male    bool
	SubInfo SubInfo // nesting sub struct
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
User struct defination：
```go
type User struct {
	Id      string
	Name    string
	Age     int32
	SubInfo SubInfo // nesting sub struct, the same as above
}

func (u *User) JavaClassName() string {
	return "com.ikurento.user.User"
}
```

start dubbo-go server:

`$ cd server `\
`$ source builddev.sh`\
`$ go run .`

### 2. Define your request structs (encode and decode protocol)
You should define your request structs in json format. we appoint that key and val in json file must be string.\
Key in json file defines your go struct's field name, such as "ID","Name".\
Value in json file defines your go struct's field type and value, in format of "type@value". We support 'type' of 'string,int,bool,time', and use value to init the field, if value is empty, we init it by zero.
We appoint that each json struct must have key 'JavaClassName', and the value of it must corresponding to server end.  

example int userCall.json:
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
'userCall.json' defines param @CallUserStruct and it's substruct 'SubInfo', meanwhile it inits all fields.


'user.json' Similarly defines all field, but you are not need to set inital value for them. Remember that 'JavaClassName' field must corresponding to server end.
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

### 3. Exec cli to send req.
`./dubbo-go-cli -h=localhost -p=20001 -proto=dubbo -i=com.ikurento.user.UserProvider -method=GetUser -sendObj="./userCall.json" -recvObj="./user.json"`

cli-end output：
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
By reading logs above, you can get req struct, cost time and rsp struct in details.\
The nesting sub struct is supported.

server-end output:
```
=======================
req:&main.CallUserStruct{ID:"A000", Male:true, SubInfo:main.SubInfo{SubID:"A001", SubMale:false, SubAge:18}}
=======================
```
It's showed that server-end got specific request.