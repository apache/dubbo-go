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

package condition

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

import (
	nacosclient "github.com/dubbogo/gost/database/kv/nacos"
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/go-zookeeper/zk"

	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/stretchr/testify/require"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/zookeeper"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// TestAdminConditionConfigCenterRPC is opt-in because it uses real config centers.
// The raw writers below use Admin's wire contract, while subscriptions and calls
// go through Dubbo-Go's production drivers, routers and Triple invokers.
func TestAdminConditionConfigCenterRPC(t *testing.T) {
	for _, backend := range []struct{ name, env string }{{"nacos", "DUBBO_GO_CONDITION_NACOS"}, {"zookeeper", "DUBBO_GO_CONDITION_ZK"}} {
		t.Run(backend.name, func(t *testing.T) {
			address := os.Getenv(backend.env)
			if address == "" {
				t.Skip("set " + backend.env + " to run with a real config center")
			}
			u, err := common.NewURL(backend.name + "://" + address)
			require.NoError(t, err)
			if backend.name == "nacos" {
				u.SetParam(constant.ConfigGroupKey, "dubbo")
				u.SetParam(constant.ClientNameKey, "condition-compat")
				u.SetParam(constant.NacosLogDirKey, t.TempDir())
				u.SetParam(constant.NacosCacheDirKey, t.TempDir())
				u.SetParam(constant.NacosLogLevelKey, "error")
			} else {
				// Admin's existing governor writes /dubbo/config/<resource-name>.
				// Go defaults to /dubbo/config/dubbo/<key>; these existing options align them.
				u.SetParam(constant.ConfigRootPathParamKey, "/dubbo")
				u.SetParam(constant.ConfigNamespaceKey, "config")
			}
			factory, err := extension.GetConfigCenterFactory(backend.name)
			require.NoError(t, err)
			dc, err := factory.GetDynamicConfiguration(u)
			require.NoError(t, err)
			t.Cleanup(func() {
				if backend.name == "zookeeper" {
					// The current driver leaves its stop channel nil. Close the
					// real client here; driver lifecycle changes are outside this test.
					dc.(interface {
						ZkClient() *gxzookeeper.ZookeeperClient
					}).ZkClient().Close()
				} else {
					dc.(interface{ Destroy() }).Destroy()
				}
			})
			adminConditionEnvironment(t, dc)
			write, remove := adminConditionRawWriter(t, backend.name, dc)
			// ListenConfigurationEvent registers asynchronously. Check delivery
			// before asserting a one-shot router update, without a fixed sleep.
			probeKey := "condition-compat-watch-probe"
			probe := &adminConditionProbe{events: make(chan struct{}, 8)}
			dc.AddListener(probeKey, probe)
			t.Cleanup(func() { dc.RemoveListener(probeKey, probe); _ = remove(probeKey) })
			require.Eventually(t, func() bool {
				if err := write(probeKey, fmt.Sprint(time.Now().UnixNano())); err != nil {
					return false
				}
				select {
				case <-probe.events:
					return true
				default:
					return false
				}
			}, 20*time.Second, 100*time.Millisecond, "config-center listener did not receive native writes")
			consumer, providers := adminConditionRPCProviders(t)
			for _, scope := range []string{"service", "application"} {
				t.Run(scope, func(t *testing.T) {
					key := adminConditionServiceKey
					var conditionRouter interface {
						router.PriorityRouter
						config_center.ConfigurationListener
					}
					if scope == "service" {
						conditionRouter = NewServiceRouter()
					} else {
						key = adminConditionApplicationKey
						conditionRouter = NewApplicationRouter(consumer)
					}
					// Retarget the fixed fixture IP to a second, genuinely reachable local IP.
					fixture := func(name string) string {
						return strings.ReplaceAll(adminConditionFixture(t, name), "127.0.0.2", providers[1].GetURL().Ip)
					}
					require.NoError(t, write(key, fixture(scope+"-initial")))
					t.Cleanup(func() { _ = remove(key); dc.RemoveListener(key, conditionRouter) })
					routeChain := &chain.RouterChain{}
					routeChain.AddRouters([]router.PriorityRouter{conditionRouter})
					routeChain.SetInvokers(providers)
					check := func(label string, want []string) {
						t.Helper()
						require.Eventually(t, func() bool {
							selected := routeChain.Route(consumer, invocation.NewRPCInvocation("Ping", nil, nil))
							if len(selected) != len(want) {
								return false
							}
							for i, p := range selected {
								if p.GetURL().GetParam("instance", "") != want[i] {
									return false
								}
							}
							return true
						}, 20*time.Second, 50*time.Millisecond, label)
						selected := routeChain.Route(consumer, invocation.NewRPCInvocation("Ping", nil, nil))
						for i, p := range selected {
							response := &wrapperspb.StringValue{}
							inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("Ping"), invocation.WithParameterRawValues([]any{wrapperspb.String("request"), response}))
							inv.SetAttribute(constant.CallTypeKey, constant.CallUnary)
							ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
							result := p.Invoke(ctx, inv)
							cancel()
							require.NoError(t, result.Error(), label)
							require.Equal(t, want[i], response.Value, label)
						}
						t.Logf("%s %s key=%s: RPC providers=%v", backend.name, label, key, want)
					}
					check("present before subscription", []string{"A", "B"})
					if scope == "service" {
						require.NoError(t, write(key, fixture("service-updated")))
						check("update", []string{"B"})
					} else {
						require.NoError(t, write(key, fixture("application-disable-a")))
						check("disable instance A", []string{"B"})
						require.NoError(t, write(key, fixture("application-restored")))
						check("restore A, preserve env condition", []string{"A", "B"})
					}
					require.NoError(t, write(key, fixture(scope+"-disabled")))
					check("disable rule", []string{"A", "B", "C"})
					require.NoError(t, write(key, fixture(scope+"-initial")))
					check("enable rule", []string{"A", "B"})
					require.NoError(t, write(key, fixture(scope+"-empty")))
					check("forced empty", []string{})
					require.NoError(t, remove(key))
					check("delete rule", []string{"A", "B", "C"})
				})
			}
		})
	}
}

func adminConditionRawWriter(t *testing.T, backend string, dc config_center.DynamicConfiguration) (func(string, string) error, func(string) error) {
	t.Helper()
	if backend == "nacos" {
		client := dc.(interface {
			NacosClient() *nacosclient.NacosConfigClient
		}).NacosClient().Client()
		return func(key, value string) error {
				ok, err := client.PublishConfig(vo.ConfigParam{DataId: key, Group: "dubbo", Content: value})
				if err == nil && !ok {
					return fmt.Errorf("Nacos rejected %s", key)
				}
				if err != nil {
					return err
				}
				deadline := time.Now().Add(5 * time.Second)
				var got string
				for time.Now().Before(deadline) {
					got, err = client.GetConfig(vo.ConfigParam{DataId: key, Group: "dubbo"})
					if err == nil && got == value {
						return nil
					}
					time.Sleep(25 * time.Millisecond)
				}
				return fmt.Errorf("Nacos group=dubbo dataId=%s content not visible: %v", key, err)
			}, func(key string) error {
				_, err := client.DeleteConfig(vo.ConfigParam{DataId: key, Group: "dubbo"})
				return err
			}
	}
	conn := dc.(interface {
		ZkClient() *gxzookeeper.ZookeeperClient
	}).ZkClient().Conn
	if _, err := conn.Create("/dubbo/config", nil, 0, zk.WorldACL(zk.PermAll)); err != nil && err != zk.ErrNodeExists {
		require.NoError(t, err)
	}
	return func(key, value string) error {
		path := "/dubbo/config/" + key
		_, err := conn.Create(path, []byte(value), 0, zk.WorldACL(zk.PermAll))
		if err == zk.ErrNodeExists {
			_, err = conn.Set(path, []byte(value), -1)
		}
		if err != nil {
			return err
		}
		got, _, err := conn.Get(path)
		if err == nil && string(got) != value {
			return fmt.Errorf("ZooKeeper path=%s content mismatch", path)
		}
		return err
	}, func(key string) error { return conn.Delete("/dubbo/config/"+key, -1) }
}

func adminConditionRPCProviders(t *testing.T) (*common.URL, []base.Invoker) {
	t.Helper()
	second := os.Getenv("DUBBO_GO_CONDITION_PROVIDER_B")
	if second == "" {
		addresses, err := net.InterfaceAddrs()
		require.NoError(t, err)
		for _, addr := range addresses {
			ip, _, err := net.ParseCIDR(addr.String())
			if err == nil && ip.To4() != nil && ip.IsGlobalUnicast() && !ip.IsLoopback() {
				second = ip.String()
				break
			}
		}
	}
	require.NotEmpty(t, second, "set DUBBO_GO_CONDITION_PROVIDER_B to a second locally reachable IPv4 address")
	require.NotEqual(t, "127.0.0.1", second, "Admin disables by host, so A and B must have different IPs")
	consumer, _ := adminConditionURLs(t)
	var providers []base.Invoker
	for i, item := range []struct{ id, host, env string }{{"A", "127.0.0.1", "stable"}, {"B", second, "stable"}, {"C", "127.0.0.1", "canary"}} {
		handler := tri.NewUnaryHandler("/"+adminConditionInterface+"/Ping", func() any { return &wrapperspb.StringValue{} }, func(_ context.Context, _ *tri.Request) (*tri.Response, error) {
			return tri.NewResponse(wrapperspb.String(item.id)), nil
		}, tri.WithGroup("compat"), tri.WithVersion("1.0.0"))
		server := &http.Server{Handler: h2c.NewHandler(handler, &http2.Server{})}
		listener, err := net.Listen("tcp", "0.0.0.0:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = server.Close() })
		go func() { _ = server.Serve(listener) }()
		address := net.JoinHostPort(item.host, fmt.Sprint(listener.Addr().(*net.TCPAddr).Port))
		u, err := common.NewURL("tri://"+address+"/"+adminConditionInterface,
			common.WithMethods([]string{"Ping"}), common.WithParamsValue(constant.IDLMode, constant.NONIDL),
			common.WithParamsValue(constant.ApplicationKey, adminConditionProviderApp), common.WithParamsValue(constant.GroupKey, "compat"), common.WithParamsValue(constant.VersionKey, "1.0.0"),
			common.WithParamsValue("env", item.env), common.WithParamsValue("instance", item.id))
		require.NoError(t, err)
		invoker := triple.GetProtocol().Refer(u)
		require.NotNil(t, invoker)
		t.Cleanup(invoker.Destroy)
		providers = append(providers, invoker)
		t.Logf("provider %s endpoint=%s env=%s index=%d", item.id, address, item.env, i)
	}
	return consumer, providers
}

type adminConditionProbe struct{ events chan struct{} }

func (p *adminConditionProbe) Process(*config_center.ConfigChangeEvent) {
	select {
	case p.events <- struct{}{}:
	default:
	}
}
