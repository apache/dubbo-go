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
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/failover"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
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
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/serialization"
	hessianstream "dubbo.apache.org/dubbo-go/v3/serialization/hessian2"
	"dubbo.apache.org/dubbo-go/v3/server"
)

var steps sync.Map

func count(key string) *atomic.Int64 {
	v, _ := steps.LoadOrStore(key, &atomic.Int64{})
	return v.(*atomic.Int64)
}

type pluginConfig struct {
	Tag string `yaml:"tag"`
}

func (*pluginConfig) Prefix() string                       { return "tagged" }
func (*pluginConfig) New() extension.Config                { return &pluginConfig{Tag: "CODEC:"} }
func (c *pluginConfig) Init(extension.Scope) error         { return nil }
func (*pluginConfig) FilterNames(extension.Scope) []string { return nil }
func (c *pluginConfig) CodecConfig() any                   { return c.Tag }

type tagOption string

func (tagOption) Prefix() string                   { return "tagged" }
func (o tagOption) Apply(c extension.Config) error { c.(*pluginConfig).Tag = string(o); return nil }

type taggedJSON struct {
	side, tag string
	closed    atomic.Bool
}

func (*taggedJSON) Name() string { return "test-tagged-json" }
func (c *taggedJSON) Marshal(v any) ([]byte, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("codec closed during RPC")
	}
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	index := 0
	if c.side == "provider" {
		index = 2
	}
	count(fmt.Sprintf("%s/%d", c.tag, index)).Add(1)
	return append([]byte(c.tag), b...), nil
}
func (c *taggedJSON) Unmarshal(b []byte, v any) error {
	if c.closed.Load() {
		return fmt.Errorf("codec closed during RPC")
	}
	if !bytes.HasPrefix(b, []byte(c.tag)) {
		return fmt.Errorf("tagged JSON prefix missing")
	}
	index := 3
	if c.side == "provider" {
		index = 1
	}
	count(fmt.Sprintf("%s/%d", c.tag, index)).Add(1)
	return json.Unmarshal(b[len(c.tag):], v)
}
func init() {
	extension.MustRegisterConfig(&pluginConfig{})
	extension.MustRegisterSerialization(serialization.Registration{
		Protocol: "triple", Name: "test-tagged-json", Mode: serialization.WrappedObjectMode, WireName: "proto", ConfigPrefix: "tagged",
		New: func(o serialization.Options) (serialization.Codec, error) {
			tag := "CODEC:"
			if o.Config != nil {
				tag = o.Config.(string)
			}
			return &taggedJSON{side: o.Side, tag: tag}, nil
		},
	})
}

type Echo struct{}

func (*Echo) Reference() string                                    { return "codec.plugin.Echo" }
func (*Echo) Greet(_ context.Context, name string) (string, error) { return "hello " + name, nil }

func freePort(t *testing.T) int {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
func ready(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		c, e := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 50*time.Millisecond)
		if e == nil {
			c.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatal(e)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
func export(t *testing.T, srv *server.Server) {
	t.Helper()
	if err := srv.GetServiceOptionsByInterfaceName("codec.plugin.Echo").Export(); err != nil {
		t.Fatal(err)
	}
}
func TestHighLevelPluginRPC(t *testing.T) {
	t.Cleanup(func() {
		for _, wire := range []string{"grpc", "triple"} {
			for _, tag := range []string{"ALPHA:", "BETA:"} {
				for _, side := range []string{"consumer", "provider"} {
					if n := count("close/" + wire + tag + "/" + side).Load(); n != 1 {
						t.Errorf("close %s %s %s count=%d", wire, tag, side, n)
					}
				}
			}
		}
	})

	defer extension.GetProtocol(constant.TriProtocol).Destroy()
	const requests = 12
	var wg sync.WaitGroup
	for _, wire := range []string{"grpc", "triple"} {
		for _, tag := range []string{"ALPHA:", "BETA:"} {
			tag = wire + tag
			port := freePort(t)
			srv, err := server.NewServer(server.WithServerProtocol(protocol.WithTriple(), protocol.WithIp("127.0.0.1"), protocol.WithPort(port)), server.WithExtension(tagOption(tag)))
			if err != nil {
				t.Fatal(err)
			}
			if err = srv.RegisterService(&Echo{}, server.WithSerialization("test-tagged-json"), server.WithNotRegister(), server.WithGroup(strconv.Itoa(port))); err != nil {
				t.Fatal(err)
			}
			export(t, srv)
			ready(t, port)
			cli, err := client.NewClient(client.WithClientURL("tri://127.0.0.1:"+strconv.Itoa(port)), client.WithClientProtocol(protocol.WithTriple(triple.WithWireProtocol(wire))), client.WithExtension(tagOption(tag)))
			if err != nil {
				t.Fatal(err)
			}
			conn, err := cli.Dial("codec.plugin.Echo", client.WithIDL(constant.NONIDL), client.WithSerialization("test-tagged-json"), client.WithGroup(strconv.Itoa(port)))
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < requests; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					var reply string
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := conn.CallUnary(ctx, []any{"world"}, &reply, "Greet"); err != nil {
						t.Error(err)
					} else if reply != "hello world" {
						t.Errorf("wrong reply %q", reply)
					}
				}()
			}
		}
	}
	wg.Wait()
	for _, wire := range []string{"grpc", "triple"} {
		for _, tag := range []string{"ALPHA:", "BETA:"} {
			for i := 0; i < 4; i++ {
				if n := count(fmt.Sprintf("%s%s/%d", wire, tag, i)).Load(); n != requests {
					t.Errorf("%s %s step %d calls=%d want=%d", wire, tag, i, n, requests)
				}
			}
		}
	}
}

func TestUnknownAndDisabledPlugin(t *testing.T) {
	cli, err := client.NewClient(client.WithClientURL("tri://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cli.Dial("codec.plugin.Echo", client.WithSerialization("not-imported")); err == nil || !strings.Contains(err.Error(), "unsupported serialization") {
		t.Fatalf("unknown codec error: %v", err)
	}
	defer extension.GetProtocol(constant.TriProtocol).Destroy()
	for _, wire := range []string{"grpc", "triple"} {
		port := freePort(t)
		srv, e := server.NewServer(server.WithServerProtocol(protocol.WithTriple(), protocol.WithIp("127.0.0.1"), protocol.WithPort(port)))
		if e != nil {
			t.Fatal(e)
		}
		if e = srv.RegisterService(&Echo{}, server.WithSerialization("test-tagged-json"), server.WithAcceptedSerializations("hessian2"), server.WithNotRegister(), server.WithGroup(strconv.Itoa(port))); e != nil {
			t.Fatal(e)
		}
		export(t, srv)
		ready(t, port)
		cli, e := client.NewClient(client.WithClientURL("tri://127.0.0.1:"+strconv.Itoa(port)), client.WithClientProtocol(protocol.WithTriple(triple.WithWireProtocol(wire))))
		if e != nil {
			t.Fatal(e)
		}
		conn, e := cli.Dial("codec.plugin.Echo", client.WithIDL(constant.NONIDL), client.WithSerialization("test-tagged-json"), client.WithGroup(strconv.Itoa(port)))
		if e != nil {
			t.Fatal(e)
		}
		var reply string
		e = conn.CallUnary(context.Background(), []any{"world"}, &reply, "Greet")
		if e == nil || !strings.Contains(e.Error(), "disabled") {
			t.Fatalf("%s disabled codec error: %v", wire, e)
		}
	}
}

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

func TestBuiltinsAndCorruptPayload(t *testing.T) {
	defer extension.GetProtocol(constant.TriProtocol).Destroy()
	defer extension.GetProtocol("dubbo").Destroy()
	for _, format := range []string{"hessian2", "msgpack", "default-dubbo", "corrupt"} {
		for _, wire := range []string{"grpc", "triple"} {
			if format == "default-dubbo" && wire == "triple" {
				continue
			}
			t.Run(format+"/"+wire, func(t *testing.T) {
				port := freePort(t)
				scheme := "tri"
				serverProtocol := protocol.WithTriple()
				if format == "default-dubbo" {
					scheme = "dubbo"
					serverProtocol = protocol.WithDubbo()
				}
				serverOpts := []server.ServerOption{server.WithServerProtocol(serverProtocol, protocol.WithIp("127.0.0.1"), protocol.WithPort(port))}
				if format == "corrupt" {
					serverOpts = append(serverOpts, server.WithExtension(tagOption("RIGHT:")))
				}
				srv, err := server.NewServer(serverOpts...)
				if err != nil {
					t.Fatal(err)
				}
				serviceOpts := []server.ServiceOption{server.WithNotRegister(), server.WithGroup(strconv.Itoa(port))}
				if format == "corrupt" {
					serviceOpts = append(serviceOpts, server.WithSerialization("test-tagged-json"))
				} else if format != "default-dubbo" {
					serviceOpts = append(serviceOpts, server.WithSerialization(format))
				}
				if err = srv.RegisterService(&Echo{}, serviceOpts...); err != nil {
					t.Fatal(err)
				}
				export(t, srv)
				ready(t, port)
				clientOpts := []client.ClientOption{client.WithClientURL(scheme + "://127.0.0.1:" + strconv.Itoa(port)), client.WithClientProtocol(protocol.WithTriple(triple.WithWireProtocol(wire)))}
				if scheme == "dubbo" {
					clientOpts = append(clientOpts, client.WithClientProtocolDubbo())
				}
				if format == "corrupt" {
					clientOpts = append(clientOpts, client.WithExtension(tagOption("WRONG:")))
				}
				cli, err := client.NewClient(clientOpts...)
				if err != nil {
					t.Fatal(err)
				}
				refOpts := []client.ReferenceOption{client.WithGroup(strconv.Itoa(port)), client.WithIDL(constant.NONIDL), client.WithRetries(0)}
				if format == "corrupt" {
					refOpts = append(refOpts, client.WithSerialization("test-tagged-json"))
				} else if format != "default-dubbo" {
					refOpts = append(refOpts, client.WithSerialization(format))
				}
				conn, err := cli.Dial("codec.plugin.Echo", refOpts...)
				if err != nil {
					t.Fatal(err)
				}
				var reply string
				err = conn.CallUnary(context.Background(), []any{"world"}, &reply, "Greet")
				if format == "corrupt" {
					if err == nil || !strings.Contains(err.Error(), "prefix missing") {
						t.Fatalf("corrupt codec error=%v", err)
					}
				} else if err != nil || reply != "hello world" {
					t.Fatalf("builtin reply=%q error=%v", reply, err)
				}
			})
		}
	}
}

func (c countedRemoting) Close() error {
	if closer, ok := c.Codec.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (c *taggedJSON) Close() error {
	c.closed.Store(true)
	count("close/" + c.tag + "/" + c.side).Add(1)
	return nil
}
