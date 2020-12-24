package dubbo3

import (
	"context"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
	"google.golang.org/grpc/metadata"
)

type baseUserStream struct {
	stream     stream
	serilizer  remoting.Dubbo3Serializer
	pkgHandler remoting.PackageHandler
}

func (ss *baseUserStream) SetHeader(metadata.MD) error {
	return nil
}
func (ss *baseUserStream) SendHeader(metadata.MD) error {
	return nil
}
func (ss *baseUserStream) SetTrailer(metadata.MD) {

}
func (ss *baseUserStream) Context() context.Context {
	return nil
}
func (ss *baseUserStream) SendMsg(m interface{}) error {
	// 这里直接调用stream上的packageHandler 的 decode函数，从msg到byte
	replyData, err := ss.serilizer.Marshal(m)
	if err != nil {
		logger.Error("sen msg error with msg = ", m)
		return err
	}
	rspFrameData := ss.pkgHandler.Pkg2FrameData(replyData)
	ss.stream.putSend(rspFrameData, DataMsgType)
	return nil
}

func (ss *baseUserStream) RecvMsg(m interface{}) error {
	recvChan := ss.stream.getRecv()
	readBuf := <-recvChan
	pkgData := ss.pkgHandler.Frame2PkgData(readBuf.buffer.Bytes())
	if err := ss.serilizer.Unmarshal(pkgData, m); err != nil {
		return err
	}
	return nil
}

type serverUserStream struct {
	baseUserStream
}

func newServerUserStream(s stream, serilizer remoting.Dubbo3Serializer, pkgHandler remoting.PackageHandler) *serverUserStream {
	return &serverUserStream{
		baseUserStream: baseUserStream{
			serilizer:  serilizer,
			pkgHandler: pkgHandler,
			stream:     s,
		},
	}
}

type clientUserStream struct {
	baseUserStream
}

func (ss *clientUserStream) Header() (metadata.MD, error) {
	return nil, nil
}
func (ss *clientUserStream) Trailer() metadata.MD {
	return nil
}
func (ss *clientUserStream) CloseSend() error {
	// todo
	return nil
}

func newClientUserStream(s stream, serilizer remoting.Dubbo3Serializer, pkgHandler remoting.PackageHandler) *clientUserStream {
	return &clientUserStream{
		baseUserStream: baseUserStream{
			serilizer:  serilizer,
			pkgHandler: pkgHandler,
			stream:     s,
		},
	}
}
