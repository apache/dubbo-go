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
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
)

// tagRouter defines url, enable and the priority
type tagRouter struct {
	url      *common.URL
	enabled  bool
	priority int64
}

// NewTagRouter returns a tagRouter instance if url is not nil
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

// nolint
func (c *tagRouter) isEnabled() bool {
	return c.enabled
}

// Route gets a list of invoker
func (c *tagRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if !c.isEnabled() {
		return invokers
	}
	if len(invokers) == 0 {
		return invokers
	}
	return filterUsingStaticTag(invokers, url, invocation)
}

// URL gets the url of tagRouter
func (c *tagRouter) URL() common.URL {
	return *c.url
}

// Priority gets the priority of tagRouter
func (c *tagRouter) Priority() int64 {
	return c.priority
}

// filterUsingStaticTag gets a list of invoker using static tag
func filterUsingStaticTag(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if tag, ok := invocation.Attachments()[constant.Tagkey]; ok {
		result := make([]protocol.Invoker, 0, 8)
		for _, v := range invokers {
			if v.GetUrl().GetParam(constant.Tagkey, "") == tag {
				result = append(result, v)
			}
		}
		if len(result) == 0 && !isForceUseTag(url, invocation) {
			return invokers
		}
		return result
	}
	return invokers
}

// isForceUseTag returns whether force use tag
func isForceUseTag(url *common.URL, invocation protocol.Invocation) bool {
	if b, e := strconv.ParseBool(invocation.AttachmentsByKey(constant.ForceUseTag, url.GetParam(constant.ForceUseTag, "false"))); e == nil {
		return b
	}
	return false
}
