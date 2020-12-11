package dubbo3

import (
	"bytes"
	"fmt"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
	"google.golang.org/grpc"
	"sync"
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
	processor *processor
	url       *common.URL
	header    remoting.ProtocolHeader
	md        grpc.MethodDesc
	service   common.RPCService
}

func newStream(header remoting.ProtocolHeader, md grpc.MethodDesc, url *common.URL, service common.RPCService) (*stream, error) {
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
		md:      md,
		service: service,
	}

	// pkgHandler and processor are the same level
	newStream.processor, err = newProcessor(newStream, pkgHandler, md)
	if err != nil {
		logger.Errorf("newStream error: ", err)
		return nil, err
	}
	go newStream.run()
	return newStream, nil
}

func (s *stream) run() {
	// stream 建立时，获得抽象protocHeader，同时根据protocHeader拿到了实现好的对应协议的package Handler
	// package Handler里面封装了协议codec codec里面封装了 与协议独立的serillizer

	//拿到了本次调用的打解包协议类型、调用的方法名。

	recvChan := s.recvBuf.get()
	for {
		recvMsg := <-recvChan
		if recvMsg.err != nil {
			continue
		}
		rspBuffer, err := s.processor.processUnaryRPC(*recvMsg.buffer, s.service)
		if err != nil {
			fmt.Println("error ,s.processUnaryRPC err = ", err)
			continue
		}
		s.sendBuf.put(BufferMsg{buffer: rspBuffer})
	}
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
