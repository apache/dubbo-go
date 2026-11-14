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
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	"dubbo.apache.org/dubbo-go/v3/serialization"
	"dubbo.apache.org/dubbo-go/v3/server"
)

type HandlerContextEcho struct {
	calls         atomic.Int64
	entered, gate chan struct{}
}

func (*HandlerContextEcho) Reference() string { return "codec.handler.Context" }
func (s *HandlerContextEcho) Echo(_ context.Context, value string) (string, error) {
	s.calls.Add(1)
	if value == "blocked" && s.entered != nil {
		close(s.entered)
		<-s.gate
	}
	return "hello " + value, nil
}
func handlerContextServer(t *testing.T, port int, group, version, tag string, svc *HandlerContextEcho, options ...server.ServiceOption) *server.ServiceOptions {
	t.Helper()
	srv, err := server.NewServer(server.WithServerProtocol(protocol.WithTriple(), protocol.WithIp("127.0.0.1"), protocol.WithPort(port)), server.WithExtension(tagOption(tag)))
	if err != nil {
		t.Fatal(err)
	}
	opts := []server.ServiceOption{server.WithSerialization("test-tagged-json"), server.WithGroup(group), server.WithVersion(version), server.WithNotRegister(), server.WithFilter("-default")}
	opts = append(opts, options...)
	if err = srv.RegisterService(svc, opts...); err != nil {
		t.Fatal(err)
	}
	result := srv.GetServiceOptionsByInterfaceName(svc.Reference())
	if err = result.Export(); err != nil {
		t.Fatal(err)
	}
	return result
}
func handlerContextClient(t *testing.T, port int, group, version, tag, wire string) *client.Connection {
	t.Helper()
	cli, err := client.NewClient(client.WithClientURL("tri://127.0.0.1:"+strconv.Itoa(port)), client.WithClientProtocol(protocol.WithTriple(triple.WithWireProtocol(wire))), client.WithExtension(tagOption(tag)))
	if err != nil {
		t.Fatal(err)
	}
	conn, err := cli.Dial("codec.handler.Context", client.WithIDL(constant.NONIDL), client.WithSerialization("test-tagged-json"), client.WithGroup(group), client.WithVersion(version), client.WithRetries(0), client.WithFilter("-default"))
	if err != nil {
		t.Fatal(err)
	}
	return conn
}
func handlerContextCall(conn *client.Connection, value string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var reply string
	err := conn.CallUnary(ctx, []any{value}, &reply, "Echo")
	return reply, err
}
func TestCodecSharedTripleHandlerContexts(t *testing.T) {
	defer extension.GetProtocol("tri").Destroy()
	for _, wire := range []string{"grpc", "triple"} {
		t.Run(wire, func(t *testing.T) {
			port := freePort(t)
			group := "handler-" + strconv.Itoa(port)
			a, b, c := &HandlerContextEcho{}, &HandlerContextEcho{}, &HandlerContextEcho{}
			handlerContextServer(t, port, group, "a", wire+"-A:", a)
			ready(t, port)
			clientA := handlerContextClient(t, port, group, "a", wire+"-A:", wire)
			if reply, err := handlerContextCall(clientA, "a"); err != nil || reply != "hello a" {
				t.Fatalf("A reply=%q error=%v", reply, err)
			}
			// Identical procedure and group, distinct version: codec configuration differs.
			handlerContextServer(t, port, group, "b", wire+"-B:", b)
			clientB := handlerContextClient(t, port, group, "b", wire+"-B:", wire)
			if reply, err := handlerContextCall(clientB, "b"); err != nil || reply != "hello b" {
				t.Fatalf("B reply=%q error=%v", reply, err)
			}
			// A second group with the same prefix deliberately removes format mismatch
			// as an explanation: an exact empty policy must reject the request itself.
			handlerContextServer(t, port, group+"-disabled", "a", wire+"-A:", c, server.WithAcceptedSerializations())
			denied := handlerContextClient(t, port, group+"-disabled", "a", wire+"-A:", wire)
			if _, err := handlerContextCall(denied, "denied"); err == nil || !strings.Contains(err.Error(), "disabled") {
				t.Fatalf("disabled group error=%v", err)
			}
			if c.calls.Load() != 0 {
				t.Fatal("disabled group executed business code")
			}
			if reply, err := handlerContextCall(clientA, "again"); err != nil || reply != "hello again" {
				t.Fatalf("A after B reply=%q error=%v", reply, err)
			}
			if a.calls.Load() != 2 || b.calls.Load() != 1 {
				t.Fatalf("wrong routing: A=%d B=%d", a.calls.Load(), b.calls.Load())
			}
		})
	}
}

type handlerLaterFailure struct{ err error }

func (p *handlerLaterFailure) PrepareCodec(*common.URL, string) error { return p.err }
func (*handlerLaterFailure) Export(base.Invoker) base.Exporter        { return nil }
func (*handlerLaterFailure) Refer(*common.URL) base.Invoker           { return nil }
func (*handlerLaterFailure) Destroy()                                 {}
func TestCodecTripleFailedExportRetryRPC(t *testing.T) {
	defer extension.GetProtocol("tri").Destroy()
	const later = "codec-handler-later-failure"
	failure := errors.New("later protocol failed")
	extension.SetProtocol(later, func() base.Protocol { return &handlerLaterFailure{err: failure} })
	defer extension.UnregisterProtocol(later)
	port := freePort(t)
	group := "retry-" + strconv.Itoa(port)
	tag := group + ":"
	srv, err := server.NewServer(server.SetServerProtocols(map[string]*global.ProtocolConfig{"tri": {Name: "tri", Ip: "127.0.0.1", Port: strconv.Itoa(port), TripleConfig: global.DefaultTripleConfig()}, later: {Name: later, Ip: "127.0.0.1", Port: "1"}}), server.WithExtension(tagOption(tag)))
	if err != nil {
		t.Fatal(err)
	}
	svc := &HandlerContextEcho{}
	if err = srv.RegisterService(svc, server.WithSerialization("test-tagged-json"), server.WithGroup(group), server.WithProtocolIDs([]string{"tri", later}), server.WithNotRegister(), server.WithFilter("-default")); err != nil {
		t.Fatal(err)
	}
	opts := srv.GetServiceOptionsByInterfaceName(svc.Reference())
	if err = opts.Export(); !errors.Is(err, failure) {
		t.Fatalf("first export error=%v", err)
	}
	if opts.IsExport() || count("close/"+tag+"/provider").Load() != 1 {
		t.Fatal("failed export did not roll back")
	}
	opts.Service.ProtocolIDs = []string{"tri"}
	if err = opts.Export(); err != nil {
		t.Fatal(err)
	}
	ready(t, port)
	conn := handlerContextClient(t, port, group, "", tag, "grpc")
	if reply, err := handlerContextCall(conn, "retry"); err != nil || reply != "hello retry" {
		t.Fatalf("successful Export must permit RPC: reply=%q error=%v", reply, err)
	}
	if svc.calls.Load() != 1 {
		t.Fatal("retry did not execute business method")
	}
	extension.GetProtocol("tri").Destroy()
	if n := count("close/" + tag + "/provider").Load(); n != 2 {
		t.Fatalf("two generations must each close once: %d", n)
	}
}
func TestCodecClusterAssemblyFailureReleasesLease(t *testing.T) {
	defer extension.GetProtocol("tri").Destroy()
	const name = "codec-cluster-assembly-lease"
	var made, closed atomic.Int64
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "triple", Name: name, Mode: serialization.WrappedObjectMode, WireName: "proto", New: func(serialization.Options) (serialization.Codec, error) {
		made.Add(1)
		return &clusterLeaseCodec{name: name, closed: &closed}, nil
	}})
	for _, providers := range []int{1, 2} {
		t.Run(fmt.Sprint(providers), func(t *testing.T) {
			addresses := make([]string, providers)
			for i := range addresses {
				addresses[i] = "tri://127.0.0.1:" + strconv.Itoa(i+1)
			}
			cli, err := client.NewClient(client.WithClientURL(strings.Join(addresses, ";")))
			if err != nil {
				t.Fatal(err)
			}
			before := made.Load()
			conn, err := cli.Dial("codec.ClusterLease", client.WithIDL(constant.NONIDL), client.WithSerialization(name), client.WithCluster("codec-missing-cluster"), client.WithFilter("-default"))
			if err == nil || conn != nil {
				t.Fatalf("Dial should fail: conn=%v error=%v", conn, err)
			}
			if made.Load()-before != int64(providers) {
				t.Fatalf("expected %d prepared codecs", providers)
			}
			if made.Load() != closed.Load() {
				t.Fatalf("Dial leaked leases: made=%d closed=%d", made.Load(), closed.Load())
			}
		})
	}
}

type clusterLeaseCodec struct {
	name   string
	closed *atomic.Int64
}

func (c *clusterLeaseCodec) Name() string              { return c.name }
func (*clusterLeaseCodec) Marshal(any) ([]byte, error) { return []byte("x"), nil }
func (*clusterLeaseCodec) Unmarshal([]byte, any) error { return nil }
func (c *clusterLeaseCodec) Close() error              { c.closed.Add(1); return nil }
