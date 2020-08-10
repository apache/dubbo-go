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
	"strconv"
)

import (
	"github.com/RoaringBitmap/roaring"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
)

const (
	name = "tag-router"
)

type tagRouter struct {
	url      *common.URL
	enabled  bool
	priority int64
}

func NewTagRouter(url *common.URL) (*tagRouter, error) {
	if url == nil {
		return nil, perrors.Errorf("Illegal route URL!")
	}
	return &tagRouter{
		url:      url,
		enabled:  url.GetParamBool(constant.RouterEnabled, true),
		priority: url.GetParamInt(constant.RouterPriority, 0),
	}, nil
}

func (c *tagRouter) isEnabled() bool {
	return c.enabled
}

func (c *tagRouter) Route(invokers *roaring.Bitmap, cache router.Cache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	if !c.isEnabled() || invokers.IsEmpty() {
		return invokers
	}

	tag := findStaticTag(invocation)
	if tag == "" {
		return invokers
	}

	ret := utils.EmptyAddr
	if target, ok := cache.FindAddrPool(c)[tag]; ok {
		ret = utils.JoinIfNotEqual(target, invokers)
	}

	if ret.IsEmpty() && !isForceUseTag(url, invocation) {
		return invokers
	}

	return ret
}

func (c *tagRouter) URL() common.URL {
	return *c.url
}

func (c *tagRouter) Priority() int64 {
	return c.priority
}

// Pool divided invokers into different address pool by tag.
func (c *tagRouter) Pool(invokers []protocol.Invoker) (router.AddrPool, router.AddrMetadata) {
	rb := make(router.AddrPool)
	for i, invoker := range invokers {
		url := invoker.GetUrl()
		tag := url.GetParam(constant.Tagkey, "")
		if tag != "" {
			if _, ok := rb[tag]; !ok {
				rb[tag] = roaring.NewBitmap()
			}
			rb[tag].AddInt(i)
		}
	}
	return rb, nil
}

// ShouldPool returns false, to make sure address cache for tag router happens once and only once.
func (c *tagRouter) ShouldPool() bool {
	return false
}

func (c *tagRouter) Name() string {
	return name
}

func findStaticTag(invocation protocol.Invocation) string {
	return invocation.Attachments()[constant.Tagkey]
}

func isForceUseTag(url *common.URL, invocation protocol.Invocation) bool {
	if b, e := strconv.ParseBool(invocation.AttachmentsByKey(constant.ForceUseTag, url.GetParam(constant.ForceUseTag, "false"))); e == nil {
		return b
	}
	return false
}
