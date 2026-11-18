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
	"net/http/httptest"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/emptypb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

func TestRetiringOldSerializationOwnerPreservesReplacement(t *testing.T) {
	server := NewServer("", nil)
	oldOwner, newOwner := &serialization.Resources{}, &serialization.Resources{}
	oldProbe, newProbe := &codecCloseProbe{}, &codecCloseProbe{}
	oldOwner.Add(oldProbe)
	newOwner.Add(newProbe)
	entered, gate := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(gate) })
	require.NoError(t, server.RegisterUnaryHandler("/codec.Retire/Call", func() any { return &emptypb.Empty{} }, func(context.Context, *Request) (*Response, error) {
		close(entered)
		<-gate
		return NewResponse(&emptypb.Empty{}), nil
	}, WithSerializationResources(oldOwner)))
	transport := httptest.NewServer(server.mux)
	defer func() { once.Do(func() { close(gate) }); transport.Close() }()
	cli := NewClient(transport.Client(), transport.URL+"/codec.Retire")
	call := func() error {
		return cli.CallUnary(context.Background(), NewRequest(&emptypb.Empty{}), "Call", NewResponse(&emptypb.Empty{}))
	}
	result := make(chan error, 1)
	go func() { result <- call() }()
	codecWait(t, entered)
	require.NoError(t, server.RegisterUnaryHandler("/codec.Retire/Call", func() any { return &emptypb.Empty{} }, func(context.Context, *Request) (*Response, error) { return NewResponse(&emptypb.Empty{}), nil }, WithSerializationResources(newOwner)))
	server.RemoveSerializationOwner(oldOwner)
	require.NoError(t, call())
	router := server.handlers["/codec.Retire/Call"]
	require.Nil(t, router.codecResources)
	require.Empty(t, router.implementations)
	for _, selected := range router.variants {
		require.Same(t, newOwner, selected.serializationOwner)
	}
	closed := make(chan struct{})
	go func() { _ = oldOwner.Close(); close(closed) }()
	require.Zero(t, oldProbe.calls.Load())
	once.Do(func() { close(gate) })
	require.NoError(t, <-result)
	codecWait(t, closed)
	require.EqualValues(t, 1, oldProbe.calls.Load())
	server.RemoveSerializationOwner(newOwner)
	require.Equal(t, CodeUnimplemented, CodeOf(call()))
	require.Empty(t, router.variants)
	require.NoError(t, newOwner.Close())
	require.EqualValues(t, 1, newProbe.calls.Load())
}
