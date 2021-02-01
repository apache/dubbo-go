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

package local

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
	localPriority = "local-priority"
	name          = "local-priority-router"
)

// LocalPriorityRouter provides a ip-same-first routing logic
// if there is not provider with same ip as consumer, it would not filter any invoker
// if exists same ip invoker, it would retains this invoker only
type LocalPriorityRouter struct {
	url     *common.URL
	localIP string
}

// NewLocalPriorityRouter construct an LocalPriorityRouter via url
func NewLocalPriorityRouter(url *common.URL) (router.PriorityRouter, error) {
	r := &LocalPriorityRouter{
		url:     url,
		localIP: url.Ip,
	}
	return r, nil
}

// Route gets a list of match-logic invoker
func (r *LocalPriorityRouter) Route(invokers *roaring.Bitmap, cache router.Cache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	addrPool := cache.FindAddrPool(r)
	// Add localPriority invoker to the list
	selectedInvokers := utils.JoinIfNotEqual(addrPool[localPriority], invokers)
	// If all invokers are considered not match, downgrade to all invoker
	if selectedInvokers.IsEmpty() {
		logger.Warnf(" Now all invokers are not match, so downgraded to all! Service: [%s]", url.ServiceKey())
		return invokers
	}
	return selectedInvokers
}

// Pool separates same ip invoker from others.
func (r *LocalPriorityRouter) Pool(invokers []protocol.Invoker) (router.AddrPool, router.AddrMetadata) {
	rb := make(router.AddrPool, 8)
	rb[localPriority] = roaring.NewBitmap()
	localIpFound := false
	for i, invoker := range invokers {
		if invoker.GetUrl().Ip == r.localIP {
			rb[localPriority].Add(uint32(i))
			localIpFound = true
		}
	}
	if localIpFound {
		// found local desc
		logger.Debug("found local ip ")
		return rb, nil
	}
	for i, _ := range invokers {
		rb[localPriority].Add(uint32(i))
	}
	return rb, nil
}

// ShouldPool will always return true to make sure local call logic constantly.
func (r *LocalPriorityRouter) ShouldPool() bool {
	return true
}

func (r *LocalPriorityRouter) Name() string {
	return name
}

// Priority
func (r *LocalPriorityRouter) Priority() int64 {
	return 0
}

// URL Return URL in router
func (r *LocalPriorityRouter) URL() *common.URL {
	return r.url
}
