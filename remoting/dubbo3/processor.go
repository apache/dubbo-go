package dubbo3

import (
	"bytes"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/remoting"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

type processor struct {
	stream                *stream
	pkgHandler            remoting.PackageHandler
	readWriteMaxBufferLen uint32 // useless
	serializer            remoting.Dubbo3Serializer
	methodDesc            grpc.MethodDesc
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
func newProcessor(s *stream, pkgHandler remoting.PackageHandler, md grpc.MethodDesc) (*processor, error) {

	serilizer, err := remoting.GetDubbo3Serializer(defaultSerilization)
	if err != nil {
		logger.Error("newProcessor with serlizationg ", defaultSerilization, " error")
		return nil, err
	}

	return &processor{
		serializer:            serilizer,
		stream:                s,
		pkgHandler:            pkgHandler,
		readWriteMaxBufferLen: defaultRWBufferMaxLen,
		methodDesc:            md,
	}, nil
}

func (p *processor) processUnaryRPC(buf bytes.Buffer, service common.RPCService, header remoting.ProtocolHeader) (*bytes.Buffer, error) {
	readBuf := buf.Bytes()

	pkgData := p.pkgHandler.Frame2PkgData(readBuf)

	descFunc := func(v interface{}) error {
		if err := p.serializer.Unmarshal(pkgData, v.(proto.Message)); err != nil {
			return err
		}
		return nil
	}

	reply, err := p.methodDesc.Handler(service, header.FieldToCtx(), descFunc, nil)
	if err != nil {
		return nil, err
	}

	// 这里直接调用stream上的packageHandler 的 decode函数，从msg到byte
	replyData, err := p.serializer.Marshal(reply.(proto.Message))
	if err != nil {
		return nil, err
	}

	rspFrameData := p.pkgHandler.Pkg2FrameData(replyData)
	return bytes.NewBuffer(rspFrameData), nil
}
