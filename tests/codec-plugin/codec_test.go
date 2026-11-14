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
	"fmt"
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
	_ "dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/serialization"
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

func (c *taggedJSON) Close() error {
	c.closed.Store(true)
	count("close/" + c.tag + "/" + c.side).Add(1)
	return nil
}
