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
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const (
	subscribeHostA = "127.0.0.1"
	subscribeHostB = "127.0.0.2"
)

// snapshotWindowConfig returns a stale GetRule snapshot while the test injects a
// listener event in the gap after AddListener and before that snapshot is applied.
type snapshotWindowConfig struct {
	adminConditionConfig
	blockKey string
	stale    string
	entered  chan struct{}
	release  chan struct{}
	openOnce sync.Once
}

func (c *snapshotWindowConfig) GetRule(key string, opts ...config_center.Option) (string, error) {
	c.calls = append(c.calls, "get "+key)
	if key != c.blockKey {
		return c.adminConditionConfig.GetRule(key, opts...)
	}
	c.openOnce.Do(func() { close(c.entered) })
	<-c.release
	return c.stale, nil
}

func conditionSelectsHost(host string) string {
	return "configVersion: v3.0\nenabled: true\nforce: true\nconditions:\n  - =>host=" + host + "\n"
}

func staticApplicationHostRule(host string) *global.RouterConfig {
	force := true
	enabled := true
	return &global.RouterConfig{
		Scope:      constant.RouterScopeApplication,
		Key:        adminConditionConsumerApp,
		Force:      &force,
		Enabled:    &enabled,
		Conditions: []string{"=> host = " + host},
	}
}

func chainRoutedHosts(ch *chain.RouterChain, consumer *common.URL) []string {
	selected := ch.Route(consumer, invocation.NewRPCInvocation("Ping", nil, nil))
	hosts := make([]string, 0, len(selected))
	for _, invoker := range selected {
		hosts = append(hosts, invoker.GetURL().Ip)
	}
	return hosts
}

// TestNewRouterChainKeepsSubscribedConsumerRuleOverStaticApplicationRule is the
// init-order acceptance: a consumer-application dynamic rule that selects B is
// already stored before subscribe, and an application static rule selects A.
// After NewRouterChain, including injectStaticRouters, routing selects B with
// no later config event.
func TestNewRouterChainKeepsSubscribedConsumerRuleOverStaticApplicationRule(t *testing.T) {
	key := adminConditionConsumerApp + constant.ConditionRouterRuleSuffix
	dynamic := &adminConditionConfig{
		rules:     map[string]string{key: adminConditionFixture(t, "consumer-application-initial")},
		listeners: map[string]config_center.ConfigurationListener{},
	}
	adminConditionEnvironment(t, dynamic)
	consumer, providers := adminConditionURLs(t)
	staticRule := staticApplicationHostRule(subscribeHostA)

	bare, err := common.NewURL("consumer://127.0.0.10/" + adminConditionInterface + "?group=compat&version=1.0.0")
	require.NoError(t, err)
	staticOnly := NewApplicationRouter(bare)
	staticOnly.SetStaticConfig(staticRule)
	require.Equal(t, []string{subscribeHostA}, adminConditionSelected(staticOnly, consumer, providers), "static application rule selects A")

	consumer.SetAttribute(constant.RoutersConfigKey, []*global.RouterConfig{staticRule})
	routeChain, err := chain.NewRouterChain(consumer)
	require.NoError(t, err)
	routeChain.SetInvokers(providers)

	require.Equal(t, []string{subscribeHostB}, chainRoutedHosts(routeChain, consumer))
	require.Contains(t, dynamic.calls, "listen "+key)
	require.Contains(t, dynamic.calls, "get "+key)
}

func TestNewRouterChainAppliesStaticApplicationRuleWhenDynamicRuleIsEmpty(t *testing.T) {
	key := adminConditionConsumerApp + constant.ConditionRouterRuleSuffix
	dynamic := &adminConditionConfig{
		rules:     map[string]string{},
		listeners: map[string]config_center.ConfigurationListener{},
	}
	adminConditionEnvironment(t, dynamic)
	consumer, providers := adminConditionURLs(t)
	consumer.SetAttribute(constant.RoutersConfigKey, []*global.RouterConfig{staticApplicationHostRule(subscribeHostA)})

	routeChain, err := chain.NewRouterChain(consumer)
	require.NoError(t, err)
	routeChain.SetInvokers(providers)

	require.Equal(t, []string{subscribeHostA}, chainRoutedHosts(routeChain, consumer))
	require.Contains(t, dynamic.calls, "get "+key)
}

func TestInitialGetRuleSnapshotDoesNotOverrideListenerEvent(t *testing.T) {
	key := adminConditionConsumerApp + constant.ConditionRouterRuleSuffix
	stale := conditionSelectsHost(subscribeHostA)
	updated := conditionSelectsHost(subscribeHostB)

	t.Run("update", func(t *testing.T) {
		consumer, providers := adminConditionURLs(t)
		router := startApplicationRouterDuringSnapshot(t, key, stale, func(listener config_center.ConfigurationListener) {
			listener.Process(&config_center.ConfigChangeEvent{Key: key, Value: updated, ConfigType: remoting.EventTypeUpdate})
		})
		require.Equal(t, []string{subscribeHostB}, adminConditionSelected(router, consumer, providers))
	})

	t.Run("delete", func(t *testing.T) {
		consumer, providers := adminConditionURLs(t)
		router := startApplicationRouterDuringSnapshot(t, key, stale, func(listener config_center.ConfigurationListener) {
			listener.Process(&config_center.ConfigChangeEvent{Key: key, Value: "", ConfigType: remoting.EventTypeDel})
		})
		require.Equal(t, []string{subscribeHostA, subscribeHostB, "127.0.0.3"}, adminConditionSelected(router, consumer, providers))
	})
}

func startApplicationRouterDuringSnapshot(
	t *testing.T,
	key, stale string,
	inject func(config_center.ConfigurationListener),
) *ApplicationRouter {
	t.Helper()
	cfg := &snapshotWindowConfig{
		adminConditionConfig: adminConditionConfig{
			rules:     map[string]string{},
			listeners: map[string]config_center.ConfigurationListener{},
		},
		blockKey: key,
		stale:    stale,
		entered:  make(chan struct{}),
		release:  make(chan struct{}),
	}
	adminConditionEnvironment(t, cfg)
	consumer, _ := adminConditionURLs(t)

	routerCh := make(chan *ApplicationRouter, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		routerCh <- NewApplicationRouter(consumer)
	}()
	var unblockOnce sync.Once
	unblock := func() { unblockOnce.Do(func() { close(cfg.release) }) }
	t.Cleanup(func() {
		unblock()
		<-done
	})

	select {
	case <-cfg.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("AddListener returned, but initial GetRule has not run")
		return nil
	}
	listener, ok := cfg.listeners[key]
	require.True(t, ok, "listener was not registered before GetRule")
	inject(listener)
	unblock()

	select {
	case router := <-routerCh:
		return router
	case <-time.After(2 * time.Second):
		t.Fatal("NewApplicationRouter did not finish after the listener event")
		return nil
	}
}
