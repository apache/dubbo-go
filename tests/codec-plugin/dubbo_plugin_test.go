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
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/failover"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	_ "dubbo.apache.org/dubbo-go/v3/filter/accesslog"
	_ "dubbo.apache.org/dubbo-go/v3/filter/active"
	_ "dubbo.apache.org/dubbo-go/v3/filter/echo"
	_ "dubbo.apache.org/dubbo-go/v3/filter/exec_limit"
	_ "dubbo.apache.org/dubbo-go/v3/filter/generic"
	_ "dubbo.apache.org/dubbo-go/v3/filter/graceful_shutdown"
	_ "dubbo.apache.org/dubbo-go/v3/filter/token"
	_ "dubbo.apache.org/dubbo-go/v3/filter/tps"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/serialization"
	hessianstream "dubbo.apache.org/dubbo-go/v3/serialization/hessian2"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// A wrapping remoting plugin uses only the public SPI and delegates framing.
type countedRemoting struct{ remoting.Codec }

func (c countedRemoting) EncodeRequest(r *remoting.Request) (*bytes.Buffer, error) {
	count("remoting/request").Add(1)
	b, err := c.Codec.EncodeRequest(r)
	if err == nil && b.Len() >= 16 {
		count(fmt.Sprintf("wire-id/%d", b.Bytes()[2]&31)).Add(1)
	}
	return b, err
}
func (c countedRemoting) EncodeResponse(r *remoting.Response) (*bytes.Buffer, error) {
	count("remoting/response").Add(1)
	return c.Codec.EncodeResponse(r)
}
func (c countedRemoting) Decode(b []byte) (*remoting.DecodeResult, int, error) {
	v, n, e := c.Codec.Decode(b)
	if v != nil {
		count("remoting/decode").Add(1)
	}
	return v, n, e
}

type externalStream struct {
	hessianstream.Codec
	side string
}

func (*externalStream) Name() string { return "test-stream" }
func init() {
	extension.MustRegisterCodec(remoting.CodecRegistration{Protocol: "dubbo", Name: "test-framing", New: func(url *common.URL) (remoting.Codec, error) {
		url.SetParam("codec", "dubbo")
		c, e := remoting.NewCodec(url)
		return countedRemoting{Codec: c}, e
	}})
	id := byte(30)
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "dubbo", Name: "test-stream", Mode: serialization.StreamMode, DubboID: &id, New: func(o serialization.Options) (serialization.Codec, error) { return &externalStream{side: o.Side}, nil }})
}
func (*Echo) Fail(context.Context, string) (string, error) { return "", errors.New("business failure") }
func TestDubboPluginRPC(t *testing.T) {
	defer extension.GetProtocol("dubbo").Destroy()
	port := freePort(t)
	srv, e := server.NewServer(server.WithServerProtocol(protocol.WithDubbo(), protocol.WithCodec("test-framing"), protocol.WithIp("127.0.0.1"), protocol.WithPort(port)))
	if e != nil {
		t.Fatal(e)
	}
	if e = srv.RegisterService(&Echo{}, server.WithSerialization("test-stream"), server.WithNotRegister(), server.WithGroup(strconv.Itoa(port))); e != nil {
		t.Fatal(e)
	}
	export(t, srv)
	ready(t, port)
	cli, e := client.NewClient(client.WithClientURL("dubbo://127.0.0.1:"+strconv.Itoa(port)), client.WithClientProtocolDubbo(), client.WithClientProtocol(protocol.WithDubbo(), protocol.WithCodec("test-framing")))
	if e != nil {
		t.Fatal(e)
	}
	conn, e := cli.Dial("codec.plugin.Echo", client.WithSerialization("test-stream"), client.WithGroup(strconv.Itoa(port)))
	if e != nil {
		t.Fatal(e)
	}
	var reply string
	if e = conn.CallUnary(context.Background(), []any{"world"}, &reply, "Greet"); e != nil {
		t.Fatal(e)
	}
	if reply != "hello world" {
		t.Fatalf("wrong reply %q", reply)
	}
	if e = conn.CallUnary(context.Background(), []any{"world"}, &reply, "Fail"); e == nil || !strings.Contains(e.Error(), "business failure") {
		t.Fatalf("wrong business error: %v", e)
	}
	for _, side := range []string{"consumer", "provider"} {
		for _, step := range []string{"encode", "decode"} {
			if n := count("stream/" + step + "/" + side).Load(); n < 2 {
				steps.Range(func(k, v any) bool { t.Logf("count %s=%d", k, v.(*atomic.Int64).Load()); return true })
				t.Fatalf("stream %s %s calls=%d", side, step, n)
			}
		}
	}
	for _, step := range []string{"request", "response"} {
		if n := count("remoting/" + step).Load(); n < 2 {
			t.Fatalf("remoting %s count %d", step, n)
		}
	}
	if n := count("remoting/decode").Load(); n < 4 {
		t.Fatalf("remoting decode count %d", n)
	}
	if count("wire-id/30").Load() == 0 {
		t.Fatal("custom stream ID was not sent")
	}
	other, e := client.NewClient(client.WithClientURL("dubbo://127.0.0.1:"+strconv.Itoa(port)), client.WithClientProtocolDubbo(), client.WithClientProtocol(protocol.WithDubbo(), protocol.WithCodec("test-framing")))
	if e != nil {
		t.Fatal(e)
	}
	second, e := other.Dial("codec.plugin.Echo", client.WithSerialization("hessian2"), client.WithGroup(strconv.Itoa(port)), client.WithRetries(0))
	if e != nil {
		t.Fatal(e)
	}
	if e = second.CallUnary(context.Background(), []any{"world"}, &reply, "Greet"); e != nil {
		t.Fatal(e)
	}
	if count("wire-id/2").Load() == 0 {
		t.Fatal("same-address client reused the incompatible stream binding")
	}
	conflict, e := server.NewServer(server.WithServerProtocol(protocol.WithDubbo(), protocol.WithCodec("test-framing"), protocol.WithIp("127.0.0.1"), protocol.WithPort(port)))
	if e != nil {
		t.Fatal(e)
	}
	if e = conflict.RegisterService(&Echo{}, server.WithSerialization("hessian2"), server.WithGroup(strconv.Itoa(port)+"-other"), server.WithNotRegister()); e != nil {
		t.Fatal(e)
	}
	e = conflict.GetServiceOptionsByInterfaceName("codec.plugin.Echo").Export()
	if e == nil || !strings.Contains(e.Error(), "different") {
		t.Fatalf("expected shared-endpoint configuration conflict, got %v", e)
	}

}

type taggedStreamEncoder struct{ serialization.Encoder }

func (e taggedStreamEncoder) Finish() ([]byte, error) {
	b, err := e.Encoder.Finish()
	return append([]byte("STREAM:"), b...), err
}
func (c *externalStream) NewEncoder() (serialization.Encoder, error) {
	count("stream/encode/" + c.side).Add(1)
	e, err := c.Codec.NewEncoder()
	return taggedStreamEncoder{e}, err
}
func (c *externalStream) NewDecoder(b []byte) (serialization.Decoder, error) {
	if !bytes.HasPrefix(b, []byte("STREAM:")) {
		return nil, fmt.Errorf("missing stream prefix")
	}
	count("stream/decode/" + c.side).Add(1)
	return c.Codec.NewDecoder(b[7:])
}

func (c countedRemoting) Close() error {
	if closer, ok := c.Codec.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
