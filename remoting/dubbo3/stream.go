/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dubbo3

import (
	"bytes"
	"github.com/apache/dubbo-go/remoting/dubbo3/status"
)
import (
	"google.golang.org/grpc"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
)

// MsgType show the type of Message in buffer
type MsgType uint8

const (
	DataMsgType              = MsgType(1)
	ServerStreamCloseMsgType = MsgType(2)
)

// BufferMsg is the basic transfer unit in one stream
type BufferMsg struct {
	buffer  *bytes.Buffer
	msgType MsgType
	st *status.Status
	err     error
}

// GetMsgType can get buffer's type
func (bm *BufferMsg) GetMsgType() MsgType {
	return bm.msgType
}

// MsgBuffer contain the chan of BufferMsg
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

func (b *MsgBuffer) get() <-chan BufferMsg {
	return b.c
}

type stream interface {
	putRecv(data []byte, msgType MsgType)
	putSend(data []byte, msgType MsgType)
	putRecvErr(err error)
	getSend() <-chan BufferMsg
	getRecv() <-chan BufferMsg
}

// baseStream is the basic  impl of stream interface, it impl for basic function of stream
type baseStream struct {
	ID      uint32
	recvBuf *MsgBuffer
	sendBuf *MsgBuffer
	url     *common.URL
	header  remoting.ProtocolHeader
	service common.RPCService

	// On client-side it is the status error received from the server.
	// On server-side it is unused.
	status *status.Status
}

func (s *baseStream) WriteStatus(st *status.Status) {
	s.sendBuf.put(BufferMsg{
		st : st,
		msgType: ServerStreamCloseMsgType,
	})
}

func (s *baseStream) putRecv(data []byte, msgType MsgType) {
	s.recvBuf.put(BufferMsg{
		buffer:  bytes.NewBuffer(data),
		msgType: msgType,
	})
}

func (s *baseStream) putRecvErr(err error) {
	s.recvBuf.put(BufferMsg{
		err: err,
		msgType: ServerStreamCloseMsgType,
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

// serverStream is running in server end
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

// clientStream is running in client end
type clientStream struct {
	baseStream
}

func newClientStream(streamID uint32, url *common.URL) *clientStream {
	baseStream := newBaseStream(streamID, url, nil)
	return &clientStream{
		baseStream: *baseStream,
	}
}
