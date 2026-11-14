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

package dubbo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"runtime"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo/impl"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/serialization"
	hs "dubbo.apache.org/dubbo-go/v3/serialization/hessian2"
)

type testStream struct{ hs.Codec }

func (*testStream) Name() string { return "unit-stream" }
func init() {
	id := byte(29)
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "dubbo", Name: "unit-stream", Mode: serialization.StreamMode, DubboID: &id, New: func(serialization.Options) (serialization.Codec, error) { return &testStream{}, nil }})
}
func streamURL(name string) *common.URL {
	return common.NewURLWithOptions(common.WithProtocol("dubbo"), common.WithLocation("127.0.0.1:20000"), common.WithParamsValue("serialization", name), common.WithParamsValue("side", "consumer"))
}
func TestConfiguredDubboIDAndFraming(t *testing.T) {
	c, err := newConfiguredDubboCodec(streamURL("unit-stream"))
	require.NoError(t, err)
	var call base.Invocation = invocation.NewRPCInvocationWithOptions(invocation.WithArguments([]any{"hello"}), invocation.WithMethodName("Echo"), invocation.WithAttachments(map[string]any{"path": "codec.Echo", "interface": "codec.Echo"}))
	data, err := c.EncodeRequest(&remoting.Request{ID: 23, TwoWay: true, Data: &call})
	require.NoError(t, err)
	require.Equal(t, byte(29), data.Bytes()[2]&31)
	decoded, n, err := c.Decode(data.Bytes())
	require.NoError(t, err)
	require.Equal(t, data.Len(), n)
	require.True(t, decoded.IsRequest)
	require.Equal(t, byte(29), decoded.Result.(*remoting.Request).SerialID)
	oneway, err := c.EncodeRequest(&remoting.Request{ID: 27, Data: &call})
	require.NoError(t, err)
	onewayResult, _, err := c.Decode(oneway.Bytes())
	require.NoError(t, err)
	require.False(t, onewayResult.Result.(*remoting.Request).TwoWay)
	var reply string
	pending := remoting.NewPendingResponse(40)
	pending.Reply = &reply
	remoting.AddPendingResponse(pending)
	defer remoting.RemovePendingResponse(40)
	normal, err := c.EncodeResponse(&remoting.Response{ID: 40, SerialID: 29, Status: impl.Response_OK, Result: result.RPCResult{Rest: "response", Attrs: map[string]any{impl.DUBBO_VERSION_KEY: "2.0.2", "tag": "attached"}}})
	require.NoError(t, err)
	decodedResponse, _, err := c.Decode(normal.Bytes())
	require.NoError(t, err)
	require.Equal(t, "response", reply)
	require.Equal(t, "attached", decodedResponse.Result.(*remoting.Response).Result.(*result.RPCResult).Attrs["tag"])
	remoting.RemovePendingResponse(40)
	_, n, err = c.Decode(normal.Bytes())
	require.NoError(t, err)
	require.Equal(t, normal.Len(), n)
	for _, size := range []int{0, 7, 15} {
		value, consumed, decodeErr := c.Decode(data.Bytes()[:size])
		require.NoError(t, decodeErr)
		require.Nil(t, value)
		require.Zero(t, consumed)
	}
	value, n, err := c.Decode(data.Bytes()[:data.Len()-1])
	require.NoError(t, err)
	require.Nil(t, value)
	require.Equal(t, data.Len(), n)
	multi := append(bytes.Clone(data.Bytes()), data.Bytes()...)
	_, n, err = c.Decode(multi)
	require.NoError(t, err)
	require.Equal(t, data.Len(), n)
	for _, id := range []byte{0, 2, 30} {
		bad := bytes.Clone(data.Bytes())
		bad[2] = (bad[2] &^ 31) | id
		_, _, err = c.Decode(bad)
		require.Error(t, err, "ID %d", id)
	}
	bad := bytes.Clone(data.Bytes())
	bad[0] = 0
	_, _, err = c.Decode(bad)
	require.Error(t, err)
	event, err := c.EncodeRequest(&remoting.Request{ID: 24, Event: true})
	require.NoError(t, err)
	value, _, err = c.Decode(event.Bytes())
	require.NoError(t, err)
	require.True(t, value.Result.(*remoting.Request).Event)
	event, err = c.EncodeResponse(&remoting.Response{ID: 24, SerialID: 29, Event: true, Status: impl.Response_OK})
	require.NoError(t, err)
	value, _, err = c.Decode(event.Bytes())
	require.NoError(t, err)
	require.True(t, value.Result.(*remoting.Response).Event)
	failure, err := c.EncodeResponse(&remoting.Response{ID: 25, SerialID: 29, Status: impl.Response_SERVER_ERROR, Result: result.RPCResult{Err: errors.New("server failure")}})
	require.NoError(t, err)
	value, _, err = c.Decode(failure.Bytes())
	require.NoError(t, err)
	require.ErrorContains(t, value.Result.(*remoting.Response).Error, "server failure")
}
func TestDubboInitializationAndReceivePolicy(t *testing.T) {
	_, err := newConfiguredDubboCodec(streamURL("protobuf"))
	require.ErrorContains(t, err, "unsupported Dubbo")
	url := streamURL("unit-stream")
	url.SetParam("accepted-serializations", "")
	raw, err := newConfiguredDubboCodec(url)
	require.NoError(t, err)
	data, err := raw.EncodeRequest(&remoting.Request{ID: 1, Event: true})
	require.NoError(t, err)
	_, _, err = raw.Decode(data.Bytes())
	require.ErrorContains(t, err, "disabled serialization ID")
}

type codecBoundFraming struct {
	remoting.Codec
	closes int
}

func (c *codecBoundFraming) Close() error { c.closes++; return nil }
func TestCodecEndpointConflictAndBindingIsolation(t *testing.T) {
	const framing = "codec-endpoint-view"
	var instances []*codecBoundFraming
	require.NoError(t, remoting.RegisterCodec(remoting.CodecRegistration{Protocol: "dubbo", Name: framing, New: func(*common.URL) (remoting.Codec, error) {
		c := &codecBoundFraming{}
		instances = append(instances, c)
		return c, nil
	}}))
	scope := map[string]any{"prefix": "local"}
	listener := common.NewURLWithOptions(common.WithProtocol("dubbo"), common.WithLocation("127.0.0.1:1"), common.WithParamsValue("codec", framing), common.WithParamsValue("serialization", "hessian2"), common.WithParamsValue("accepted-serializations", "hessian2"), common.WithAttribute(extension.SerializationConfigAttribute, scope))
	dp := NewDubboProtocol()
	require.NoError(t, dp.PrepareCodec(listener, "provider"))
	dp.serverMap[listener.Location] = &remoting.ExchangeServer{URL: listener.Clone()}
	t.Cleanup(func() { _ = remoting.ReleaseCodec(listener) })
	for _, test := range []struct {
		name   string
		change func(*common.URL)
	}{
		{"codec", func(u *common.URL) { u.SetParam("codec", "different") }},
		{"receive-set", func(u *common.URL) { u.SetParam("accepted-serializations", "") }},
		{"configuration", func(u *common.URL) {
			u.SetAttribute(extension.SerializationConfigAttribute, map[string]any{"prefix": "different"})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			u := listener.Clone()
			u.DeleteAttribute(remoting.CodecAttribute)
			test.change(u)
			require.Error(t, dp.PrepareCodec(u, "provider"))
			require.Zero(t, instances[0].closes)
			bound, err := remoting.NewCodec(listener)
			require.NoError(t, err)
			require.Same(t, instances[0], bound)
		})
	}
	borrower := listener.Clone()
	borrower.DeleteAttribute(remoting.CodecAttribute)
	require.NoError(t, dp.PrepareCodec(borrower, "provider"))
	require.Len(t, instances, 1)
	require.Equal(t, exchangeCacheKey(listener), exchangeCacheKey(borrower))
	require.NoError(t, dp.ReleaseCodec(borrower, "provider"))
	require.Zero(t, instances[0].closes)
	independent := listener.Clone()
	independent.DeleteAttribute(remoting.CodecAttribute)
	require.NoError(t, remoting.PrepareCodec(independent))
	t.Cleanup(func() { _ = remoting.ReleaseCodec(independent) })
	require.NotEqual(t, exchangeCacheKey(listener), exchangeCacheKey(independent))
}

type codecLegacySerializer struct{ calls int }

func (c *codecLegacySerializer) Marshal(impl.DubboPackage) ([]byte, error) {
	c.calls++
	return []byte("legacy"), nil
}
func (c *codecLegacySerializer) Unmarshal([]byte, *impl.DubboPackage) error { c.calls++; return nil }
func TestCodecRuntimeViewAndLegacyOverride(t *testing.T) {
	previous, err := impl.GetSerializerById(2)
	require.NoError(t, err)
	t.Cleanup(func() { impl.SetSerializer("hessian2", previous) })
	view, err := newConfiguredDubboCodec(streamURL("hessian2"))
	require.NoError(t, err)
	legacy := &codecLegacySerializer{}
	impl.SetSerializer("hessian2", legacy)
	data, err := view.EncodeRequest(&remoting.Request{ID: 1, Event: true})
	require.NoError(t, err)
	require.NotContains(t, data.String(), "legacy")
	require.Zero(t, legacy.calls)
	later, err := newConfiguredDubboCodec(streamURL("hessian2"))
	require.NoError(t, err)
	data, err = later.EncodeRequest(&remoting.Request{ID: 2, Event: true})
	require.NoError(t, err)
	require.Contains(t, data.String(), "legacy")
	require.Equal(t, 1, legacy.calls)
}
func TestCodecConfiguredFrameBoundariesAndCorrelation(t *testing.T) {
	c, err := newConfiguredDubboCodec(streamURL("unit-stream"))
	require.NoError(t, err)
	one, err := c.EncodeRequest(&remoting.Request{ID: 7, Event: true})
	require.NoError(t, err)
	two, err := c.EncodeRequest(&remoting.Request{ID: 8, Event: true})
	require.NoError(t, err)
	frames := append(bytes.Clone(one.Bytes()), two.Bytes()...)
	for _, expected := range []struct {
		id     int64
		length int
	}{{7, one.Len()}, {8, two.Len()}} {
		decoded, n, decodeErr := c.Decode(frames)
		require.NoError(t, decodeErr)
		require.Equal(t, expected.length, n)
		require.Equal(t, expected.id, decoded.Result.(*remoting.Request).ID)
		frames = frames[n:]
	}
	require.Empty(t, frames)
	invalid := bytes.Clone(one.Bytes())
	invalid[1] = 0
	_, _, err = c.Decode(invalid)
	require.Error(t, err)
	id := remoting.SequenceID()
	reply := "unchanged"
	pending := remoting.NewPendingResponse(id)
	pending.Reply = &reply
	remoting.AddPendingResponse(pending)
	t.Cleanup(func() { remoting.RemovePendingResponse(remoting.SequenceType(id)) })
	response, err := c.EncodeResponse(&remoting.Response{ID: id, SerialID: 29, Status: impl.Response_OK, Result: result.RPCResult{Rest: "new"}})
	require.NoError(t, err)
	corrupted := append(bytes.Clone(response.Bytes()[:16]), byte(255))
	binary.BigEndian.PutUint32(corrupted[12:], 1)
	_, _, err = c.Decode(corrupted)
	var decode *remoting.ResponseDecodeError
	require.ErrorAs(t, err, &decode)
	require.Equal(t, id, decode.RequestID)
	require.Equal(t, rpcerror.Codec, rpcerror.Of(err))
	require.Equal(t, "unchanged", reply)
	remoting.RemovePendingResponse(remoting.SequenceType(id))
	_, n, err := c.Decode(response.Bytes())
	require.NoError(t, err)
	require.Equal(t, response.Len(), n)
	require.Equal(t, "unchanged", reply)
	large := bytes.Clone(one.Bytes()[:16])
	const size = 16 << 20
	binary.BigEndian.PutUint32(large[12:], size)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	decoded, n, err := c.Decode(large)
	runtime.ReadMemStats(&after)
	require.NoError(t, err)
	require.Nil(t, decoded)
	require.Equal(t, size+16, n)
	require.Less(t, after.TotalAlloc-before.TotalAlloc, uint64(1<<20), "declared length must not reserve its body")
}
