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

package triple_protocol

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/emptypb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type leaseProtocolClient struct {
	protocolClient
	conn StreamingClientConn
}

func (p leaseProtocolClient) NewConn(context.Context, Spec, http.Header) StreamingClientConn {
	return p.conn
}
func TestServerStreamRequestFailureReleasesCodec(t *testing.T) {
	for _, responseError := range []error{nil, errors.New("response close failed")} {
		primary := errors.New("request close failed")
		raw := &stubStreamingClientConn{requestHdr: make(http.Header), closeReqErr: primary, closeRespErr: responseError}
		owner := &serialization.Resources{}
		probe := &codecCloseProbe{}
		owner.Add(probe)
		cli := NewClient(http.DefaultClient, "http://codec.invalid/codec.Service", WithSerializationResources(owner))
		cli.protocolClient = leaseProtocolClient{protocolClient: cli.protocolClient, conn: raw}
		stream, err := cli.CallServerStream(context.Background(), NewRequest(&emptypb.Empty{}), "Call")
		require.Nil(t, stream)
		require.ErrorIs(t, err, primary)
		require.Equal(t, 1, raw.closeRespCalls)
		done := make(chan struct{})
		go func() { _ = owner.Close(); close(done) }()
		codecWait(t, done)
		require.EqualValues(t, 1, probe.calls.Load())
	}
}
func TestServerStreamHalfCloseRetainsCodec(t *testing.T) {
	raw := &stubStreamingClientConn{requestHdr: make(http.Header), receiveErr: io.EOF}
	owner := &serialization.Resources{}
	probe := &codecCloseProbe{}
	owner.Add(probe)
	cli := NewClient(http.DefaultClient, "http://codec.invalid/codec.Service", WithSerializationResources(owner))
	cli.protocolClient = leaseProtocolClient{protocolClient: cli.protocolClient, conn: raw}
	stream, err := cli.CallServerStream(context.Background(), NewRequest(&emptypb.Empty{}), "Call")
	require.NoError(t, err)
	done := make(chan struct{})
	go func() { _ = owner.Close(); close(done) }()
	select {
	case <-done:
		t.Fatal("request half-close released an active response")
	case <-time.After(20 * time.Millisecond):
	}
	require.Zero(t, raw.closeRespCalls)
	require.False(t, stream.Receive(&emptypb.Empty{}))
	codecWait(t, done)
	require.NoError(t, stream.Close())
	require.EqualValues(t, 1, probe.calls.Load())
}
