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
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
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
	protocol base.Protocol
}

// Export service for remote invocation
func (pfw *ProtocolFilterWrapper) Export(invoker base.Invoker) base.Exporter {
	if pfw.protocol == nil {
		pfw.protocol = extension.GetProtocol(invoker.GetURL().Protocol)
	}
	invoker = BuildInvokerChain(invoker, constant.ServiceFilterKey)
	return pfw.protocol.Export(invoker)
}

// Refer a remote service
func (pfw *ProtocolFilterWrapper) Refer(url *common.URL) base.Invoker {
	if pfw.protocol == nil {
		pfw.protocol = extension.GetProtocol(url.Protocol)
	}
	invoker := pfw.protocol.Refer(url)
	if invoker == nil {
		return nil
	}
	return BuildInvokerChain(invoker, constant.ReferenceFilterKey)
}

// Destroy will destroy all invoker and exporter.
func (pfw *ProtocolFilterWrapper) Destroy() {
	pfw.protocol.Destroy()
}

func BuildInvokerChain(invoker base.Invoker, key string) base.Invoker {
	filterName := invoker.GetURL().GetParam(key, "")
	if filterName == "" {
		return invoker
	}
	filterNames := strings.Split(filterName, ",")

	// The order of filters is from left to right, so loading from right to left
	next := invoker
	for i := len(filterNames) - 1; i >= 0; i-- {
		flt, _ := extension.GetFilter(strings.TrimSpace(filterNames[i]))
		fi := &FilterInvoker{next: next, invoker: invoker, filter: flt}
		next = fi
	}
	switch key {
	case constant.ServiceFilterKey:
		logger.Debugf("[BuildInvokerChain] The provider invocation link is %s, invoker: %s",
			strings.Join(append(filterNames, "proxyInvoker"), " -> "), invoker)
	case constant.ReferenceFilterKey:
		logger.Debugf("[BuildInvokerChain] The consumer filters are %s, invoker: %s",
			strings.Join(append(filterNames, "proxyInvoker"), " -> "), invoker)
	}

	return next
}

// nolint
func GetProtocol() base.Protocol {
	return &ProtocolFilterWrapper{}
}

// FilterInvoker defines invoker and filter
type FilterInvoker struct {
	next    base.Invoker
	invoker base.Invoker
	filter  filter.Filter
}

// GetURL is used to get url from FilterInvoker
func (fi *FilterInvoker) GetURL() *common.URL {
	return fi.invoker.GetURL()
}

// IsAvailable is used to get available status
func (fi *FilterInvoker) IsAvailable() bool {
	return fi.invoker.IsAvailable()
}

// Invoke is used to call service method by invocation
func (fi *FilterInvoker) Invoke(ctx context.Context, invocation base.Invocation) base.Result {
	result := fi.filter.Invoke(ctx, fi.next, invocation)
	return fi.filter.OnResponse(ctx, result, fi.invoker, invocation)
}

// Destroy will destroy invoker
func (fi *FilterInvoker) Destroy() {
	fi.invoker.Destroy()
}
