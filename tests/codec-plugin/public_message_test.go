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
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/serialization"
	"dubbo.apache.org/dubbo-go/v3/server"
)

type publicMessageCodec struct {
	side  string
	steps *[4]atomic.Int64
}

func (*publicMessageCodec) Name() string { return "test-public-message" }
func (c *publicMessageCodec) Marshal(v any) ([]byte, error) {
	i := 0
	if c.side == "provider" {
		i = 2
	}
	c.steps[i].Add(1)
	return json.Marshal(v)
}
func (c *publicMessageCodec) Unmarshal(b []byte, v any) error {
	i := 3
	if c.side == "provider" {
		i = 1
	}
	c.steps[i].Add(1)
	return json.Unmarshal(b, v)
}

type PublicMessageService struct{}

func (*PublicMessageService) Greet(_ context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	return wrapperspb.String("hello " + req.Value), nil
}
func TestPublicDialUsesMessagePlugin(t *testing.T) {
	defer extension.GetProtocol("tri").Destroy()
	var calls [4]atomic.Int64
	var made atomic.Int64
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "triple", Name: "test-public-message", WireName: "test-public-message", Mode: serialization.MessageMode, New: func(o serialization.Options) (serialization.Codec, error) {
		made.Add(1)
		return &publicMessageCodec{side: o.Side, steps: &calls}, nil
	}})
	for _, wire := range []string{"grpc", "triple"} {
		port := freePort(t)
		srv, err := server.NewServer(server.WithServerProtocol(protocol.WithTriple(), protocol.WithIp("127.0.0.1"), protocol.WithPort(port)))
		if err != nil {
			t.Fatal(err)
		}
		info := &common.ServiceInfo{InterfaceName: "codec.PublicMessage", ServiceType: (*PublicMessageService)(nil), Methods: []common.MethodInfo{{Name: "Greet", Type: constant.CallUnary, ReqInitFunc: func() any { return &wrapperspb.StringValue{} }, MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
			reply, err := handler.(*PublicMessageService).Greet(ctx, args[0].(*wrapperspb.StringValue))
			return tri.NewResponse(reply), err
		}}}}
		if err = srv.Register(&PublicMessageService{}, info, server.WithSerialization("test-public-message"), server.WithNotRegister(), server.WithGroup(wire), server.WithFilter("-default")); err != nil {
			t.Fatal(err)
		}
		if err = srv.GetServiceOptionsByInterfaceName(info.InterfaceName).Export(); err != nil {
			t.Fatal(err)
		}
		ready(t, port)
		cli, err := client.NewClient(client.WithClientURL("tri://127.0.0.1:"+strconv.Itoa(port)), client.WithClientProtocol(protocol.WithTriple(triple.WithWireProtocol(wire))))
		if err != nil {
			t.Fatal(err)
		}
		for _, withInfo := range []bool{false, true} {
			var conn *client.Connection
			options := []client.ReferenceOption{client.WithSerialization("test-public-message"), client.WithGroup(wire), client.WithFilter("-default")}
			if withInfo {
				conn, err = cli.DialWithInfo(info.InterfaceName, &client.ClientInfo{InterfaceName: info.InterfaceName}, options...)
			} else {
				conn, err = cli.Dial(info.InterfaceName, options...)
			}
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			reply := &wrapperspb.StringValue{}
			err = conn.CallUnary(ctx, []any{wrapperspb.String("world")}, reply, "Greet")
			cancel()
			if err != nil || reply.Value != "hello world" {
				t.Fatalf("wire=%s info=%t reply=%v err=%v", wire, withInfo, reply, err)
			}
		}
		before := made.Load()
		if _, err = cli.DialWithService(info.InterfaceName, &Echo{}, client.WithSerialization("test-public-message")); err == nil || !strings.Contains(err.Error(), "legacy generated clients") {
			t.Fatalf("legacy initialization: %v", err)
		}
		if made.Load() != before {
			t.Fatal("unsupported legacy client created a plugin")
		}
	}
	for i := range calls {
		if n := calls[i].Load(); n != 4 {
			t.Fatalf("step %d got %d calls, want 4", i, n)
		}
	}
}
