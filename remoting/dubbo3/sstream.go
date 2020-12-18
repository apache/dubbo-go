package dubbo3

import (
	"context"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
	"google.golang.org/grpc/metadata"
)

type serverStream struct {
	stream     *stream
	serilizer  remoting.Dubbo3Serializer
	pkgHandler remoting.PackageHandler
}

func (ss *serverStream) SetHeader(metadata.MD) error {
	return nil
}
func (ss *serverStream) SendHeader(metadata.MD) error {
	return nil
}
func (ss *serverStream) SetTrailer(metadata.MD) {

}
func (ss *serverStream) Context() context.Context {
	return nil
}
func (ss *serverStream) SendMsg(m interface{}) error {
	// 这里直接调用stream上的packageHandler 的 decode函数，从msg到byte
	replyData, err := ss.serilizer.Marshal(m)
	if err != nil {
		logger.Error("sen msg error with msg = ", m)
		return err
	}
	rspFrameData := ss.pkgHandler.Pkg2FrameData(replyData)
	ss.stream.putSend(rspFrameData)
	return nil
}

func (ss *serverStream) RecvMsg(m interface{}) error {
	recvChan := ss.stream.getRecv()
	readBuf := <-recvChan
	pkgData := ss.pkgHandler.Frame2PkgData(readBuf.buffer.Bytes())
	if err := ss.serilizer.Unmarshal(pkgData, m); err != nil {
		return err
	}
	return nil
}

func newServerStream(s *stream, serilizer remoting.Dubbo3Serializer, pkgHandler remoting.PackageHandler) *serverStream {
	return &serverStream{
		serilizer:  serilizer,
		pkgHandler: pkgHandler,
		stream:     s,
	}
}
