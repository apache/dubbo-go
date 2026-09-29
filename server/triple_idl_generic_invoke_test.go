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
	"strconv"
	"testing"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

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
	_ "dubbo.apache.org/dubbo-go/v3/filter/generic"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/triple"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	dubboserver "dubbo.apache.org/dubbo-go/v3/server"
)

const (
	idlGenericServiceName = "com.example.IDLGenericGreeter"
	idlGenericHelloBody   = "hello from an IDL service"
)

// IDLGenericGreeter stands in for a protoc-generated provider: its method takes
// and returns protobuf messages, and it is exported together with a ServiceInfo.
type IDLGenericGreeter struct{}

func (s *IDLGenericGreeter) SayHello(_ context.Context, _ *emptypb.Empty) (*wrapperspb.StringValue, error) {
	return wrapperspb.String(idlGenericHelloBody), nil
}

func (s *IDLGenericGreeter) Reference() string {
	return idlGenericServiceName
}

// TestIDLServiceServesGenericInvocation covers a caller without generated stubs.
// The provider is exported in IDL mode, so only its declared methods are carried
// by the ServiceInfo; a generic call has to reach it through the $invoke
// procedure that the framework registers for that export mode.
func TestIDLServiceServesGenericInvocation(t *testing.T) {
	port := freePortForNewConfigAPITest(t)

	ins, err := dubbo.NewInstance(
		dubbo.WithName("idl-generic-invocation"),
		dubbo.WithProtocol(
			protocol.WithTriple(),
			protocol.WithIp("127.0.0.1"),
			protocol.WithPort(port),
		),
	)
	require.NoError(t, err)

	srv, err := ins.NewServer()
	require.NoError(t, err)

	svc := &IDLGenericGreeter{}
	svcInfo := &common.ServiceInfo{
		InterfaceName: idlGenericServiceName,
		ServiceType:   svc,
		Methods: []common.MethodInfo{
			{
				Name: "SayHello",
				Type: constant.CallUnary,
				ReqInitFunc: func() any {
					return &emptypb.Empty{}
				},
				MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
					return handler.(*IDLGenericGreeter).SayHello(ctx, args[0].(*emptypb.Empty))
				},
			},
		},
	}

	require.NoError(t, srv.Register(
		svc,
		svcInfo,
		dubboserver.WithInterface(idlGenericServiceName),
		dubboserver.WithNotRegister(),
		dubboserver.WithFilter("generic_service"),
		// Generic calls carry a TripleRequestWrapper whose inner serialization is
		// hessian2, so the provider has to declare it for the wrapper to be decoded.
		dubboserver.WithSerialization(constant.Hessian2Serialization),
	))

	svcOpts := srv.GetServiceOptions(svc.Reference())
	require.NotNil(t, svcOpts)
	require.NoError(t, svcOpts.Export())
	t.Cleanup(func() {
		svcOpts.Unexport()
		extension.GetProtocol(constant.TriProtocol).Destroy()
	})

	cli, err := ins.NewClient()
	require.NoError(t, err)
	genericService, err := cli.NewGenericService(
		idlGenericServiceName,
		client.WithProtocolTriple(),
		client.WithURL("tri://127.0.0.1:"+strconv.Itoa(port)),
		client.WithClusterAvailable(),
		client.WithGenericType(constant.GenericSerializationProtobufJson),
	)
	require.NoError(t, err)

	var (
		result   any
		callErr  error
		attempts int
		deadline = time.Now().Add(10 * time.Second)
	)
	for time.Now().Before(deadline) {
		attempts++
		attemptCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		result, callErr = genericService.Invoke(
			attemptCtx,
			"SayHello",
			[]string{"google.protobuf.Empty"},
			[]hessian.Object{"{}"},
		)
		cancel()
		if callErr == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.NoErrorf(t, callErr, "generic invocation failed after %d attempts", attempts)
	require.IsType(t, "", result)
	assert.JSONEq(t, strconv.Quote(idlGenericHelloBody), result.(string))
}
