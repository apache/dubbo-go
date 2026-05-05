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
	"regexp"
	"strings"
	"sync"
)

import (
	"github.com/RoaringBitmap/roaring"

	"github.com/dubbogo/gost/log/logger"

	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type PriorityRouter struct {
	routerConfigs sync.Map
	cache         router.Cache
}

func NewTagPriorityRouter() (*PriorityRouter, error) {
	return &PriorityRouter{}, nil
}

// Route Determine the target invokers list.
func (p *PriorityRouter) Route(invokers []base.Invoker, url *common.URL, invocation base.Invocation) []base.Invoker {
	if len(invokers) == 0 {
		logger.Warnf("[tag router] invokers from previous router is empty")
		return invokers
	}

	if p.cache != nil {
		fullInvokers, pool := p.cache.FindAddrPoolWithInvokers(p)
		if pool != nil && len(invokers) == len(fullInvokers) {
			return p.routeWithPool(fullInvokers, pool, url, invocation)
		}
	}

	// get application name from invoker to look up tag routing config
	application := invokers[0].GetURL().GetParam(constant.ApplicationKey, "")
	key := strings.Join([]string{application, constant.TagRouterRuleSuffix}, "")
	value, ok := p.routerConfigs.Load(key)
	if !ok {
		return staticTag(invokers, url, invocation)
	}
	routerCfg := value.(global.RouterConfig)
	enabled := routerCfg.Enabled == nil || *routerCfg.Enabled
	valid := (routerCfg.Valid != nil && *routerCfg.Valid) || (routerCfg.Valid == nil && len(routerCfg.Tags) > 0)
	if !enabled || !valid {
		return staticTag(invokers, url, invocation)
	}
	return dynamicTag(invokers, url, invocation, routerCfg)
}

func (p *PriorityRouter) URL() *common.URL {
	return nil
}

func (p *PriorityRouter) Priority() int64 {
	return 0
}

func (p *PriorityRouter) Notify(invokers []base.Invoker) {
	if len(invokers) == 0 {
		return
	}
	application := invokers[0].GetURL().GetParam(constant.ApplicationKey, "")
	if application == "" {
		logger.Warn("url application is empty, tag router will not be enabled")
		return
	}
	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Infof("Config center does not start, Tag router will not be enabled")
		return
	}
	key := strings.Join([]string{application, constant.TagRouterRuleSuffix}, "")
	dynamicConfiguration.AddListener(key, p)
	value, err := dynamicConfiguration.GetRule(key)
	if err != nil {
		logger.Errorf("query router rule fail,key=%s,err=%v", key, err)
		return
	}
	if value == "" {
		logger.Infof("router rule is empty,key=%s", key)
		return
	}
	p.Process(&config_center.ConfigChangeEvent{Key: key, Value: value, ConfigType: remoting.EventTypeAdd})
}

// SetStaticConfig applies a RouterConfig directly, bypassing YAML parsing.
// This is the correct entry point for static (code-configured) rules;
// Process is designed for dynamic config-center updates that arrive as YAML text.
// Static and dynamic rules are not merged: later Process updates replace the current state built here.
func (p *PriorityRouter) SetStaticConfig(cfg *global.RouterConfig) {
	if cfg == nil || cfg.Scope != constant.RouterScopeApplication || len(cfg.Tags) == 0 {
		return
	}
	cfgCopy := cfg.Clone()
	cfgCopy.Valid = new(bool)
	*cfgCopy.Valid = len(cfgCopy.Tags) > 0
	if cfgCopy.Enabled == nil {
		cfgCopy.Enabled = new(bool)
		*cfgCopy.Enabled = true
	}
	// Derive storage key the same way Notify() does: application + suffix
	key := strings.Join([]string{cfg.Key, constant.TagRouterRuleSuffix}, "")
	p.routerConfigs.Store(key, *cfgCopy)
	logger.Infof("[tag router] Applied static tag router config: key=%s", key)
}

// Process applies config-center updates as the authoritative rule source at runtime.
// It does not merge with static rules bootstrapped via SetStaticConfig; any later
// dynamic update replaces the current static-derived state.
func (p *PriorityRouter) Process(event *config_center.ConfigChangeEvent) {
	if event.ConfigType == remoting.EventTypeDel {
		p.routerConfigs.Delete(event.Key)
		return
	}
	routerConfig, err := parseRoute(event.Value.(string))
	if err != nil {
		logger.Warnf("[tag router]Parse new tag route config error, %+v "+
			"and we will use the original tag rule configuration.", err)
		return
	}
	p.routerConfigs.Store(event.Key, *routerConfig)
	logger.Infof("[tag router]Parse tag router config success,routerConfig=%+v", routerConfig)
}

func parseRoute(routeContent string) (*global.RouterConfig, error) {
	routeDecoder := yaml.NewDecoder(strings.NewReader(routeContent))
	routerConfig := &global.RouterConfig{}
	err := routeDecoder.Decode(routerConfig)
	if err != nil {
		return nil, err
	}
	routerConfig.Valid = new(bool)
	*routerConfig.Valid = true
	if len(routerConfig.Tags) == 0 {
		*routerConfig.Valid = false
	}
	return routerConfig, nil
}

func (p *PriorityRouter) Name() string {
	return "tag"
}

func (p *PriorityRouter) ShouldPool() bool {
	return true
}

func (p *PriorityRouter) Pool(invokers []base.Invoker) (router.AddrPool, router.AddrMetadata) {
	pool := make(router.AddrPool)
	for i, invoker := range invokers {
		url := invoker.GetURL()
		upsertBM(pool, "*", i)
		tag := url.GetParam(constant.Tagkey, "")
		upsertBM(pool, "tag\x00"+tag, i)
		addr := url.Location
		if addr != "" {
			upsertBM(pool, "addr\x00"+addr, i)
			if idx := strings.LastIndex(addr, ":"); idx > 0 {
				upsertBM(pool, "port\x00"+addr[idx+1:], i)
			}
		}
		for _, key := range []string{"version", "group"} {
			if v := url.GetParam(key, ""); v != "" {
				upsertBM(pool, "param\x00"+key+"\x00"+v, i)
			} else {
				upsertBM(pool, "param\x00"+key+"\x00\x00", i)
			}
		}
	}
	return pool, nil
}

func (p *PriorityRouter) SetCache(cache router.Cache) {
	p.cache = cache
}

func (p *PriorityRouter) routeWithPool(invokers []base.Invoker, pool router.AddrPool, url *common.URL, invocation base.Invocation) []base.Invoker {
	tag := invocation.GetAttachmentWithDefaultValue(constant.Tagkey, url.GetParam(constant.Tagkey, ""))

	application := invokers[0].GetURL().GetParam(constant.ApplicationKey, "")
	key := strings.Join([]string{application, constant.TagRouterRuleSuffix}, "")
	value, ok := p.routerConfigs.Load(key)
	if !ok {
		return collectInvokers(invokers, p.staticTagMatchBM(pool, tag, url, invocation))
	}
	routerCfg := value.(global.RouterConfig)
	enabled := routerCfg.Enabled == nil || *routerCfg.Enabled
	valid := (routerCfg.Valid != nil && *routerCfg.Valid) || (routerCfg.Valid == nil && len(routerCfg.Tags) > 0)
	if !enabled || !valid {
		return collectInvokers(invokers, p.staticTagMatchBM(pool, tag, url, invocation))
	}
	if tag == "" {
		return collectInvokers(invokers, p.emptyTagMatchBM(pool, routerCfg))
	}
	bm := p.requestTagMatchBM(pool, url, invocation, routerCfg, tag)
	if bm == nil {
		return requestTag(invokers, url, invocation, routerCfg, tag)
	}
	return collectInvokers(invokers, bm)
}

func (p *PriorityRouter) staticTagMatchBM(pool router.AddrPool, tag string, url *common.URL, invocation base.Invocation) *roaring.Bitmap {
	if tag != "" {
		if bm := pool["tag\x00"+tag]; bm != nil && !bm.IsEmpty() {
			return bm
		}
		if requestIsForce(url, invocation) {
			return nil
		}
	}
	return pool["tag\x00"]
}

func (p *PriorityRouter) requestTagMatchBM(pool router.AddrPool, url *common.URL, invocation base.Invocation,
	cfg global.RouterConfig, tag string) *roaring.Bitmap {
	var (
		addresses []string
		match     []*common.ParamMatch
	)
	for _, tagCfg := range cfg.Tags {
		if tagCfg.Name == tag {
			addresses = tagCfg.Addresses
			match = tagCfg.Match
			break
		}
	}

	var resultBM *roaring.Bitmap
	if len(match) != 0 {
		resultBM = p.matchBM(pool, match)
	} else if len(addresses) != 0 {
		resultBM = p.addressesBM(pool, addresses)
	} else {
		resultBM = pool["tag\x00"+tag]
	}

	if (cfg.Force != nil && *cfg.Force) || requestIsForce(url, invocation) {
		return resultBM
	}
	if resultBM != nil && !resultBM.IsEmpty() {
		return resultBM
	}

	emptyBM := pool["tag\x00"]
	if emptyBM == nil {
		return nil
	}
	if len(addresses) == 0 {
		return emptyBM
	}
	addrBM := p.addressesBM(pool, addresses)
	if addrBM != nil && !addrBM.IsEmpty() {
		allBM := pool["*"]
		if allBM != nil {
			result := allBM.Clone()
			result.AndNot(addrBM)
			return result
		}
	}
	return pool["*"]
}

func (p *PriorityRouter) emptyTagMatchBM(pool router.AddrPool, cfg global.RouterConfig) *roaring.Bitmap {
	bm := pool["tag\x00"]
	if bm == nil || bm.IsEmpty() {
		return bm
	}
	for _, tagCfg := range cfg.Tags {
		if len(tagCfg.Addresses) == 0 {
			continue
		}
		addrBM := p.addressesBM(pool, tagCfg.Addresses)
		if addrBM != nil && !addrBM.IsEmpty() && bm != nil {
			result := bm.Clone()
			result.AndNot(addrBM)
			bm = result
		}
	}
	return bm
}

func (p *PriorityRouter) addressesBM(pool router.AddrPool, addrs []string) *roaring.Bitmap {
	bm := roaring.NewBitmap()
	for _, addr := range addrs {
		if ab := pool["addr\x00"+addr]; ab != nil {
			bm.Or(ab)
		}
		if idx := strings.LastIndex(addr, ":"); idx > 0 && addr[:idx] == constant.AnyHostValue {
			if pb := pool["port\x00"+addr[idx+1:]]; pb != nil {
				bm.Or(pb)
			}
		}
	}
	return bm
}

func (p *PriorityRouter) matchBM(pool router.AddrPool, matches []*common.ParamMatch) *roaring.Bitmap {
	var result *roaring.Bitmap
	for _, m := range matches {
		bm := p.paramMatchBM(pool, m)
		if bm == nil {
			return nil
		}
		if result == nil {
			result = bm.Clone()
		} else {
			result.And(bm)
		}
		if result.IsEmpty() {
			return result
		}
	}
	return result
}

func (p *PriorityRouter) paramMatchBM(pool router.AddrPool, pm *common.ParamMatch) *roaring.Bitmap {
	prefix := "param\x00" + pm.Key + "\x00"
	switch {
	case pm.Value.Exact != "":
		return pool[prefix+pm.Value.Exact]
	case pm.Value.Wildcard != "":
		if pm.Value.Wildcard == constant.AnyValue {
			return pool["*"]
		}
		return pool[prefix+pm.Value.Wildcard]
	case pm.Value.Prefix != "":
		return scanPrefixBM(pool, prefix, pm.Value.Prefix)
	case pm.Value.Regex != "":
		return scanRegexBM(pool, prefix, pm.Value.Regex)
	case pm.Value.Empty != "":
		return pool[prefix+"\x00"]
	case pm.Value.Noempty != "":
		emptyBM := pool[prefix+"\x00"]
		allBM := pool["*"]
		if allBM == nil {
			return roaring.NewBitmap()
		}
		if emptyBM != nil {
			result := allBM.Clone()
			result.AndNot(emptyBM)
			return result
		}
		return allBM
	}
	return roaring.NewBitmap()
}

func collectInvokers(invokers []base.Invoker, bm *roaring.Bitmap) []base.Invoker {
	if bm == nil || bm.IsEmpty() {
		return []base.Invoker{}
	}
	result := make([]base.Invoker, 0, bm.GetCardinality())
	for _, idx := range bm.ToArray() {
		if int(idx) < len(invokers) {
			result = append(result, invokers[int(idx)])
		}
	}
	return result
}

func upsertBM(pool router.AddrPool, key string, idx int) {
	if _, ok := pool[key]; !ok {
		pool[key] = roaring.NewBitmap()
	}
	pool[key].Add(uint32(idx))
}

func scanPrefixBM(pool router.AddrPool, prefix string, value string) *roaring.Bitmap {
	bm := roaring.NewBitmap()
	for key, entry := range pool {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if strings.HasSuffix(key, "\x00") {
			continue
		}
		val := key[len(prefix):]
		if strings.HasPrefix(val, value) {
			bm.Or(entry)
		}
	}
	return bm
}

func scanRegexBM(pool router.AddrPool, prefix string, pattern string) *roaring.Bitmap {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return roaring.NewBitmap()
	}
	bm := roaring.NewBitmap()
	for key, entry := range pool {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if strings.HasSuffix(key, "\x00") {
			continue
		}
		val := key[len(prefix):]
		if re.MatchString(val) {
			bm.Or(entry)
		}
	}
	return bm
}
