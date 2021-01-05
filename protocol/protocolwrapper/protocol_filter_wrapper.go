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

package protocolwrapper

import (
	"context"
	"strings"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

const (
	// FILTER is protocol key.
	FILTER = "filter"
)

func init() {
	extension.SetProtocol(FILTER, GetProtocol)
}

// ProtocolFilterWrapper
// protocol in url decide who ProtocolFilterWrapper.protocol is
type ProtocolFilterWrapper struct {
	protocol protocol.Protocol
}

// Export service for remote invocation
func (pfw *ProtocolFilterWrapper) Export(invoker protocol.Invoker) protocol.Exporter {
	if pfw.protocol == nil {
		pfw.protocol = extension.GetProtocol(invoker.GetUrl().Protocol)
	}
	invoker = buildInvokerChain(invoker, constant.SERVICE_FILTER_KEY)
	return pfw.protocol.Export(invoker)
}

// Refer a remote service
func (pfw *ProtocolFilterWrapper) Refer(url *common.URL) protocol.Invoker {
	if pfw.protocol == nil {
		pfw.protocol = extension.GetProtocol(url.Protocol)
	}
	return buildInvokerChain(pfw.protocol.Refer(url), constant.REFERENCE_FILTER_KEY)
}

// Destroy will destroy all invoker and exporter.
func (pfw *ProtocolFilterWrapper) Destroy() {
	pfw.protocol.Destroy()
}

func buildInvokerChain(invoker protocol.Invoker, key string) protocol.Invoker {
	filterName := invoker.GetUrl().GetParam(key, "")
	if filterName == "" {
		return invoker
	}
	filterNames := strings.Split(filterName, ",")

	// The order of filters is from left to right, so loading from right to left
	next := invoker
	for i := len(filterNames) - 1; i >= 0; i-- {
		flt := extension.GetFilter(filterNames[i])
		fi := &FilterInvoker{next: next, invoker: invoker, filter: flt}
		next = fi
	}
	return next
}

// nolint
func GetProtocol() protocol.Protocol {
	return &ProtocolFilterWrapper{}
}

///////////////////////////
// filter invoker
///////////////////////////

// FilterInvoker defines invoker and filter
type FilterInvoker struct {
	next    protocol.Invoker
	invoker protocol.Invoker
	filter  filter.Filter
}

// GetUrl is used to get url from FilterInvoker
func (fi *FilterInvoker) GetUrl() *common.URL {
	return fi.invoker.GetUrl()
}

// IsAvailable is used to get available status
func (fi *FilterInvoker) IsAvailable() bool {
	return fi.invoker.IsAvailable()
}

// Invoke is used to call service method by invocation
func (fi *FilterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	result := fi.filter.Invoke(ctx, fi.next, invocation)
	return fi.filter.OnResponse(ctx, result, fi.invoker, invocation)
}

// Destroy will destroy invoker
func (fi *FilterInvoker) Destroy() {
	fi.invoker.Destroy()
}
