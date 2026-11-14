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

package codecplugin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/getty"
	"dubbo.apache.org/dubbo-go/v3/serialization"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// No Hessian code is used by this external plugin.
type pureJSONStream struct {
	side         string
	made, closed *atomic.Int64
}

func (*pureJSONStream) Name() string                    { return "pure-json-stream" }
func (*pureJSONStream) Marshal(v any) ([]byte, error)   { return json.Marshal(v) }
func (*pureJSONStream) Unmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
func (c *pureJSONStream) NewEncoder() (serialization.Encoder, error) {
	count("pure-json/encode/" + c.side).Add(1)
	return &pureJSONEncoder{}, nil
}
func (c *pureJSONStream) NewDecoder(b []byte) (serialization.Decoder, error) {
	count("pure-json/decode/" + c.side).Add(1)
	return pureJSONDecoder{json.NewDecoder(bytes.NewReader(b))}, nil
}
func (c *pureJSONStream) Close() error { c.closed.Add(1); return nil }

type pureJSONEncoder struct{ bytes.Buffer }

func (e *pureJSONEncoder) Encode(v any) error      { return json.NewEncoder(&e.Buffer).Encode(v) }
func (e *pureJSONEncoder) Finish() ([]byte, error) { return bytes.Clone(e.Bytes()), nil }
func (*pureJSONEncoder) Close() error              { return nil }

type pureJSONDecoder struct{ *json.Decoder }

func (pureJSONDecoder) Close() error { return nil }

type SecondEcho struct{ Echo }

func (*SecondEcho) Reference() string { return "codec.plugin.SecondEcho" }
func (*SecondEcho) Greet(_ context.Context, value string) (string, error) {
	return "second " + value, nil
}

func (*Echo) Number(_ context.Context, n int32) (int32, error)             { return n + 1, nil }
func (*Echo) Wide(_ context.Context, n int64) (int64, error)               { return n, nil }
func (*Echo) Numbers(_ context.Context, n []int32) ([]int32, error)        { return n, nil }
func (*Echo) Record(_ context.Context, n *JSONRecord) (*JSONRecord, error) { return n, nil }

type JSONRecord struct {
	Count int32
	Label string
}

func TestPureJSONStreamSharedEndpoint(t *testing.T) {
	var providerMade, providerClosed, consumerMade, consumerClosed atomic.Int64
	id := byte(31)
	if err := extension.RegisterSerialization(serialization.Registration{Protocol: "dubbo", Name: "pure-json-stream", Mode: serialization.StreamMode, DubboID: &id, New: func(o serialization.Options) (serialization.Codec, error) {
		made, closed := &consumerMade, &consumerClosed
		if o.Side == "provider" {
			made, closed = &providerMade, &providerClosed
		}
		made.Add(1)
		return &pureJSONStream{side: o.Side, made: made, closed: closed}, nil
	}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		extension.GetProtocol("dubbo").Destroy()
		if providerMade.Load() != 1 || providerClosed.Load() != 1 {
			t.Errorf("listener instances: created=%d closed=%d", providerMade.Load(), providerClosed.Load())
		}
		if consumerMade.Load() != consumerClosed.Load() {
			t.Errorf("consumer instances: created=%d closed=%d", consumerMade.Load(), consumerClosed.Load())
		}
	})
	port := freePort(t)
	group := strconv.Itoa(port)
	srv, err := server.NewServer(server.WithServerProtocol(protocol.WithDubbo(), protocol.WithIp("127.0.0.1"), protocol.WithPort(port)))
	if err != nil {
		t.Fatal(err)
	}
	for _, service := range []struct {
		impl any
		name string
	}{{&Echo{}, "codec.plugin.Echo"}, {&SecondEcho{}, "codec.plugin.SecondEcho"}} {
		if err = srv.RegisterService(service.impl, server.WithSerialization("pure-json-stream"), server.WithGroup(group), server.WithNotRegister()); err != nil {
			t.Fatal(err)
		}
		if err = srv.GetServiceOptionsByInterfaceName(service.name).Export(); err != nil {
			t.Fatal(err)
		}
	}
	ready(t, port)
	if providerMade.Load() != 1 {
		t.Fatalf("shared endpoint constructed %d codecs", providerMade.Load())
	}
	cli, err := client.NewClient(client.WithClientURL("dubbo://127.0.0.1:"+group), client.WithClientProtocolDubbo())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"codec.plugin.Echo", "codec.plugin.SecondEcho"} {
		conn, err := cli.Dial(name, client.WithSerialization("pure-json-stream"), client.WithGroup(group), client.WithRetries(0))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var reply string
		err = conn.CallUnary(ctx, []any{"world"}, &reply, "Greet")
		want := "hello world"
		if name == "codec.plugin.SecondEcho" {
			want = "second world"
		}
		if err != nil || reply != want {
			t.Fatalf("%s reply=%q error=%v", name, reply, err)
		}
		if name == "codec.plugin.SecondEcho" {
			cancel()
			continue
		}

		var number int32
		if err = conn.CallUnary(ctx, []any{int32(41)}, &number, "Number"); err != nil || number != 42 {
			t.Fatalf("typed int32 reply=%d error=%v", number, err)
		}
		var wide int64
		if err = conn.CallUnary(ctx, []any{int64(9007199254740993)}, &wide, "Wide"); err != nil || wide != 9007199254740993 {
			t.Fatalf("typed int64 reply=%d error=%v", wide, err)
		}
		var numbers []int32
		if err = conn.CallUnary(ctx, []any{[]int32{1, 2, 3}}, &numbers, "Numbers"); err != nil || !reflect.DeepEqual(numbers, []int32{1, 2, 3}) {
			t.Fatalf("typed slice reply=%v error=%v", numbers, err)
		}
		record := &JSONRecord{Count: 42, Label: "typed"}
		var decodedRecord JSONRecord
		if err = conn.CallUnary(ctx, []any{record}, &decodedRecord, "Record"); err != nil || decodedRecord != *record {
			t.Fatalf("typed struct reply=%v error=%v", decodedRecord, err)
		}
		err = conn.CallUnary(ctx, []any{"world"}, &reply, "Fail")
		cancel()
		if err == nil || !strings.Contains(err.Error(), "business failure") {
			t.Fatalf("%s exception=%v", name, err)
		}
	}
	for _, side := range []string{"consumer", "provider"} {
		for _, step := range []string{"encode", "decode"} {
			if n := count("pure-json/" + step + "/" + side).Load(); n < 7 {
				t.Errorf("%s %s operations=%d", side, step, n)
			}
		}
	}
}

type leaseJSONStream struct{ pureJSONStream }

func (*leaseJSONStream) Name() string { return "lease-json-stream" }

func TestDubboDiscoveredProviderCodecLeases(t *testing.T) {
	var made, closed atomic.Int64
	id := byte(27)
	if err := extension.RegisterSerialization(serialization.Registration{Protocol: "dubbo", Name: "lease-json-stream", Mode: serialization.StreamMode, DubboID: &id, New: func(o serialization.Options) (serialization.Codec, error) {
		closes := &atomic.Int64{}
		if o.Side == "consumer" {
			made.Add(1)
			closes = &closed
		}
		return &leaseJSONStream{pureJSONStream{side: o.Side, closed: closes}}, nil
	}}); err != nil {
		t.Fatal(err)
	}
	providers := make([]*common.URL, 2)
	for i := range providers {
		port := freePort(t)
		url, err := common.NewURL("dubbo://127.0.0.1:"+strconv.Itoa(port)+"/lease.Echo", common.WithParamsValue("serialization", "lease-json-stream"), common.WithParamsValue("side", "provider"))
		if err != nil {
			t.Fatal(err)
		}
		transport, err := getty.NewServerWithError(url, func(call *invocation.RPCInvocation) result.RPCResult {
			return result.RPCResult{Rest: "hello " + call.Arguments()[0].(string)}
		})
		if err != nil {
			t.Fatal(err)
		}
		transport.Start()
		t.Cleanup(transport.Stop)
		ready(t, port)
		providers[i], err = common.NewURL("dubbo://127.0.0.1:" + strconv.Itoa(port) + "/lease.Echo")
		if err != nil {
			t.Fatal(err)
		}
	}
	ref := common.NewURLWithOptions(common.WithProtocol("dubbo"), common.WithPath("lease.Echo"), common.WithParamsValue("serialization", "lease-json-stream"), common.WithParamsValue("side", "consumer"))
	if err := extension.PrepareCodecURL(ref, "consumer"); err != nil {
		t.Fatal(err)
	}
	urls := []*common.URL{providers[0].MergeURL(ref), providers[1].MergeURL(ref)}
	connect := func(url *common.URL) *remoting.ExchangeClient {
		t.Helper()
		c := remoting.NewExchangeClient(url, getty.NewClient(getty.Options{ConnectTimeout: time.Second, RequestTimeout: time.Second}), time.Second, false)
		if c == nil {
			t.Fatal("connect failed")
		}
		t.Cleanup(c.Close)
		return c
	}
	a, b := connect(urls[0]), connect(urls[1])
	call := func(c *remoting.ExchangeClient, url *common.URL) {
		t.Helper()
		var reply string
		var inv base.Invocation = invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("Echo"), invocation.WithArguments([]any{"world"}), invocation.WithReply(&reply), invocation.WithAttachments(map[string]any{"path": "lease.Echo", "interface": "lease.Echo"}))
		var response result.RPCResult
		if err := c.Request(&inv, url, time.Second, &response); err != nil || response.Err != nil || reply != "hello world" {
			t.Fatalf("reply=%q error=%v resultError=%v", reply, err, response.Err)
		}
	}
	call(a, urls[0])
	call(b, urls[1])
	a.Close()
	if closed.Load() != 0 {
		t.Fatalf("sibling close released active codec: %d", closed.Load())
	}
	call(b, urls[1])
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	badURL, err := common.NewURL("dubbo://" + listener.Addr().String() + "/lease.Echo")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	failed := getty.NewClient(getty.Options{ConnectTimeout: 30 * time.Millisecond})
	if err = failed.Connect(badURL.MergeURL(ref)); err == nil {
		t.Fatal("expected failed connection")
	}
	failed.Close()
	if closed.Load() != 0 {
		t.Fatalf("failed setup closed active codec: %d", closed.Load())
	}
	call(b, urls[1])
	b.Close()
	if made.Load() != 1 || closed.Load() != 1 {
		t.Fatalf("first generation made=%d closed=%d", made.Load(), closed.Load())
	}
	urls[1].SetParam("serialization", "hessian2")
	c := connect(urls[1])
	call(c, urls[1])
	c.Close()
	if made.Load() != 2 || closed.Load() != 2 {
		t.Fatalf("recovery made=%d closed=%d", made.Load(), closed.Load())
	}
}
