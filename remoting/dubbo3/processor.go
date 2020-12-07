package dubbo3

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

const defaultRWBufferMaxLen = 4096

type processor struct {
	stream                *stream
	codec                 CodeC
	readWriteMaxBufferLen uint32 // useless
}

// Dubbo3GrpcService is gRPC service
type Dubbo3GrpcService interface {
	// SetProxyImpl sets proxy.
	SetProxyImpl(impl protocol.Invoker)
	// GetProxyImpl gets proxy.
	GetProxyImpl() protocol.Invoker
	// ServiceDesc gets an RPC service's specification.
	ServiceDesc() *grpc.ServiceDesc
}

// protoc config参数增加,对codec进行选择
func newProcessor(s *stream) *processor {
	return &processor{
		stream:                s,
		codec:                 NewProtobufCodeC(),
		readWriteMaxBufferLen: defaultRWBufferMaxLen,
	}
}

func (p *processor) processUnaryRPC(buf bytes.Buffer, method string, service common.RPCService, url *common.URL) (*bytes.Buffer, error) {
	readBuf := buf.Bytes()
	header := readBuf[:5]
	length := binary.BigEndian.Uint32(header[1:])

	descFunc := func(v interface{}) error {
		if err := p.codec.Unmarshal(readBuf[5:5+length], v.(proto.Message)); err != nil {
			return err
		}
		return nil
	}
	// todo 这里不能用dubbo handler，应该用pb的handler，否则无法protocol解包
	// 需要cli支持
	//var v interface{}
	//p.codec.Unmarshal(readBuf[5:5+length], v.(proto.Message))
	//var args []interface{}
	//args = append(args, v)
	//// 执行函数
	//handler(invocation.NewRPCInvocation(method,args,nil))
	ds, ok := service.(Dubbo3GrpcService)
	if !ok{
		logger.Error("service is not Dubbo3GrpcService")
	}

	// todo 明天 尝试一下用proto-gen-dubbo生成的desc能不能成功调用。
	reply, err := ds.ServiceDesc().Methods[0].Handler(service, context.Background(), descFunc, nil)
	if err != nil {
		return nil, err
	}

	replyData, err := proto.Marshal(reply.(proto.Message))
	if err != nil {
		return nil, err
	}
	rsp := make([]byte, 5+len(replyData))
	rsp[0] = byte(0)
	binary.BigEndian.PutUint32(rsp[1:], uint32(len(replyData)))
	copy(rsp[5:], replyData[:])
	return bytes.NewBuffer(rsp), nil
}

///// 先放着
//// server is used to implement helloworld.GreeterServer.
//type server struct{}
//
//// SayHello implements helloworld.GreeterServer
//func (s server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
//	fmt.Println("######### get server request name :" + in.Name)
//	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
//}
