package dubbo3

import (
	"bytes"
	"google.golang.org/grpc"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
)

// recvMsg represents the received msg from the transport. All transport
// protocol specific info has been removed.
type MsgType uint8

const DataMsgType = MsgType(1)
const ServerStreamCloseMsgType = MsgType(2)

type BufferMsg struct {
	buffer *bytes.Buffer
	// nil: received some data
	// io.EOF: stream is completed. data is nil.
	// other non-nil error: transport failure. data is nil.
	msgType MsgType
	err     error
}

func (bm *BufferMsg) GetMsgType() MsgType {
	return bm.msgType
}

type MsgBuffer struct {
	c   chan BufferMsg
	err error
}

func newRecvBuffer() *MsgBuffer {
	b := &MsgBuffer{
		c: make(chan BufferMsg, 1),
	}
	return b
}

func (b *MsgBuffer) put(r BufferMsg) {
	b.c <- r
}

// get returns the channel that receives a recvMsg in the buffer.
//
// Upon receipt of a recvMsg, the caller should call load to send another
// recvMsg onto the channel if there is any.
func (b *MsgBuffer) get() <-chan BufferMsg {
	return b.c
}

type stream interface {
	putRecv(data []byte, msgType MsgType)
	putSend(data []byte, msgType MsgType)
	getSend() <-chan BufferMsg
	getRecv() <-chan BufferMsg
}

type baseStream struct {
	ID      uint32
	recvBuf *MsgBuffer
	sendBuf *MsgBuffer
	url     *common.URL
	header  remoting.ProtocolHeader
	service common.RPCService
}

func (s *baseStream) putRecv(data []byte, msgType MsgType) {
	s.recvBuf.put(BufferMsg{
		buffer:  bytes.NewBuffer(data),
		msgType: msgType,
	})
}

func (s *baseStream) putSend(data []byte, msgType MsgType) {
	s.sendBuf.put(BufferMsg{
		buffer:  bytes.NewBuffer(data),
		msgType: msgType,
	})
}

func (s *baseStream) getRecv() <-chan BufferMsg {
	return s.recvBuf.get()
}

func (s *baseStream) getSend() <-chan BufferMsg {
	return s.sendBuf.get()
}

func newBaseStream(streamID uint32, url *common.URL, service common.RPCService) *baseStream {
	// stream and pkgHeader are the same level
	return &baseStream{
		url:     url,
		ID:      streamID,
		recvBuf: newRecvBuffer(),
		sendBuf: newRecvBuffer(),
		service: service,
	}
}

type serverStream struct {
	baseStream
	processor processor
	header    remoting.ProtocolHeader
}

func newServerStream(header remoting.ProtocolHeader, desc interface{}, url *common.URL, service common.RPCService) (*serverStream, error) {
	baseStream := newBaseStream(header.GetStreamID(), url, service)

	serverStream := &serverStream{
		baseStream: *baseStream,
		header:     header,
	}
	pkgHandler, err := remoting.GetPackagerHandler(url.Protocol)
	if err != nil {
		logger.Error("GetPkgHandler error with err = ", err)
		return nil, err
	}
	if methodDesc, ok := desc.(grpc.MethodDesc); ok {
		// pkgHandler and processor are the same level
		serverStream.processor, err = newUnaryProcessor(serverStream, pkgHandler, methodDesc)
	} else if streamDesc, ok := desc.(grpc.StreamDesc); ok {
		serverStream.processor, err = newStreamingProcessor(serverStream, pkgHandler, streamDesc)
	} else {
		logger.Error("grpc desc invalid:", desc)
		return nil, nil
	}

	serverStream.processor.runRPC()

	return serverStream, nil
}

func (s *serverStream) getService() common.RPCService {
	return s.service
}

func (s *serverStream) getHeader() remoting.ProtocolHeader {
	return s.header
}

func (s *serverStream) getID() uint32 {
	return s.ID
}

type clientStream struct {
	baseStream
}

func newClientStream(streamID uint32, url *common.URL) *clientStream {
	baseStream := newBaseStream(streamID, url, nil)
	return &clientStream{
		baseStream: *baseStream,
	}
}
