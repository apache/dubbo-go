package dubbo3

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/remoting"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"net"
	"reflect"
)

// TripleClient client endpoint for client end
type TripleClient struct {
	conn           net.Conn
	h2Controller   *H2Controller
	exchangeClient *remoting.ExchangeClient
	addr           string
	Invoker        reflect.Value
}

type TripleConn struct {
	client *TripleClient
}

func (t *TripleConn) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	t.client.Request(method, args, reply)
	return nil
}

// todo 封装网络逻辑在这个函数里
func NewTripleConn(client *TripleClient) *TripleConn {
	return &TripleConn{
		client: client,
	}
}

// 返回包含SayHello方法的struct
func getInvoker(impl interface{}, conn *TripleConn) interface{} {
	var in []reflect.Value
	in = append(in, reflect.ValueOf(conn))

	method := reflect.ValueOf(impl).MethodByName("GetDubboStub")
	res := method.Call(in)
	// res[0] 是一个包含了SayHello方法的struct: greeterClient,他的SayHello方法会调用conn的Invoke来具体实现
	// 会给conn传入具体结构体，conn断言直接打包即可
	return res[0].Interface()
}

func NewTripleClient(url *common.URL) *TripleClient {
	key := url.GetParam(constant.BEAN_NAME_KEY, "")
	impl := config.GetConsumerService(key)
	tripleClient := &TripleClient{}
	tripleClient.Connect(url)
	invoker := getInvoker(impl, NewTripleConn(tripleClient)) // 要把网络逻辑放到conn里面
	tripleClient.Invoker = reflect.ValueOf(invoker)
	return tripleClient
}

func (t *TripleClient) Connect(url *common.URL) error {
	logger.Warn("want to connect to url = ", url.Location)
	conn, err := net.Dial("tcp", url.Location)
	if err != nil {
		return err
	}
	t.addr = url.Location
	t.conn = conn
	t.h2Controller, err = NewH2Controller(t.conn, false, nil, nil)
	if err != nil {
		return err
	}
	t.h2Controller.H2ShakeHand()
	return nil
}

func (t *TripleClient) Request(method string, arg, reply interface{}) error {
	reqData, err := proto.Marshal(arg.(proto.Message))
	if err != nil {
		panic("client request marshal not ok ")
	}
	fmt.Printf("getInvocation first arg = %+v\n", reqData)
	t.h2Controller.UnaryInvoke(method, t.conn.RemoteAddr().String(), reqData, reply)
	// todo call remote and recv rsp
	return nil
}

func (t *TripleClient) Close() {

}

func (t *TripleClient) IsAvailable() bool {
	return true
}
