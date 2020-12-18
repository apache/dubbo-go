package dubbo3

import (
	"bytes"
	"google.golang.org/grpc"
	"sync"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
)

// recvMsg represents the received msg from the transport. All transport
// protocol specific info has been removed.
type BufferMsg struct {
	buffer *bytes.Buffer
	// nil: received some data
	// io.EOF: stream is completed. data is nil.
	// other non-nil error: transport failure. data is nil.
	err error
}

type MsgBuffer struct {
	c   chan BufferMsg
	mu  sync.Mutex
	err error
}

func newRecvBuffer() *MsgBuffer {
	b := &MsgBuffer{
		c: make(chan BufferMsg, 1),
	}
	return b
}

func (b *MsgBuffer) put(r BufferMsg) {
	b.mu.Lock()
	if b.err != nil {
		b.mu.Unlock()
		// An error had occurred earlier, don't accept more
		// data or errors.
		return
	}
	select {
	case b.c <- r:
		b.mu.Unlock()
		return
	default:
	}
	b.mu.Unlock()
}

// get returns the channel that receives a recvMsg in the buffer.
//
// Upon receipt of a recvMsg, the caller should call load to send another
// recvMsg onto the channel if there is any.
func (b *MsgBuffer) get() <-chan BufferMsg {
	return b.c
}

type stream struct {
	ID        uint32
	recvBuf   *MsgBuffer
	sendBuf   *MsgBuffer
	processor processor
	url       *common.URL
	header    remoting.ProtocolHeader
	desc      interface{}
	service   common.RPCService
}

func newStream(header remoting.ProtocolHeader, desc interface{}, url *common.URL, service common.RPCService) (*stream, error) {
	pkgHandler, err := remoting.GetPackagerHandler(url.Protocol)
	if err != nil {
		logger.Error("GetPkgHandler error with err = ", err)
		return nil, err
	}

	if err != nil {
		logger.Error("new stream error with err = ", err)
		return nil, err
	}

	// stream and pkgHeader are the same level
	newStream := &stream{
		url:     url,
		ID:      header.GetStreamID(),
		recvBuf: newRecvBuffer(),
		sendBuf: newRecvBuffer(),
		header:  header,
		desc:    desc,
		service: service,
	}

	if methodDesc, ok := desc.(grpc.MethodDesc); ok {
		// pkgHandler and processor are the same level
		newStream.processor, err = newUnaryProcessor(newStream, pkgHandler, methodDesc)
	} else if streamDesc, ok := desc.(grpc.StreamDesc); ok {
		newStream.processor, err = newStreamingProcessor(newStream, pkgHandler, streamDesc)
	} else {
		logger.Error("grpc desc invalid:", desc)
		return nil, nil
	}

	newStream.processor.runRPC()
	return newStream, nil
}

func (s *stream) putRecv(data []byte) {
	s.recvBuf.put(BufferMsg{
		buffer: bytes.NewBuffer(data),
	})
}

func (s *stream) putSend(data []byte) {
	s.sendBuf.put(BufferMsg{
		buffer: bytes.NewBuffer(data),
	})
}

func (s *stream) getRecv() <-chan BufferMsg {
	return s.recvBuf.get()
}

func (s *stream) getSend() <-chan BufferMsg {
	return s.sendBuf.get()
}
