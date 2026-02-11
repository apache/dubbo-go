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

package client

import (
	"context"
	"errors"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type fakeInvoker struct {
	lastCtx        context.Context
	lastInvocation base.Invocation
	res            result.Result
}

func (f *fakeInvoker) GetURL() *common.URL {
	return nil
}

func (f *fakeInvoker) IsAvailable() bool {
	return true
}

func (f *fakeInvoker) Destroy() {}

func (f *fakeInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	f.lastCtx = ctx
	f.lastInvocation = inv
	return f.res
}

func TestClientDefinitionConnection(t *testing.T) {
	var def ClientDefinition
	conn, err := def.GetConnection()
	require.Error(t, err)
	require.Nil(t, conn)

	expectedConn := &Connection{}
	def.SetConnection(expectedConn)
	gotConn, err := def.GetConnection()
	require.NoError(t, err)
	require.Equal(t, expectedConn, gotConn)
}

func TestGenerateInvocation(t *testing.T) {
	var resp int
	opts := &CallOptions{RequestTimeout: "1s", Retries: "2"}

	inv, err := generateInvocation(context.Background(), "Echo", []any{"foo", 1}, &resp, constant.CallUnary, opts)
	require.NoError(t, err)

	timeout, _ := inv.GetAttachment(constant.TimeoutKey)
	retries, _ := inv.GetAttachment(constant.RetriesKey)
	require.Equal(t, "1s", timeout)
	require.Equal(t, "2", retries)

	attr, ok := inv.GetAttribute(constant.CallTypeKey)
	require.True(t, ok)
	require.Equal(t, constant.CallUnary, attr)

	require.Equal(t, []any{"foo", 1}, inv.Arguments())
	require.Equal(t, &resp, inv.Reply())
	require.Equal(t, []any{"foo", 1, &resp}, inv.ParameterRawValues())
}

func TestGenerateInvocationWithContextAttachments(t *testing.T) {
	var resp int
	opts := &CallOptions{RequestTimeout: "1s", Retries: "2"}

	userAttachments := map[string]any{
		"userKey1": "userValue1",
		"userKey2": 12345,
		"traceID":  "abc-123",
	}
	ctx := context.WithValue(context.Background(), constant.AttachmentKey, userAttachments)

	inv, err := generateInvocation(ctx, "Echo", []any{"foo", 1}, &resp, constant.CallUnary, opts)
	require.NoError(t, err)

	timeout, _ := inv.GetAttachment(constant.TimeoutKey)
	retries, _ := inv.GetAttachment(constant.RetriesKey)
	require.Equal(t, "1s", timeout)
	require.Equal(t, "2", retries)

	require.Equal(t, "userValue1", inv.GetAttachmentInterface("userKey1"))
	require.Equal(t, 12345, inv.GetAttachmentInterface("userKey2"))
	require.Equal(t, "abc-123", inv.GetAttachmentInterface("traceID"))
}

func TestConnectionCallPassesOptions(t *testing.T) {
	invRes := &result.RPCResult{}
	invoker := &fakeInvoker{res: invRes}
	conn := &Connection{refOpts: &ReferenceOptions{invoker: invoker}}

	var resp string
	res, err := conn.call(context.Background(), []any{"req"}, &resp, "Ping", constant.CallUnary, WithCallRequestTimeout(1500*time.Millisecond), WithCallRetries(3))
	require.NoError(t, err)
	require.Equal(t, invRes, res)

	inv := invoker.lastInvocation
	timeout, _ := inv.GetAttachment(constant.TimeoutKey)
	retries, _ := inv.GetAttachment(constant.RetriesKey)
	require.Equal(t, "1.5s", timeout)
	require.Equal(t, "3", retries)

	requireCallType(t, inv, constant.CallUnary)
	require.Equal(t, []any{"req", &resp}, inv.ParameterRawValues())
}

func TestCallUnary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		invoker := &fakeInvoker{res: &result.RPCResult{}}
		conn := &Connection{refOpts: &ReferenceOptions{invoker: invoker}}

		var resp string
		err := conn.CallUnary(context.Background(), []any{"a"}, &resp, "Unary")
		require.NoError(t, err)
		requireCallType(t, invoker.lastInvocation, constant.CallUnary)
		require.Equal(t, []any{"a", &resp}, invoker.lastInvocation.ParameterRawValues())
	})

	t.Run("error", func(t *testing.T) {
		resErr := errors.New("fail")
		invoker := &fakeInvoker{res: &result.RPCResult{Err: resErr}}
		conn := &Connection{refOpts: &ReferenceOptions{invoker: invoker}}

		var resp string
		err := conn.CallUnary(context.Background(), []any{"a"}, &resp, "Unary")
		require.ErrorIs(t, err, resErr)
	})
}

func TestCallClientStream(t *testing.T) {
	resVal := "stream"
	invoker := &fakeInvoker{res: &result.RPCResult{Rest: resVal}}
	conn := &Connection{refOpts: &ReferenceOptions{invoker: invoker}}

	out, err := conn.CallClientStream(context.Background(), "ClientStream")
	require.NoError(t, err)
	require.Equal(t, resVal, out)

	requireCallType(t, invoker.lastInvocation, constant.CallClientStream)
	require.Empty(t, invoker.lastInvocation.ParameterRawValues())
}

func TestCallServerStream(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := "payload"
		resVal := "server-stream"
		invoker := &fakeInvoker{res: &result.RPCResult{Rest: resVal}}
		conn := &Connection{refOpts: &ReferenceOptions{invoker: invoker}}

		out, err := conn.CallServerStream(context.Background(), req, "ServerStream")
		require.NoError(t, err)
		require.Equal(t, resVal, out)

		requireCallType(t, invoker.lastInvocation, constant.CallServerStream)
		require.Equal(t, []any{req}, invoker.lastInvocation.ParameterRawValues())
	})

	t.Run("error", func(t *testing.T) {
		req := "payload"
		resErr := errors.New("stream err")
		invoker := &fakeInvoker{res: &result.RPCResult{Err: resErr}}
		conn := &Connection{refOpts: &ReferenceOptions{invoker: invoker}}

		out, err := conn.CallServerStream(context.Background(), req, "ServerStream")
		require.ErrorIs(t, err, resErr)
		require.Nil(t, out)
	})
}

func TestCallBidiStream(t *testing.T) {
	resVal := "bidi"
	invoker := &fakeInvoker{res: &result.RPCResult{Rest: resVal}}
	conn := &Connection{refOpts: &ReferenceOptions{invoker: invoker}}

	out, err := conn.CallBidiStream(context.Background(), "Bidi")
	require.NoError(t, err)
	require.Equal(t, resVal, out)

	requireCallType(t, invoker.lastInvocation, constant.CallBidiStream)
	require.Empty(t, invoker.lastInvocation.ParameterRawValues())
}

func requireCallType(t *testing.T, inv base.Invocation, callType string) {
	t.Helper()
	attr, ok := inv.GetAttribute(constant.CallTypeKey)
	require.True(t, ok)
	require.Equal(t, callType, attr)
}
