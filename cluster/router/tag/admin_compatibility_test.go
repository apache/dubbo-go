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
	"os"
	"strconv"
	"strings"
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

const adminTagProvider = "tag-compat-provider"
const adminTagKey = adminTagProvider + constant.TagRouterRuleSuffix

// The embedded interface catches any unexpected config-center calls. Notify
// must subscribe and fetch the provider application's key, without a group override.
type adminTagConfiguration struct {
	config_center.DynamicConfiguration
	content  string
	calls    []string
	listener config_center.ConfigurationListener
}

func (c *adminTagConfiguration) AddListener(key string, l config_center.ConfigurationListener, opts ...config_center.Option) {
	c.calls = append(c.calls, "listen:"+key)
	if len(opts) != 0 {
		panic("unexpected listener options")
	}
	c.listener = l
}
func (c *adminTagConfiguration) GetRule(key string, opts ...config_center.Option) (string, error) {
	c.calls = append(c.calls, "get:"+key)
	if len(opts) != 0 {
		panic("unexpected rule options")
	}
	return c.content, nil
}
func adminTagFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/admin/" + name + ".yaml")
	require.NoError(t, err)
	return string(b)
}
func adminTagInvokers(t *testing.T) ([]base.Invoker, *common.URL) {
	t.Helper()
	var invokers []base.Invoker
	for _, addr := range []string{"127.0.0.1:20890", "127.0.0.1:20891"} {
		u, err := common.NewURL("tri://" + addr + "/org.example.TagIdentity?application=" + adminTagProvider)
		require.NoError(t, err)
		invokers = append(invokers, base.NewBaseInvoker(u))
	}
	consumer, err := common.NewURL("consumer://127.0.0.1/org.example.TagIdentity?application=tag-compat-consumer")
	require.NoError(t, err)
	return invokers, consumer
}
func assertAdminTagSelection(t *testing.T, p *PriorityRouter, invokers []base.Invoker, consumer *common.URL, tag string, force bool, want ...base.Invoker) {
	t.Helper()
	pool, _ := p.Pool(invokers)
	p.SetCache(&mockCache{invokers: invokers, pool: pool, generation: 7})
	for _, cached := range []bool{false, true} {
		call := invocation.NewRPCInvocation("Identity", nil, map[string]any{constant.Tagkey: tag, constant.ForceUseTag: strconv.FormatBool(force)})
		call.SetAttribute(constant.RouterCacheDisable, !cached)
		call.SetAttribute(constant.RouterChainCacheGeneration, uint64(7))
		got := p.Route(invokers, consumer, call)
		require.ElementsMatch(t, want, got, "tag=%q force=%v cached=%v", tag, force, cached)
	}
}

func TestAdminTagRuleLifecycle(t *testing.T) {
	invokers, consumer := adminTagInvokers(t)
	dc := &adminTagConfiguration{content: adminTagFixture(t, "enabled")}
	env := conf.GetEnvInstance()
	previous := env.GetDynamicConfiguration()
	env.SetDynamicConfiguration(dc)
	t.Cleanup(func() { env.SetDynamicConfiguration(previous) })
	p, err := NewTagPriorityRouter()
	require.NoError(t, err)
	p.Notify(invokers)
	require.Equal(t, []string{"listen:" + adminTagKey, "get:" + adminTagKey}, dc.calls)
	require.Same(t, p, dc.listener)
	assertAdminTagSelection(t, p, invokers, consumer, "gray", false, invokers[0])
	assertAdminTagSelection(t, p, invokers, consumer, "", false, invokers[1])
	assertAdminTagSelection(t, p, invokers, consumer, "unknown", false, invokers...)
	assertAdminTagSelection(t, p, invokers, consumer, "unknown", true)

	dc.listener.Process(&config_center.ConfigChangeEvent{Key: adminTagKey, Value: adminTagFixture(t, "updated"), ConfigType: remoting.EventTypeUpdate})
	assertAdminTagSelection(t, p, invokers, consumer, "gray", false, invokers[1])
	assertAdminTagSelection(t, p, invokers, consumer, "", false, invokers[0])
	// The companion Admin writer preserves false. Keep the legacy omitted
	// default separate so a control-plane fix cannot change SDK defaults.
	disabled := adminTagFixture(t, "fixed-disabled")
	dc.listener.Process(&config_center.ConfigChangeEvent{Key: adminTagKey, Value: disabled, ConfigType: remoting.EventTypeUpdate})
	assertAdminTagSelection(t, p, invokers, consumer, "gray", false, invokers...)
	assertAdminTagSelection(t, p, invokers, consumer, "", false, invokers...)
	dc.listener.Process(&config_center.ConfigChangeEvent{Key: adminTagKey, ConfigType: remoting.EventTypeDel})
	_, exists := p.routerConfigs.Load(adminTagKey)
	require.False(t, exists)
	assertAdminTagSelection(t, p, invokers, consumer, "gray", false, invokers...)
}

func TestAdminTagOptionalFields(t *testing.T) {
	invokers, consumer := adminTagInvokers(t)
	omitted := adminTagFixture(t, "legacy-disabled-omitted")
	cfg, err := parseRoute(omitted)
	require.NoError(t, err)
	require.Nil(t, cfg.Enabled)
	require.Nil(t, cfg.Force)
	require.Nil(t, cfg.Runtime)
	p, err := NewTagPriorityRouter()
	require.NoError(t, err)
	p.Process(&config_center.ConfigChangeEvent{Key: adminTagKey, Value: omitted, ConfigType: remoting.EventTypeAdd})
	// Missing enabled means enabled. This also exposes why the pinned Admin
	// writer cannot disable the rule by omitting enabled=false.
	assertAdminTagSelection(t, p, invokers, consumer, "gray", false, invokers[1])

	explicit := omitted + "enabled: true\nforce: false\nruntime: false\n"
	cfg, err = parseRoute(explicit)
	require.NoError(t, err)
	require.True(t, *cfg.Enabled)
	require.False(t, *cfg.Force)
	require.False(t, *cfg.Runtime)
	p.Process(&config_center.ConfigChangeEvent{Key: adminTagKey, Value: explicit, ConfigType: remoting.EventTypeUpdate})
	assertAdminTagSelection(t, p, invokers, consumer, "gray", false, invokers[1])
}

func TestAdminTagAddressFallbackAndForce(t *testing.T) {
	invokers, consumer := adminTagInvokers(t)
	forced := adminTagFixture(t, "force")
	p, err := NewTagPriorityRouter()
	require.NoError(t, err)
	p.Process(&config_center.ConfigChangeEvent{Key: adminTagKey, Value: forced, ConfigType: remoting.EventTypeAdd})
	assertAdminTagSelection(t, p, invokers, consumer, "gray", false)
	assertAdminTagSelection(t, p, invokers, consumer, "unknown", false)
	assertAdminTagSelection(t, p, invokers, consumer, "", false, invokers...)
	// With the address absent, force=false falls back to the untagged providers.
	fallback := strings.Replace(forced, "force: true", "force: false", 1)
	p.Process(&config_center.ConfigChangeEvent{Key: adminTagKey, Value: fallback, ConfigType: remoting.EventTypeUpdate})
	assertAdminTagSelection(t, p, invokers, consumer, "gray", false, invokers...)
	assertAdminTagSelection(t, p, invokers, consumer, "gray", true)
	// Disabled dynamic rules retain the existing static tag behavior.
	invokers[0].GetURL().SetParam(constant.Tagkey, "gray")
	p.Process(&config_center.ConfigChangeEvent{Key: adminTagKey, Value: strings.Replace(forced, "enabled: true", "enabled: false", 1), ConfigType: remoting.EventTypeUpdate})
	assertAdminTagSelection(t, p, invokers, consumer, "gray", false, invokers[0])
	assertAdminTagSelection(t, p, invokers, consumer, "", false, invokers[1])
}
