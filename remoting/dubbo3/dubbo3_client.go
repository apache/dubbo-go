package dubbo3

import (
	"context"
	"net"
	"reflect"
)

import (
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/remoting"
)

// TripleClient client endpoint for client end
type TripleClient struct {
	conn           net.Conn
	h2Controller   *H2Controller
	exchangeClient *remoting.ExchangeClient
	addr           string
	Invoker        reflect.Value
	url            *common.URL
}

type TripleConn struct {
	client *TripleClient
}

func (t *TripleConn) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	t.client.Request(ctx, method, args, reply)
	return nil
}

func (t *TripleConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	// todo use dubbo3 network
	return nil, nil
}

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

func NewTripleClient(url *common.URL) (*TripleClient, error) {
	key := url.GetParam(constant.BEAN_NAME_KEY, "")
	impl := config.GetConsumerService(key)
	tripleClient := &TripleClient{
		url: url,
	}
	if err := tripleClient.Connect(url); err != nil {
		return nil, err
	}
	invoker := getInvoker(impl, NewTripleConn(tripleClient)) // 要把网络逻辑放到conn里面
	tripleClient.Invoker = reflect.ValueOf(invoker)
	return tripleClient, nil
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
	return t.h2Controller.H2ShakeHand()
}

func (t *TripleClient) Request(ctx context.Context, method string, arg, reply interface{}) error {
	reqData, err := proto.Marshal(arg.(proto.Message))
	if err != nil {
		panic("client request marshal not ok ")
	}
	//fmt.Printf("getInvocation first arg = %+v\n", reqData)
	if err := t.h2Controller.UnaryInvoke(ctx, method, t.conn.RemoteAddr().String(), reqData, reply, t.url); err != nil {
		return err
	}
	return nil
}

func (t *TripleClient) Close() {

}

func (t *TripleClient) IsAvailable() bool {
	return true
}
