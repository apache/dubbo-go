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
	"context"
	"net"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/available"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	_ "dubbo.apache.org/dubbo-go/v3/filter/echo"
	_ "dubbo.apache.org/dubbo-go/v3/filter/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/triple"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/server"
)

const (
	newConfigAPIServiceName = "com.example.NewConfigAPIService"
	newConfigAPIHelloBody   = "hello-new-config-api"
)

type NewConfigAPIService struct{}

func (s *NewConfigAPIService) Reference() string {
	return newConfigAPIServiceName
}

func (s *NewConfigAPIService) SayHello(context.Context, *emptypb.Empty) (*wrapperspb.StringValue, error) {
	return wrapperspb.String(newConfigAPIHelloBody), nil
}

func TestNewConfigAPI_InstanceNewServerNewClientCallUnary(t *testing.T) {
	port := freePortForNewConfigAPITest(t)

	ins, err := NewInstance(
		WithName("new-config-api-itest"),
		WithProtocol(
			protocol.WithTriple(),
			protocol.WithIp("127.0.0.1"),
			protocol.WithPort(port),
		),
	)
	require.NoError(t, err)

	srv, err := ins.NewServer()
	require.NoError(t, err)

	svc := &NewConfigAPIService{}
	svcInfo := &common.ServiceInfo{
		InterfaceName: newConfigAPIServiceName,
		ServiceType:   svc,
		Methods: []common.MethodInfo{
			{
				Name: "SayHello",
				Type: constant.CallUnary,
				ReqInitFunc: func() any {
					return &emptypb.Empty{}
				},
				MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
					req := args[0].(*emptypb.Empty)
					res, callErr := handler.(*NewConfigAPIService).SayHello(ctx, req)
					if callErr != nil {
						return nil, callErr
					}
					return tri.NewResponse(res), nil
				},
			},
		},
	}

	err = srv.Register(
		svc,
		svcInfo,
		server.WithInterface(newConfigAPIServiceName),
		server.WithNotRegister(),
		server.WithFilter("echo"),
	)
	require.NoError(t, err)

	svcOpts := srv.GetServiceOptions(svc.Reference())
	require.NotNil(t, svcOpts)
	require.NoError(t, svcOpts.Export())

	t.Cleanup(func() {
		svcOpts.Unexport()
		extension.GetProtocol(constant.TriProtocol).Destroy()
	})

	cli, err := ins.NewClient()
	require.NoError(t, err)

	conn, err := cli.DialWithInfo(
		newConfigAPIServiceName,
		&client.ClientInfo{
			InterfaceName: newConfigAPIServiceName,
			MethodNames:   []string{"SayHello"},
		},
		client.WithClusterAvailable(),
		client.WithProtocolTriple(),
		client.WithURL("tri://127.0.0.1:"+strconv.Itoa(port)),
	)
	require.NoError(t, err)

	var callErr error
	resp := new(wrapperspb.StringValue)
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		resp = new(wrapperspb.StringValue)
		callErr = conn.CallUnary(context.Background(), []any{&emptypb.Empty{}}, resp, "SayHello")
		if callErr == nil && resp.Value == newConfigAPIHelloBody {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.NoError(t, callErr)
	assert.Equal(t, newConfigAPIHelloBody, resp.Value)
}

func freePortForNewConfigAPITest(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = listener.Close()
	}()

	return listener.Addr().(*net.TCPAddr).Port
}
