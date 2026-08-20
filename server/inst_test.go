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

package server_test

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
	dubbo "dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/available"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	_ "dubbo.apache.org/dubbo-go/v3/filter/echo"
	_ "dubbo.apache.org/dubbo-go/v3/filter/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	dubboserver "dubbo.apache.org/dubbo-go/v3/server"
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

func (s *NewConfigAPIService) Echo(_ context.Context, req *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error) {
	return wrapperspb.Bytes(req.Value), nil
}

type newConfigAPITestSetup struct {
	ins  *dubbo.Instance
	srv  *dubboserver.Server
	port int
}

// TestCfgAPI_Create verifies instance construction and
// object creation for NewServer/NewClient.
func TestCfgAPI_Create(t *testing.T) {
	setup := setupNewConfigAPITest(t)

	cli, err := setup.ins.NewClient()
	require.NoError(t, err)
	require.NotNil(t, cli)
}

// TestCfgAPI_Export verifies service registration and
// export path only, isolated from client invocation.
func TestCfgAPI_Export(t *testing.T) {
	setup := setupNewConfigAPITest(t)

	svcOpts := registerAndExportNewConfigAPIService(t, setup.srv)
	require.NotNil(t, svcOpts)

	urls := svcOpts.GetExportedUrls()
	require.Len(t, urls, 1)
	assert.Empty(t, urls[0].GetParam(constant.MaxServerSendMsgSize, ""))
	assert.Equal(t, "4mib", urls[0].GetParam(constant.MaxServerRecvMsgSize, ""))
}

// TestCfgAPI_Call verifies the end-to-end
// invocation path: NewInstance -> NewServer/NewClient -> unary call.
func TestCfgAPI_Call(t *testing.T) {
	setup := setupNewConfigAPITest(t)
	_ = registerAndExportNewConfigAPIService(t, setup.srv)

	conn := dialNewConfigAPIConnection(t, setup.ins, setup.port)
	assertNewConfigAPIUnaryHello(t, conn)
}

func TestCfgAPITripleMessageSizePropagation(t *testing.T) {
	setup := setupNewConfigAPITest(
		t,
		triple.WithMaxServerSendMsgSize("8kib"),
		triple.WithMaxServerRecvMsgSize("1kib"),
	)
	svcOpts := registerAndExportNewConfigAPIService(t, setup.srv)

	urls := svcOpts.GetExportedUrls()
	require.Len(t, urls, 1)
	assert.Equal(t, "8kib", urls[0].GetParam(constant.MaxServerSendMsgSize, ""))
	assert.Equal(t, "1kib", urls[0].GetParam(constant.MaxServerRecvMsgSize, ""))

	conn := dialNewConfigAPIConnection(t, setup.ins, setup.port)
	assertNewConfigAPIUnaryHello(t, conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	smallResponse := new(wrapperspb.BytesValue)
	err := conn.CallUnary(ctx, []any{wrapperspb.Bytes(make([]byte, 512))}, smallResponse, "Echo")
	require.NoError(t, err)
	assert.Len(t, smallResponse.Value, 512)

	largeResponse := new(wrapperspb.BytesValue)
	err = conn.CallUnary(ctx, []any{wrapperspb.Bytes(make([]byte, 4*1024))}, largeResponse, "Echo")
	require.Error(t, err)
}

func TestCfgAPIExportedMessageSizeCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		protocol *global.ProtocolConfig
		wantSend string
		wantRecv string
	}{
		{
			name: "deprecated values fill empty nested values",
			protocol: &global.ProtocolConfig{
				MaxServerSendMsgSize: "2mib",
				MaxServerRecvMsgSize: "3mib",
				TripleConfig:         &global.TripleConfig{},
			},
			wantSend: "2mib",
			wantRecv: "3mib",
		},
		{
			name: "nested values take precedence",
			protocol: &global.ProtocolConfig{
				MaxServerSendMsgSize: "2mib",
				MaxServerRecvMsgSize: "3mib",
				TripleConfig: &global.TripleConfig{
					MaxServerSendMsgSize: "5mib",
					MaxServerRecvMsgSize: "6mib",
				},
			},
			wantSend: "5mib",
			wantRecv: "6mib",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := freePortForNewConfigAPITest(t)
			tt.protocol.Name = constant.TriProtocol
			tt.protocol.Ip = "127.0.0.1"
			tt.protocol.Port = strconv.Itoa(port)

			ins, err := dubbo.NewInstance(
				dubbo.WithName("message-size-compatibility"),
				func(opts *dubbo.InstanceOptions) {
					opts.Protocols = map[string]*global.ProtocolConfig{
						constant.TriProtocol: tt.protocol,
					}
				},
			)
			require.NoError(t, err)

			srv, err := ins.NewServer()
			require.NoError(t, err)
			svcOpts := registerAndExportNewConfigAPIService(t, srv)

			urls := svcOpts.GetExportedUrls()
			require.Len(t, urls, 1)
			assert.Equal(t, tt.wantSend, urls[0].GetParam(constant.MaxServerSendMsgSize, ""))
			assert.Equal(t, tt.wantRecv, urls[0].GetParam(constant.MaxServerRecvMsgSize, ""))
		})
	}
}

func TestDeprecatedProtocolConfigMessageSizeFieldsCompileForExternalConsumers(t *testing.T) {
	config := global.ProtocolConfig{
		MaxServerSendMsgSize: "2mib",
		MaxServerRecvMsgSize: "3mib",
	}

	assert.Equal(t, "2mib", config.MaxServerSendMsgSize)
	assert.Equal(t, "3mib", config.MaxServerRecvMsgSize)
}

func setupNewConfigAPITest(t *testing.T, tripleOpts ...triple.Option) *newConfigAPITestSetup {
	t.Helper()

	port := freePortForNewConfigAPITest(t)

	ins, err := dubbo.NewInstance(
		dubbo.WithName("new-config-api-integration"),
		dubbo.WithProtocol(
			protocol.WithTriple(tripleOpts...),
			protocol.WithIp("127.0.0.1"),
			protocol.WithPort(port),
		),
	)
	require.NoError(t, err)

	srv, err := ins.NewServer()
	require.NoError(t, err)

	return &newConfigAPITestSetup{
		ins:  ins,
		srv:  srv,
		port: port,
	}
}

func registerAndExportNewConfigAPIService(t *testing.T, srv *dubboserver.Server) *dubboserver.ServiceOptions {
	t.Helper()

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
			{
				Name: "Echo",
				Type: constant.CallUnary,
				ReqInitFunc: func() any {
					return &wrapperspb.BytesValue{}
				},
				MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
					req := args[0].(*wrapperspb.BytesValue)
					res, callErr := handler.(*NewConfigAPIService).Echo(ctx, req)
					if callErr != nil {
						return nil, callErr
					}
					return tri.NewResponse(res), nil
				},
			},
		},
	}

	err := srv.Register(
		svc,
		svcInfo,
		dubboserver.WithInterface(newConfigAPIServiceName),
		dubboserver.WithNotRegister(),
		dubboserver.WithFilter("echo"),
	)
	require.NoError(t, err)

	svcOpts := srv.GetServiceOptions(svc.Reference())
	require.NotNil(t, svcOpts)
	require.NoError(t, svcOpts.Export())

	t.Cleanup(func() {
		svcOpts.Unexport()
		extension.GetProtocol(constant.TriProtocol).Destroy()
	})

	return svcOpts
}

func dialNewConfigAPIConnection(t *testing.T, ins *dubbo.Instance, port int) *client.Connection {
	t.Helper()

	cli, err := ins.NewClient()
	require.NoError(t, err)

	conn, err := cli.DialWithInfo(
		newConfigAPIServiceName,
		&client.ClientInfo{
			InterfaceName: newConfigAPIServiceName,
			MethodNames:   []string{"SayHello", "Echo"},
		},
		client.WithClusterAvailable(),
		client.WithProtocolTriple(),
		client.WithURL("tri://127.0.0.1:"+strconv.Itoa(port)),
	)
	require.NoError(t, err)

	return conn
}

func assertNewConfigAPIUnaryHello(t *testing.T, conn *client.Connection) {
	t.Helper()

	var callErr error
	resp := new(wrapperspb.StringValue)
	deadline := time.Now().Add(5 * time.Second)
	const attemptTimeout = 500 * time.Millisecond
	attempt := 0

	for time.Now().Before(deadline) {
		attempt++
		timeout := attemptTimeout
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
		if timeout <= 0 {
			break
		}

		attemptCtx, cancel := context.WithTimeout(context.Background(), timeout)
		resp = new(wrapperspb.StringValue)
		callErr = conn.CallUnary(attemptCtx, []any{&emptypb.Empty{}}, resp, "SayHello")
		cancel()
		if callErr == nil && resp.Value == newConfigAPIHelloBody {
			break
		}
		if callErr != nil {
			t.Logf("attempt %d failed: call unary error=%v, timeout=%s, remaining=%s", attempt, callErr, timeout, time.Until(deadline))
		} else {
			t.Logf("attempt %d got unexpected response: value=%q, expect=%q, timeout=%s, remaining=%s", attempt, resp.Value, newConfigAPIHelloBody, timeout, time.Until(deadline))
		}
		time.Sleep(50 * time.Millisecond)
	}

	if callErr != nil {
		t.Logf("final failure after %d attempts: last error=%v, last response=%q", attempt, callErr, resp.Value)
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
