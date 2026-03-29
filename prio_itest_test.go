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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/available"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/triple"
	"dubbo.apache.org/dubbo-go/v3/server"
)

const (
	newConfigAPIPriorityInstanceGroup   = "new-config-instance-group"
	newConfigAPIPriorityInstanceVersion = "new-config-instance-version"
	newConfigAPIPriorityServerGroup     = "new-config-server-group"
	newConfigAPIPriorityServerVersion   = "new-config-server-version"
	newConfigAPIPriorityClientGroup     = "new-config-client-group"
	newConfigAPIPriorityClientVersion   = "new-config-client-version"
	newConfigAPIPriorityServiceName     = "com.example.NewConfigAPIPriorityService"
	newConfigAPIPriorityServiceMethod   = "Ping"
)

type newConfigAPIReferenceSnapshot struct {
	Group    string
	Version  string
	Protocol string
}

func TestNewConfigAPI_Priority_ServerOverridesInstanceDefaults(t *testing.T) {
	ins, err := NewInstance(
		WithName("new-config-api-prio-server"),
		WithGroup(newConfigAPIPriorityInstanceGroup),
		WithVersion(newConfigAPIPriorityInstanceVersion),
	)
	require.NoError(t, err)

	srvDefault, err := ins.NewServer()
	require.NoError(t, err)
	defaultServiceOpts := registerNewConfigAPIPriorityService(t, srvDefault)
	assert.Equal(t, newConfigAPIPriorityInstanceGroup, defaultServiceOpts.Service.Group)
	assert.Equal(t, newConfigAPIPriorityInstanceVersion, defaultServiceOpts.Service.Version)

	srvOverride, err := ins.NewServer(
		server.WithServerGroup(newConfigAPIPriorityServerGroup),
		server.WithServerVersion(newConfigAPIPriorityServerVersion),
	)
	require.NoError(t, err)
	overrideServiceOpts := registerNewConfigAPIPriorityService(t, srvOverride)
	assert.Equal(t, newConfigAPIPriorityServerGroup, overrideServiceOpts.Service.Group)
	assert.Equal(t, newConfigAPIPriorityServerVersion, overrideServiceOpts.Service.Version)

	srvVerify, err := ins.NewServer()
	require.NoError(t, err)
	verifyServiceOpts := registerNewConfigAPIPriorityService(t, srvVerify)
	assert.Equal(t, newConfigAPIPriorityInstanceGroup, verifyServiceOpts.Service.Group)
	assert.Equal(t, newConfigAPIPriorityInstanceVersion, verifyServiceOpts.Service.Version)
}

func TestNewConfigAPI_Priority_ClientOverridesInstanceDefaults(t *testing.T) {
	ins, err := NewInstance(
		WithName("new-config-api-prio-client"),
		WithGroup(newConfigAPIPriorityInstanceGroup),
		WithVersion(newConfigAPIPriorityInstanceVersion),
	)
	require.NoError(t, err)

	cliDefault, err := ins.NewClient()
	require.NoError(t, err)

	defaultSnapshot := captureNewConfigAPIEffectiveReference(t, cliDefault)
	assert.Equal(t, newConfigAPIPriorityInstanceGroup, defaultSnapshot.Group)
	assert.Equal(t, newConfigAPIPriorityInstanceVersion, defaultSnapshot.Version)
	assert.Equal(t, constant.TriProtocol, defaultSnapshot.Protocol)

	cliOverride, err := ins.NewClient(
		client.WithClientGroup(newConfigAPIPriorityClientGroup),
		client.WithClientVersion(newConfigAPIPriorityClientVersion),
		client.WithClientProtocolTriple(),
	)
	require.NoError(t, err)

	overrideSnapshot := captureNewConfigAPIEffectiveReference(t, cliOverride)
	assert.Equal(t, newConfigAPIPriorityClientGroup, overrideSnapshot.Group)
	assert.Equal(t, newConfigAPIPriorityClientVersion, overrideSnapshot.Version)
	assert.Equal(t, constant.TriProtocol, overrideSnapshot.Protocol)

	cliVerify, err := ins.NewClient()
	require.NoError(t, err)

	verifySnapshot := captureNewConfigAPIEffectiveReference(t, cliVerify)
	assert.Equal(t, newConfigAPIPriorityInstanceGroup, verifySnapshot.Group)
	assert.Equal(t, newConfigAPIPriorityInstanceVersion, verifySnapshot.Version)
	assert.Equal(t, constant.TriProtocol, verifySnapshot.Protocol)
}

func captureNewConfigAPIEffectiveReference(t *testing.T, cli *client.Client) newConfigAPIReferenceSnapshot {
	t.Helper()

	snapshot := newConfigAPIReferenceSnapshot{}
	var refOpts *client.ReferenceOptions

	_, err := cli.DialWithInfo(
		newConfigAPIPriorityServiceName,
		&client.ClientInfo{
			InterfaceName: newConfigAPIPriorityServiceName,
			MethodNames:   []string{newConfigAPIPriorityServiceMethod},
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

type newConfigAPIPriorityService struct{}

func (s *newConfigAPIPriorityService) Reference() string {
	return newConfigAPIPriorityServiceName
}

func (s *newConfigAPIPriorityService) Ping(context.Context, string) (string, error) {
	return "ok", nil
}

func registerNewConfigAPIPriorityService(t *testing.T, srv *server.Server) *server.ServiceOptions {
	t.Helper()

	svc := &newConfigAPIPriorityService{}
	err := srv.RegisterService(
		svc,
		server.WithInterface(newConfigAPIPriorityServiceName),
		server.WithNotRegister(),
	)
	require.NoError(t, err)

	svcOpts := srv.GetServiceOptionsByInterfaceName(newConfigAPIPriorityServiceName)
	require.NotNil(t, svcOpts)

	return svcOpts
}
