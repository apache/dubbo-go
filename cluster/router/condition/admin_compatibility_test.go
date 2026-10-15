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
	"os"
	"path/filepath"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const (
	adminConditionInterface      = "org.apache.dubbo.condition.CompatService"
	adminConditionServiceKey     = adminConditionInterface + ":1.0.0:compat.condition-router"
	adminConditionProviderApp    = "condition-compat-provider"
	adminConditionConsumerApp    = "condition-compat-consumer"
	adminConditionApplicationKey = adminConditionProviderApp + ".condition-router"
)

// The fake records router subscriptions. The separate config-center test uses the
// real drivers to check transport groups and paths, rather than duplicating them here.
type adminConditionConfig struct {
	config_center.DynamicConfiguration
	rules     map[string]string
	listeners map[string]config_center.ConfigurationListener
	calls     []string
}

func (c *adminConditionConfig) AddListener(key string, l config_center.ConfigurationListener, _ ...config_center.Option) {
	c.calls = append(c.calls, "listen "+key)
	c.listeners[key] = l
}
func (c *adminConditionConfig) RemoveListener(key string, _ config_center.ConfigurationListener, _ ...config_center.Option) {
	c.calls = append(c.calls, "remove "+key)
	delete(c.listeners, key)
}
func (c *adminConditionConfig) GetRule(key string, _ ...config_center.Option) (string, error) {
	c.calls = append(c.calls, "get "+key)
	return c.rules[key], nil
}
func (c *adminConditionConfig) change(t *testing.T, key, value string, typ remoting.EventType) {
	t.Helper()
	l, ok := c.listeners[key]
	require.True(t, ok, "no subscription for %s", key)
	c.rules[key] = value
	l.Process(&config_center.ConfigChangeEvent{Key: key, Value: value, ConfigType: typ})
}
func adminConditionFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "admin", name+".yaml"))
	require.NoError(t, err)
	return string(b)
}
func adminConditionEnvironment(t *testing.T, dc config_center.DynamicConfiguration) {
	t.Helper()
	env := conf.GetEnvInstance()
	old := env.GetDynamicConfiguration()
	env.SetDynamicConfiguration(dc)
	t.Cleanup(func() { env.SetDynamicConfiguration(old) })
}
func adminConditionURLs(t *testing.T) (*common.URL, []base.Invoker) {
	t.Helper()
	consumer, err := common.NewURL("consumer://127.0.0.10/" + adminConditionInterface + "?application=" + adminConditionConsumerApp + "&group=compat&version=1.0.0")
	require.NoError(t, err)
	var providers []base.Invoker
	for _, p := range []struct{ host, env string }{{"127.0.0.1", "stable"}, {"127.0.0.2", "stable"}, {"127.0.0.3", "canary"}} {
		u, err := common.NewURL("tri://" + p.host + ":20880/" + adminConditionInterface + "?application=" + adminConditionProviderApp + "&group=compat&version=1.0.0&env=" + p.env)
		require.NoError(t, err)
		providers = append(providers, base.NewBaseInvoker(u))
	}
	return consumer, providers
}
func adminConditionSelected(router interface {
	Route([]base.Invoker, *common.URL, base.Invocation) []base.Invoker
}, consumer *common.URL, providers []base.Invoker) []string {
	selected := router.Route(providers, consumer, invocation.NewRPCInvocation("Ping", nil, nil))
	hosts := make([]string, 0, len(selected))
	for _, p := range selected {
		hosts = append(hosts, p.GetURL().Ip)
	}
	return hosts
}

func TestAdminServiceConditionLifecycle(t *testing.T) {
	c := &adminConditionConfig{rules: map[string]string{adminConditionServiceKey: adminConditionFixture(t, "service-initial")}, listeners: map[string]config_center.ConfigurationListener{}}
	adminConditionEnvironment(t, c)
	consumer, providers := adminConditionURLs(t)
	router := NewServiceRouter()
	router.Notify(providers)
	require.Equal(t, []string{"listen " + adminConditionServiceKey, "get " + adminConditionServiceKey}, c.calls)
	require.Equal(t, []string{"127.0.0.1", "127.0.0.2"}, adminConditionSelected(router, consumer, providers))
	for _, step := range []struct {
		name string
		want []string
	}{
		{"service-updated", []string{"127.0.0.2"}},
		{"service-disabled", []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}},
		{"service-updated", []string{"127.0.0.2"}},
		{"service-empty", []string{}},
	} {
		t.Run(step.name, func(t *testing.T) {
			c.change(t, adminConditionServiceKey, adminConditionFixture(t, step.name), remoting.EventTypeUpdate)
			require.Equal(t, step.want, adminConditionSelected(router, consumer, providers))
		})
	}
	c.change(t, adminConditionServiceKey, "", remoting.EventTypeDel)
	require.Equal(t, []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}, adminConditionSelected(router, consumer, providers))
}

func TestAdminApplicationConditionLifecycle(t *testing.T) {
	c := &adminConditionConfig{rules: map[string]string{adminConditionApplicationKey: adminConditionFixture(t, "application-initial")}, listeners: map[string]config_center.ConfigurationListener{}}
	adminConditionEnvironment(t, c)
	consumer, providers := adminConditionURLs(t)
	router := NewApplicationRouter(consumer)
	router.Notify(providers)
	require.Contains(t, c.calls, "listen "+adminConditionConsumerApp+constant.ConditionRouterRuleSuffix)
	require.Contains(t, c.calls, "listen "+adminConditionApplicationKey)
	require.Contains(t, c.calls, "get "+adminConditionApplicationKey)
	require.Equal(t, []string{"127.0.0.1", "127.0.0.2"}, adminConditionSelected(router, consumer, providers))
	for _, step := range []struct {
		name string
		want []string
	}{
		{"application-disable-a", []string{"127.0.0.2"}},
		{"application-empty", []string{}},
		{"application-restored", []string{"127.0.0.1", "127.0.0.2"}},
		{"application-disabled", []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}},
		{"application-initial", []string{"127.0.0.1", "127.0.0.2"}},
	} {
		t.Run(step.name, func(t *testing.T) {
			c.change(t, adminConditionApplicationKey, adminConditionFixture(t, step.name), remoting.EventTypeUpdate)
			require.Equal(t, step.want, adminConditionSelected(router, consumer, providers))
		})
	}
	c.change(t, adminConditionApplicationKey, "", remoting.EventTypeDel)
	require.Equal(t, []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}, adminConditionSelected(router, consumer, providers))
}

// Older Admin payloads omit false-valued proto fields. Missing flags retain the
// router defaults and must not panic; explicit false is checked by the lifecycle test.
func TestAdminConditionOmittedFlags(t *testing.T) {
	consumer, providers := adminConditionURLs(t)
	router := NewServiceRouter()
	router.Process(&config_center.ConfigChangeEvent{Value: adminConditionFixture(t, "legacy-enabled-omitted"), ConfigType: remoting.EventTypeUpdate})
	require.Equal(t, []string{"127.0.0.2"}, adminConditionSelected(router, consumer, providers))
	router.Process(&config_center.ConfigChangeEvent{Value: "configVersion: v3.0\nconditions: ['=>host=192.0.2.1']\n", ConfigType: remoting.EventTypeUpdate})
	require.Equal(t, []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}, adminConditionSelected(router, consumer, providers))
}
