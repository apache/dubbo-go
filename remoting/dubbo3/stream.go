package dubbo3

import (
	"bytes"
	"fmt"
	"github.com/apache/dubbo-go/common"
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
	method    string
	processor *processor
	service   common.RPCService
	url       *common.URL
}

func newStream(data parsedTripleHeaderData, service common.RPCService, url *common.URL) *stream {
	newStream := &stream{
		url:     url,
		ID:      data.streamID,
		method:  data.method,
		recvBuf: newRecvBuffer(),
		sendBuf: newRecvBuffer(),
		service: service,
	}
	newStream.processor = newProcessor(newStream)
	go newStream.run()
	return newStream
}

func (s *stream) run() {
	recvChan := s.recvBuf.get()
	for {
		recvMsg := <-recvChan
		if recvMsg.err != nil {
			continue
		}
		rspBuffer, err := s.processor.processUnaryRPC(*recvMsg.buffer, s.method, s.service, s.url)
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
