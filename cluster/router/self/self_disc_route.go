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

package self

import (
	"github.com/RoaringBitmap/roaring"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

const (
	selfDesc = "self-desc"
	name     = "self-desc-router"
)

// SelfDiscRouter provides a ip-same-first routing logic
// if there is not provider with same ip as consumer, it would not filter any invoker
// if exists same ip invoker, it would retains this invoker only
type SelfDiscRouter struct {
	url     *common.URL
	localIP string
}

// NewSelfDiscRouter construct an SelfDiscRouter via url
func NewSelfDiscRouter(url *common.URL) (router.PriorityRouter, error) {
	r := &SelfDiscRouter{
		url:     url,
		localIP: url.Ip,
	}
	return r, nil
}

// Route gets a list of match-logic invoker
func (r *SelfDiscRouter) Route(invokers *roaring.Bitmap, cache router.Cache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	addrPool := cache.FindAddrPool(r)
	// Add selfDesc invoker to the list
	selectedInvokers := utils.JoinIfNotEqual(addrPool[selfDesc], invokers)
	// If all invokers are considered not match, downgrade to all invoker
	if selectedInvokers.IsEmpty() {
		logger.Warnf(" Now all invokers are not match, so downgraded to all! Service: [%s]", url.ServiceKey())
		return invokers
	}
	return selectedInvokers
}

// Pool separates same ip invoker from others.
func (r *SelfDiscRouter) Pool(invokers []protocol.Invoker) (router.AddrPool, router.AddrMetadata) {
	rb := make(router.AddrPool, 8)
	rb[selfDesc] = roaring.NewBitmap()
	selfDescFound := false
	for i, invoker := range invokers {
		if invoker.GetUrl().Ip == r.localIP {
			rb[selfDesc].Add(uint32(i))
			selfDescFound = true
		}
	}
	if selfDescFound {
		// found self desc
		logger.Debug("found self desc ")
		return rb, nil
	}
	for i, _ := range invokers {
		rb[selfDesc].Add(uint32(i))
	}
	return rb, nil
}

// ShouldPool will always return true to make sure self call logic constantly.
func (r *SelfDiscRouter) ShouldPool() bool {
	return true
}

func (r *SelfDiscRouter) Name() string {
	return name
}

// Priority
func (r *SelfDiscRouter) Priority() int64 {
	return 0
}

// URL Return URL in router
func (r *SelfDiscRouter) URL() *common.URL {
	return r.url
}
