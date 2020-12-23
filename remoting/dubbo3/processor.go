package dubbo3

import (
	"bytes"
	"github.com/apache/dubbo-go/protocol"
)
import (
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
)

type processor interface {
	runRPC()
}

type baseProcessor struct {
	stream                *serverStream
	pkgHandler            remoting.PackageHandler
	readWriteMaxBufferLen uint32 // useless
	serializer            remoting.Dubbo3Serializer
}

type unaryProcessor struct {
	baseProcessor
	methodDesc grpc.MethodDesc
}

// protoc config参数增加,对codec进行选择
func newUnaryProcessor(s *serverStream, pkgHandler remoting.PackageHandler, desc grpc.MethodDesc) (processor, error) {
	serilizer, err := remoting.GetDubbo3Serializer(defaultSerilization)
	if err != nil {
		logger.Error("newProcessor with serlizationg ", defaultSerilization, " error")
		return nil, err
	}

	return &unaryProcessor{
		baseProcessor: baseProcessor{
			serializer:            serilizer,
			stream:                s,
			pkgHandler:            pkgHandler,
			readWriteMaxBufferLen: defaultRWBufferMaxLen,
		},
		methodDesc: desc,
	}, nil
}

func (p *unaryProcessor) processUnaryRPC(buf bytes.Buffer, service common.RPCService, header remoting.ProtocolHeader) ([]byte, error) {
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
	return rspFrameData, nil
}

func (s *unaryProcessor) runRPC() {
	// stream 建立时，获得抽象protocHeader，同时根据protocHeader拿到了实现好的对应协议的package Handler
	// package Handler里面封装了协议codec codec里面封装了 与协议独立的serillizer
	//拿到了本次调用的打解包协议类型、调用的方法名。

	recvChan := s.stream.getRecv()
	go func() {
		for {
			recvMsg := <-recvChan
			if recvMsg.err != nil {
				continue
			}
			rspData, err := s.processUnaryRPC(*recvMsg.buffer, s.stream.getService(), s.stream.getHeader())
			if err != nil {
				logger.Error("error ,s.processUnaryRPC err = ", err)
				continue
			}
			s.stream.putSend(rspData)
		}
	}()

}

type streamingProcessor struct {
	baseProcessor
	streamDesc grpc.StreamDesc
}

func newStreamingProcessor(s *serverStream, pkgHandler remoting.PackageHandler, desc grpc.StreamDesc) (processor, error) {
	serilizer, err := remoting.GetDubbo3Serializer(defaultSerilization)
	if err != nil {
		logger.Error("newProcessor with serlizationg ", defaultSerilization, " error")
		return nil, err
	}

	return &streamingProcessor{
		baseProcessor: baseProcessor{
			serializer:            serilizer,
			stream:                s,
			pkgHandler:            pkgHandler,
			readWriteMaxBufferLen: defaultRWBufferMaxLen,
		},
		streamDesc: desc,
	}, nil
}

func (sp *streamingProcessor) runRPC() {
	serverUserstream := newServerUserStream(sp.stream, sp.serializer, sp.pkgHandler)
	go sp.streamDesc.Handler(sp.stream.getService(), serverUserstream)
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
