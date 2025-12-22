// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple_protocol

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
)

func TestClientStreamForClient_NoPanics(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	clientStream := &ClientStreamForClient{err: initErr}
	assert.ErrorIs(t, clientStream.Send(&pingv1.PingRequest{}), initErr)
	verifyHeaders(t, clientStream.RequestHeader())
	res := NewResponse(&pingv1.PingResponse{})
	err := clientStream.CloseAndReceive(res)
	assert.ErrorIs(t, err, initErr)
	conn, err := clientStream.Conn()
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func TestClientStreamForClient_WithConn(t *testing.T) {
	t.Parallel()

	t.Run("spec and peer", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{
			spec: Spec{Procedure: "/test.Service/Method", StreamType: StreamTypeClient},
			peer: Peer{Addr: "localhost:8080", Protocol: "grpc"},
		}
		stream := &ClientStreamForClient{conn: mockConn}

		assert.Equal(t, stream.Spec().Procedure, "/test.Service/Method")
		assert.Equal(t, stream.Spec().StreamType, StreamTypeClient)
		assert.Equal(t, stream.Peer().Addr, "localhost:8080")
		assert.Equal(t, stream.Peer().Protocol, "grpc")
	})

	t.Run("request header", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{
			requestHeader: http.Header{"X-Custom": []string{"value"}},
		}
		stream := &ClientStreamForClient{conn: mockConn}

		assert.Equal(t, stream.RequestHeader().Get("X-Custom"), "value")
	})

	t.Run("send success", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{}
		stream := &ClientStreamForClient{conn: mockConn}

		err := stream.Send(&pingv1.PingRequest{Number: 42})
		assert.Nil(t, err)
	})

	t.Run("conn method", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{}
		stream := &ClientStreamForClient{conn: mockConn}

		conn, err := stream.Conn()
		assert.Nil(t, err)
		assert.NotNil(t, conn)
	})
}

func TestServerStreamForClient_NoPanics(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	serverStream := &ServerStreamForClient{constructErr: initErr}
	assert.ErrorIs(t, serverStream.Err(), initErr)
	assert.ErrorIs(t, serverStream.Close(), initErr)
	assert.Nil(t, serverStream.Msg())
	assert.False(t, serverStream.Receive(NewResponse(&pingv1.PingResponse{})))
	verifyHeaders(t, serverStream.ResponseHeader())
	verifyHeaders(t, serverStream.ResponseTrailer())
	conn, err := serverStream.Conn()
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func TestServerStreamForClient(t *testing.T) {
	t.Parallel()
	stream := &ServerStreamForClient{conn: &nopStreamingClientConn{}}
	res := NewResponse(&pingv1.PingResponse{})
	assert.True(t, stream.Receive(res))
	first := fmt.Sprintf("%p", stream.Msg())
	res1 := NewResponse(&pingv1.PingResponse{})
	assert.True(t, stream.Receive(res1))
	second := fmt.Sprintf("%p", stream.Msg())
	assert.NotEqual(t, first, second)
	conn, err := stream.Conn()
	assert.Nil(t, err)
	assert.NotNil(t, conn)
}

func TestServerStreamForClient_ReceiveError(t *testing.T) {
	t.Parallel()

	t.Run("non-EOF error", func(t *testing.T) {
		t.Parallel()
		receiveErr := errors.New("receive failed")
		mockConn := &fullMockStreamingClientConn{receiveErr: receiveErr}
		stream := &ServerStreamForClient{conn: mockConn}

		result := stream.Receive(&pingv1.PingResponse{})
		assert.False(t, result)
		assert.ErrorIs(t, stream.Err(), receiveErr)
	})

	t.Run("EOF error is not returned as Err", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{receiveErr: io.EOF}
		stream := &ServerStreamForClient{conn: mockConn}

		result := stream.Receive(&pingv1.PingResponse{})
		assert.False(t, result)
		assert.Nil(t, stream.Err())
	})

	t.Run("already has receive error", func(t *testing.T) {
		t.Parallel()
		prevErr := errors.New("previous error")
		stream := &ServerStreamForClient{
			conn:       &fullMockStreamingClientConn{},
			receiveErr: prevErr,
		}

		result := stream.Receive(&pingv1.PingResponse{})
		assert.False(t, result)
	})

	t.Run("already has construct error", func(t *testing.T) {
		t.Parallel()
		constructErr := errors.New("construct error")
		stream := &ServerStreamForClient{
			conn:         &fullMockStreamingClientConn{},
			constructErr: constructErr,
		}

		result := stream.Receive(&pingv1.PingResponse{})
		assert.False(t, result)
	})
}

func TestServerStreamForClient_Headers(t *testing.T) {
	t.Parallel()

	t.Run("response header", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{
			responseHeader: http.Header{"X-Response": []string{"header-value"}},
		}
		stream := &ServerStreamForClient{conn: mockConn}

		assert.Equal(t, stream.ResponseHeader().Get("X-Response"), "header-value")
	})

	t.Run("response trailer", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{
			responseTrailer: http.Header{"X-Trailer": []string{"trailer-value"}},
		}
		stream := &ServerStreamForClient{conn: mockConn}

		assert.Equal(t, stream.ResponseTrailer().Get("X-Trailer"), "trailer-value")
	})
}

func TestServerStreamForClient_Close(t *testing.T) {
	t.Parallel()

	t.Run("close success", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{}
		stream := &ServerStreamForClient{conn: mockConn}

		err := stream.Close()
		assert.Nil(t, err)
	})

	t.Run("close error", func(t *testing.T) {
		t.Parallel()
		closeErr := errors.New("close failed")
		mockConn := &fullMockStreamingClientConn{closeResErr: closeErr}
		stream := &ServerStreamForClient{conn: mockConn}

		err := stream.Close()
		assert.ErrorIs(t, err, closeErr)
	})
}

func TestBidiStreamForClient_NoPanics(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	bidiStream := &BidiStreamForClient{err: initErr}
	res := NewResponse(&pingv1.CumSumResponse{})
	err := bidiStream.Receive(res)
	assert.ErrorIs(t, err, initErr)
	verifyHeaders(t, bidiStream.RequestHeader())
	verifyHeaders(t, bidiStream.ResponseHeader())
	verifyHeaders(t, bidiStream.ResponseTrailer())
	assert.ErrorIs(t, bidiStream.Send(&pingv1.CumSumRequest{}), initErr)
	assert.ErrorIs(t, bidiStream.CloseRequest(), initErr)
	assert.ErrorIs(t, bidiStream.CloseResponse(), initErr)
	conn, err := bidiStream.Conn()
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func TestBidiStreamForClient_WithConn(t *testing.T) {
	t.Parallel()

	t.Run("spec and peer", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{
			spec: Spec{Procedure: "/test.Service/BidiMethod", StreamType: StreamTypeBidi},
			peer: Peer{Addr: "localhost:9090", Protocol: "grpc"},
		}
		stream := &BidiStreamForClient{conn: mockConn}

		assert.Equal(t, stream.Spec().Procedure, "/test.Service/BidiMethod")
		assert.Equal(t, stream.Spec().StreamType, StreamTypeBidi)
		assert.Equal(t, stream.Peer().Addr, "localhost:9090")
		assert.Equal(t, stream.Peer().Protocol, "grpc")
	})

	t.Run("headers", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{
			requestHeader:   http.Header{"X-Request": []string{"req-value"}},
			responseHeader:  http.Header{"X-Response": []string{"res-value"}},
			responseTrailer: http.Header{"X-Trailer": []string{"trail-value"}},
		}
		stream := &BidiStreamForClient{conn: mockConn}

		assert.Equal(t, stream.RequestHeader().Get("X-Request"), "req-value")
		assert.Equal(t, stream.ResponseHeader().Get("X-Response"), "res-value")
		assert.Equal(t, stream.ResponseTrailer().Get("X-Trailer"), "trail-value")
	})

	t.Run("send and receive", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{}
		stream := &BidiStreamForClient{conn: mockConn}

		err := stream.Send(&pingv1.CumSumRequest{Number: 1})
		assert.Nil(t, err)

		err = stream.Receive(&pingv1.CumSumResponse{})
		assert.Nil(t, err)
	})

	t.Run("close request and response", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{}
		stream := &BidiStreamForClient{conn: mockConn}

		err := stream.CloseRequest()
		assert.Nil(t, err)

		err = stream.CloseResponse()
		assert.Nil(t, err)
	})

	t.Run("conn method", func(t *testing.T) {
		t.Parallel()
		mockConn := &fullMockStreamingClientConn{}
		stream := &BidiStreamForClient{conn: mockConn}

		conn, err := stream.Conn()
		assert.Nil(t, err)
		assert.NotNil(t, conn)
	})
}

func TestBidiStreamForClient_Errors(t *testing.T) {
	t.Parallel()

	t.Run("send error", func(t *testing.T) {
		t.Parallel()
		sendErr := errors.New("send failed")
		mockConn := &fullMockStreamingClientConn{sendErr: sendErr}
		stream := &BidiStreamForClient{conn: mockConn}

		err := stream.Send(&pingv1.CumSumRequest{})
		assert.ErrorIs(t, err, sendErr)
	})

	t.Run("receive error", func(t *testing.T) {
		t.Parallel()
		receiveErr := errors.New("receive failed")
		mockConn := &fullMockStreamingClientConn{receiveErr: receiveErr}
		stream := &BidiStreamForClient{conn: mockConn}

		err := stream.Receive(&pingv1.CumSumResponse{})
		assert.ErrorIs(t, err, receiveErr)
	})

	t.Run("close request error", func(t *testing.T) {
		t.Parallel()
		closeErr := errors.New("close request failed")
		mockConn := &fullMockStreamingClientConn{closeReqErr: closeErr}
		stream := &BidiStreamForClient{conn: mockConn}

		err := stream.CloseRequest()
		assert.ErrorIs(t, err, closeErr)
	})

	t.Run("close response error", func(t *testing.T) {
		t.Parallel()
		closeErr := errors.New("close response failed")
		mockConn := &fullMockStreamingClientConn{closeResErr: closeErr}
		stream := &BidiStreamForClient{conn: mockConn}

		err := stream.CloseResponse()
		assert.ErrorIs(t, err, closeErr)
	})
}

func verifyHeaders(t *testing.T, headers http.Header) {
	t.Helper()
	assert.Equal(t, headers, http.Header{})

	// Verify set/del don't panic
	headers.Set("a", "b")
	headers.Del("a")
}

type nopStreamingClientConn struct {
	StreamingClientConn
}

func (c *nopStreamingClientConn) Receive(msg any) error {
	return nil
}

// fullMockStreamingClientConn is a complete mock for StreamingClientConn
type fullMockStreamingClientConn struct {
	StreamingClientConn
	spec            Spec
	peer            Peer
	requestHeader   http.Header
	responseHeader  http.Header
	responseTrailer http.Header
	sendErr         error
	receiveErr      error
	closeReqErr     error
	closeResErr     error
}

func (m *fullMockStreamingClientConn) Spec() Spec {
	return m.spec
}

func (m *fullMockStreamingClientConn) Peer() Peer {
	return m.peer
}

func (m *fullMockStreamingClientConn) RequestHeader() http.Header {
	if m.requestHeader == nil {
		m.requestHeader = make(http.Header)
	}
	return m.requestHeader
}

func (m *fullMockStreamingClientConn) ResponseHeader() http.Header {
	if m.responseHeader == nil {
		m.responseHeader = make(http.Header)
	}
	return m.responseHeader
}

func (m *fullMockStreamingClientConn) ResponseTrailer() http.Header {
	if m.responseTrailer == nil {
		m.responseTrailer = make(http.Header)
	}
	return m.responseTrailer
}

func (m *fullMockStreamingClientConn) Send(msg any) error {
	return m.sendErr
}

func (m *fullMockStreamingClientConn) Receive(msg any) error {
	return m.receiveErr
}

func (m *fullMockStreamingClientConn) CloseRequest() error {
	return m.closeReqErr
}

func (m *fullMockStreamingClientConn) CloseResponse() error {
	return m.closeResErr
}
