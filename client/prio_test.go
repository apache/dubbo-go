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

package client_test

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	dubbo "dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/available"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/triple"
	"dubbo.apache.org/dubbo-go/v3/server"
)

const (
	prioTestInstanceGroup   = "new-config-instance-group"
	prioTestInstanceVersion = "new-config-instance-version"
	prioTestServerGroup     = "new-config-server-group"
	prioTestServerVersion   = "new-config-server-version"
	prioTestClientGroup     = "new-config-client-group"
	prioTestClientVersion   = "new-config-client-version"
	prioTestServiceName     = "com.example.NewConfigAPIPriorityService"
	prioTestServiceMethod   = "Ping"
)

type prioRefSnapshot struct {
	Group    string
	Version  string
	Protocol string
}

// TestPrio_ServerOverride verifies that
// server-level options override instance defaults, and the override does not
// leak into later server creations.
func TestPrio_ServerOverride(t *testing.T) {
	ins, err := dubbo.NewInstance(
		dubbo.WithName("new-config-api-prio-server"),
		dubbo.WithGroup(prioTestInstanceGroup),
		dubbo.WithVersion(prioTestInstanceVersion),
	)
	require.NoError(t, err)

	srvDefault, err := ins.NewServer()
	require.NoError(t, err)
	defaultServiceOpts := registerPrioService(t, srvDefault)
	assert.Equal(t, prioTestInstanceGroup, defaultServiceOpts.Service.Group)
	assert.Equal(t, prioTestInstanceVersion, defaultServiceOpts.Service.Version)

	srvOverride, err := ins.NewServer(
		server.WithServerGroup(prioTestServerGroup),
		server.WithServerVersion(prioTestServerVersion),
	)
	require.NoError(t, err)
	overrideServiceOpts := registerPrioService(t, srvOverride)
	assert.Equal(t, prioTestServerGroup, overrideServiceOpts.Service.Group)
	assert.Equal(t, prioTestServerVersion, overrideServiceOpts.Service.Version)

	srvVerify, err := ins.NewServer()
	require.NoError(t, err)
	verifyServiceOpts := registerPrioService(t, srvVerify)
	assert.Equal(t, prioTestInstanceGroup, verifyServiceOpts.Service.Group)
	assert.Equal(t, prioTestInstanceVersion, verifyServiceOpts.Service.Version)
}

// TestPrio_ClientOverride verifies that
// client-level options override instance defaults, and the override does not
// leak into later client creations.
func TestPrio_ClientOverride(t *testing.T) {
	ins, err := dubbo.NewInstance(
		dubbo.WithName("new-config-api-prio-client"),
		dubbo.WithGroup(prioTestInstanceGroup),
		dubbo.WithVersion(prioTestInstanceVersion),
	)
	require.NoError(t, err)

	cliDefault, err := ins.NewClient()
	require.NoError(t, err)

	defaultSnapshot := capturePrioRef(t, cliDefault)
	assert.Equal(t, prioTestInstanceGroup, defaultSnapshot.Group)
	assert.Equal(t, prioTestInstanceVersion, defaultSnapshot.Version)
	assert.Equal(t, constant.TriProtocol, defaultSnapshot.Protocol)

	cliOverride, err := ins.NewClient(
		client.WithClientGroup(prioTestClientGroup),
		client.WithClientVersion(prioTestClientVersion),
		client.WithClientProtocolTriple(),
	)
	require.NoError(t, err)

	overrideSnapshot := capturePrioRef(t, cliOverride)
	assert.Equal(t, prioTestClientGroup, overrideSnapshot.Group)
	assert.Equal(t, prioTestClientVersion, overrideSnapshot.Version)
	assert.Equal(t, constant.TriProtocol, overrideSnapshot.Protocol)

	cliVerify, err := ins.NewClient()
	require.NoError(t, err)

	verifySnapshot := capturePrioRef(t, cliVerify)
	assert.Equal(t, prioTestInstanceGroup, verifySnapshot.Group)
	assert.Equal(t, prioTestInstanceVersion, verifySnapshot.Version)
	assert.Equal(t, constant.TriProtocol, verifySnapshot.Protocol)
}

// capturePrioRef captures the effective reference
// values after client option initialization.
func capturePrioRef(t *testing.T, cli *client.Client) prioRefSnapshot {
	t.Helper()

	snapshot := prioRefSnapshot{}
	var refOpts *client.ReferenceOptions

	_, err := cli.DialWithInfo(
		prioTestServiceName,
		&client.ClientInfo{
			InterfaceName: prioTestServiceName,
			MethodNames:   []string{prioTestServiceMethod},
		},
		client.WithClusterAvailable(),
		client.WithProtocolTriple(),
		client.WithURL("tri://127.0.0.1:1"),
		func(opts *client.ReferenceOptions) {
			refOpts = opts
		},
	)
	require.NoError(t, err)
	require.NotNil(t, refOpts)

	snapshot.Group = refOpts.Reference.Group
	snapshot.Version = refOpts.Reference.Version
	snapshot.Protocol = refOpts.Reference.Protocol

	return snapshot
}

type prioService struct{}

func (s *prioService) Reference() string {
	return prioTestServiceName
}

func (s *prioService) Ping(context.Context, string) (string, error) {
	return "ok", nil
}

// registerPrioService registers a minimal non-IDL service and
// returns its resolved service options for assertions.
func registerPrioService(t *testing.T, srv *server.Server) *server.ServiceOptions {
	t.Helper()

	svc := &prioService{}
	err := srv.RegisterService(
		svc,
		server.WithInterface(prioTestServiceName),
		server.WithNotRegister(),
	)
	require.NoError(t, err)

	svcOpts := srv.GetServiceOptionsByInterfaceName(prioTestServiceName)
	require.NotNil(t, svcOpts)
	require.NotNil(t, svcOpts.Application)
	require.NotNil(t, svcOpts.Provider)
	require.NotNil(t, svcOpts.Service)
	assert.Equal(t, prioTestServiceName, svcOpts.Id)
	assert.Equal(t, prioTestServiceName, svcOpts.Service.Interface)
	assert.True(t, svcOpts.Service.NotRegister)

	svcOptsByID := srv.GetServiceOptions(prioTestServiceName)
	require.NotNil(t, svcOptsByID)
	assert.Same(t, svcOpts, svcOptsByID)

	return svcOpts
}
