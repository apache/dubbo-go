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
	"strings"
)

import (
	"github.com/RoaringBitmap/roaring"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

func (p *PriorityRouter) Name() string {
	return "tag"
}

func (p *PriorityRouter) ShouldPool() bool {
	return true
}

// Pool builds bitmap indices for tag, address, and port keys.
// Key space uses "\x00" as separator.
// Other Poolable routers should use different prefixes to avoid conflicts.
func (p *PriorityRouter) Pool(invokers []base.Invoker) (router.AddrPool, router.AddrMetadata) {
	pool := make(router.AddrPool)
	for i, invoker := range invokers {
		url := invoker.GetURL()
		upsertBM(pool, constant.PoolKeyAll, i)
		tag := url.GetParam(constant.Tagkey, "")
		upsertBM(pool, constant.PoolKeyTagPrefix+tag, i)
		addr := url.Location
		if addr != "" {
			upsertBM(pool, constant.PoolKeyAddrPrefix+addr, i)
			if idx := strings.LastIndex(addr, ":"); idx > 0 {
				upsertBM(pool, constant.PoolKeyPortPrefix+addr[idx+1:], i)
			}
		}
	}
	return pool, nil
}

func (p *PriorityRouter) SetCache(cache router.Cache) {
	if cache != nil {
		p.cache.Store(cache)
	}
}

func (p *PriorityRouter) routeWithPool(invokers []base.Invoker, pool router.AddrPool, url *common.URL, invocation base.Invocation) []base.Invoker {
	if len(invokers) == 0 {
		return invokers
	}
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
		if bm := pool[constant.PoolKeyTagPrefix+tag]; bm != nil && !bm.IsEmpty() {
			return bm
		}
		if requestIsForce(url, invocation) {
			return nil
		}
	}
	return pool[constant.PoolKeyTagPrefix]
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

	// ParamMatch is not bitmap-cached; fall back to requestTag.
	if len(match) != 0 {
		return nil
	}

	var resultBM *roaring.Bitmap
	if len(addresses) != 0 {
		resultBM = p.addressesBM(pool, addresses)
	} else {
		resultBM = pool[constant.PoolKeyTagPrefix+tag]
	}

	if (cfg.Force != nil && *cfg.Force) || requestIsForce(url, invocation) {
		return resultBM
	}
	if resultBM != nil && !resultBM.IsEmpty() {
		return resultBM
	}

	emptyBM := pool[constant.PoolKeyTagPrefix]
	if emptyBM == nil {
		return nil
	}
	if len(addresses) == 0 {
		return emptyBM
	}
	addrBM := p.addressesBM(pool, addresses)
	result := emptyBM.Clone()
	if addrBM != nil {
		result.AndNot(addrBM)
	}
	return result
}

func (p *PriorityRouter) emptyTagMatchBM(pool router.AddrPool, cfg global.RouterConfig) *roaring.Bitmap {
	bm := pool[constant.PoolKeyTagPrefix]
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
		if ab := pool[constant.PoolKeyAddrPrefix+addr]; ab != nil {
			bm.Or(ab)
		}
		if idx := strings.LastIndex(addr, ":"); idx > 0 && addr[:idx] == constant.AnyHostValue {
			if pb := pool[constant.PoolKeyPortPrefix+addr[idx+1:]]; pb != nil {
				bm.Or(pb)
			}
		}
	}
	return bm
}

func collectInvokers(invokers []base.Invoker, bm *roaring.Bitmap) []base.Invoker {
	if bm == nil || bm.IsEmpty() {
		return []base.Invoker{}
	}
	result := make([]base.Invoker, 0, bm.GetCardinality())
	it := bm.Iterator()
	for it.HasNext() {
		idx := it.Next()
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
