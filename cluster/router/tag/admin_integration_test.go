//go:build adminintegration

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

package tag

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/go-zookeeper/zk"

	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/zookeeper"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// Run only against disposable services: this test publishes and deletes one
// application rule. Ordinary package tests do not need an external service.
func TestAdminTagConfigCenterRPC(t *testing.T) {
	for _, backend := range []string{"nacos", "zookeeper"} {
		t.Run(backend, func(t *testing.T) {
			endpoint := os.Getenv("DUBBO_TAG_" + strings.ToUpper(backend))
			if endpoint == "" {
				t.Skip("set DUBBO_TAG_" + strings.ToUpper(backend) + " to a disposable endpoint")
			}
			app := fmt.Sprintf("tag-compat-%s-%d", backend, time.Now().UnixNano())
			key := app + constant.TagRouterRuleSuffix
			invokers := []base.Invoker{startAdminTagProvider(t, app), startAdminTagProvider(t, app)}
			consumer, err := common.NewURL("consumer://127.0.0.1/org.example.TagIdentity?application=tag-compat-consumer")
			require.NoError(t, err)
			fixture := func(name string) string {
				payload := strings.ReplaceAll(adminTagFixture(t, name), adminTagProvider, app)
				payload = strings.ReplaceAll(payload, "127.0.0.1:20890", invokers[0].GetURL().Location)
				return strings.ReplaceAll(payload, "127.0.0.1:20891", invokers[1].GetURL().Location)
			}
			dc, publish, remove := adminTagRealConfiguration(t, backend, endpoint, key)
			publish(fixture("enabled"))
			t.Cleanup(remove)
			require.Eventually(t, func() bool { got, e := dc.GetRule(key); return e == nil && got == fixture("enabled") }, 15*time.Second, 50*time.Millisecond)
			env := conf.GetEnvInstance()
			previous := env.GetDynamicConfiguration()
			env.SetDynamicConfiguration(dc)
			t.Cleanup(func() { env.SetDynamicConfiguration(previous) })
			p, err := NewTagPriorityRouter()
			require.NoError(t, err)
			p.Notify(invokers)
			t.Cleanup(func() { dc.RemoveListener(key, p) })
			check := func(tag string, force bool, want ...base.Invoker) {
				t.Helper()
				require.Eventually(t, func() bool {
					got := p.Route(invokers, consumer, invocation.NewRPCInvocation("Identity", nil, map[string]any{constant.Tagkey: tag, constant.ForceUseTag: strconv.FormatBool(force)}))
					if len(got) != len(want) {
						return false
					}
					for i := range got {
						if got[i] != want[i] {
							return false
						}
					}
					return true
				}, 15*time.Second, 50*time.Millisecond, "tag=%q force=%v", tag, force)
				assertAdminTagSelection(t, p, invokers, consumer, tag, force, want...)
				for _, provider := range want {
					require.Equal(t, provider.GetURL().Location, invokeAdminTagProvider(t, provider))
				}
				t.Logf("tag=%q force=%v providers=%v", tag, force, adminTagAddresses(want))
			}
			check("gray", false, invokers[0])
			check("", false, invokers[1])
			check("unknown", false, invokers...)
			check("unknown", true)
			publish(fixture("updated"))
			check("gray", false, invokers[1])
			check("", false, invokers[0])
			// The pinned Admin disabled payload omits false and remains enabled. Keep
			// this observation separate from the SDK's explicit-false contract below.
			publish(fixture("legacy-disabled-omitted"))
			check("gray", false, invokers[1])
			t.Log("pinned Admin enabled=false omitted: rule remains active")
			publish(fixture("fixed-disabled"))
			check("gray", false, invokers...)
			publish(fixture("force"))
			check("gray", false)
			check("unknown", false)
			publish(strings.Replace(fixture("force"), "force: true", "force: false", 1))
			check("gray", false, invokers...)
			check("gray", true)
			publish(fixture("enabled"))
			check("gray", false, invokers[0])
			remove()
			require.Eventually(t, func() bool { _, exists := p.routerConfigs.Load(key); return !exists }, 15*time.Second, 50*time.Millisecond)
			check("gray", false, invokers...)
			check("", false, invokers...)
		})
	}
}

func adminTagAddresses(invokers []base.Invoker) []string {
	addresses := make([]string, 0, len(invokers))
	for _, invoker := range invokers {
		addresses = append(addresses, invoker.GetURL().Location)
	}
	return addresses
}

type adminTagIdentityInvoker struct{ *base.BaseInvoker }

func (p *adminTagIdentityInvoker) Invoke(context.Context, base.Invocation) result.Result {
	return &result.RPCResult{Rest: tri.NewResponse(wrapperspb.String(p.GetURL().Location))}
}
func startAdminTagProvider(t *testing.T, app string) base.Invoker {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	u, err := common.NewURL("tri://" + address + "/org.example.TagIdentity?interface=org.example.TagIdentity&application=" + app + "&timeout=3s&serialization=protobuf")
	require.NoError(t, err)
	server := triple.NewServer(nil)
	info := &common.ServiceInfo{InterfaceName: "org.example.TagIdentity", Methods: []common.MethodInfo{{Name: "Identity", Type: constant.CallUnary, ReqInitFunc: func() any { return &emptypb.Empty{} }}}}
	require.NoError(t, server.Start(&adminTagIdentityInvoker{base.NewBaseInvoker(u)}, info))
	t.Cleanup(server.Stop)
	client, err := triple.NewTripleInvoker(u.Clone())
	require.NoError(t, err)
	t.Cleanup(client.Destroy)
	require.Eventually(t, func() bool {
		response := new(wrapperspb.StringValue)
		call := adminTagRPCInvocation(response)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err := client.Invoke(ctx, call).Error()
		return err == nil && response.Value == address
	}, 5*time.Second, 50*time.Millisecond, "provider at %s", address)
	return client
}
func adminTagRPCInvocation(response *wrapperspb.StringValue) *invocation.RPCInvocation {
	call := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("Identity"), invocation.WithParameterRawValues([]any{&emptypb.Empty{}, response}))
	call.SetAttribute(constant.CallTypeKey, constant.CallUnary)
	return call
}
func invokeAdminTagProvider(t *testing.T, provider base.Invoker) string {
	t.Helper()
	response := new(wrapperspb.StringValue)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	require.NoError(t, provider.Invoke(ctx, adminTagRPCInvocation(response)).Error())
	return response.Value
}

func adminTagRealConfiguration(t *testing.T, backend, endpoint, key string) (config_center.DynamicConfiguration, func(string), func()) {
	t.Helper()
	u, err := common.NewURL(backend + "://" + endpoint)
	require.NoError(t, err)
	u.SetParam(constant.ClientNameKey, key)
	var publish func(string)
	var remove func()
	switch backend {
	case "nacos":
		u.SetParam(constant.ConfigGroupKey, "dubbo")
		mutate := func(method, content string) {
			values := url.Values{"dataId": {key}, "group": {"dubbo"}}
			if method == http.MethodPost {
				values.Set("content", content)
				values.Set("type", "yaml")
			}
			request, err := http.NewRequest(method, "http://"+endpoint+"/nacos/v1/cs/configs?"+values.Encode(), nil)
			require.NoError(t, err)
			response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
			require.NoError(t, err)
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode, string(body))
			require.Equal(t, "true", strings.TrimSpace(string(body)))
		}
		publish = func(content string) { mutate(http.MethodPost, content) }
		remove = func() { mutate(http.MethodDelete, "") }
		t.Logf("Admin Nacos location: group=dubbo dataId=%s", key)
	case "zookeeper":
		// Admin writes /dubbo/config/<name>. Go's defaults add another /dubbo
		// component, so use these supported settings to read the Admin location.
		u.SetParam(constant.ConfigRootPathParamKey, "/dubbo")
		u.SetParam(constant.ConfigNamespaceKey, "config")
		conn, events, err := zk.Connect([]string{endpoint}, 5*time.Second)
		require.NoError(t, err)
		t.Cleanup(conn.Close)
		require.Eventually(t, func() bool {
			select {
			case event := <-events:
				return event.State == zk.StateHasSession
			default:
				return conn.State() == zk.StateHasSession
			}
		}, 10*time.Second, 20*time.Millisecond)
		for _, path := range []string{"/dubbo", "/dubbo/config"} {
			_, err = conn.Create(path, nil, 0, zk.WorldACL(zk.PermAll))
			if err != zk.ErrNodeExists {
				require.NoError(t, err)
			}
		}
		path := "/dubbo/config/" + key
		publish = func(content string) {
			_, err := conn.Set(path, []byte(content), -1)
			if err == zk.ErrNoNode {
				_, err = conn.Create(path, []byte(content), 0, zk.WorldACL(zk.PermAll))
			}
			require.NoError(t, err)
		}
		remove = func() {
			err := conn.Delete(path, -1)
			if err != zk.ErrNoNode {
				require.NoError(t, err)
			}
		}
		t.Logf("Admin ZooKeeper location: %s; Go root-path=/dubbo namespace=config", path)
	}
	factory, err := extension.GetConfigCenterFactory(backend)
	require.NoError(t, err)
	dc, err := factory.GetDynamicConfiguration(u)
	require.NoError(t, err)
	if backend == "zookeeper" {
		// The current ZooKeeper driver does not initialize its done channel. Close
		// the transport directly until its Destroy lifecycle is repaired.
		if client, ok := dc.(interface {
			ZkClient() *gxzookeeper.ZookeeperClient
		}); ok {
			t.Cleanup(client.ZkClient().Close)
		}
	} else if destroyer, ok := dc.(interface{ Destroy() }); ok {
		t.Cleanup(destroyer.Destroy)
	}
	return dc, publish, remove
}
