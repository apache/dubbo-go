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

package filter_impl

import (
	"bytes"
	"context"
	"fmt"
)

import (
	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

func init() {
	extension.SetFilter(SentinelProviderFilterName, GetSentinelProviderFilter)
	extension.SetFilter(SentinelConsumerFilterName, GetSentinelConsumerFilter)
}

func GetSentinelConsumerFilter() filter.Filter {
	return &SentinelConsumerFilter{}
}

func GetSentinelProviderFilter() filter.Filter {
	return &SentinelProviderFilter{}
}

type SentinelProviderFilter struct{}

func (d *SentinelProviderFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	methodResourceName := getResourceName(invoker, invocation, getProviderPrefix())
	interfaceResourceName := ""
	if getInterfaceGroupAndVersionEnabled() {
		interfaceResourceName = getColonSeparatedKey(invoker.GetUrl())
	} else {
		interfaceResourceName = invoker.GetUrl().Service()
	}
	var (
		interfaceEntry *base.SentinelEntry
		methodEntry    *base.SentinelEntry
		b              *base.BlockError
	)
	interfaceEntry, b = sentinel.Entry(interfaceResourceName, sentinel.WithResourceType(base.ResTypeRPC), sentinel.WithTrafficType(base.Inbound))
	if b != nil {
		// interface blocked
		return sentinelDubboProviderFallback(ctx, invoker, invocation, b)
	}
	ctx = context.WithValue(ctx, InterfaceEntryKey, interfaceEntry)

	methodEntry, b = sentinel.Entry(methodResourceName, sentinel.WithResourceType(base.ResTypeRPC),
		sentinel.WithTrafficType(base.Inbound), sentinel.WithArgs(invocation.Arguments()...))
	if b != nil {
		// method blocked
		return sentinelDubboProviderFallback(ctx, invoker, invocation, b)
	}
	ctx = context.WithValue(ctx, MethodEntryKey, methodEntry)
	return invoker.Invoke(ctx, invocation)
}

func (d *SentinelProviderFilter) OnResponse(ctx context.Context, result protocol.Result, _ protocol.Invoker, _ protocol.Invocation) protocol.Result {
	if methodEntry := ctx.Value(MethodEntryKey); methodEntry != nil {
		e := methodEntry.(*base.SentinelEntry)
		sentinel.TraceError(e, result.Error())
		e.Exit()
	}
	if interfaceEntry := ctx.Value(InterfaceEntryKey); interfaceEntry != nil {
		e := interfaceEntry.(*base.SentinelEntry)
		sentinel.TraceError(e, result.Error())
		e.Exit()
	}
	return result
}

type SentinelConsumerFilter struct{}

func (d *SentinelConsumerFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	methodResourceName := getResourceName(invoker, invocation, getConsumerPrefix())
	interfaceResourceName := ""
	if getInterfaceGroupAndVersionEnabled() {
		interfaceResourceName = getColonSeparatedKey(invoker.GetUrl())
	} else {
		interfaceResourceName = invoker.GetUrl().Service()
	}
	var (
		interfaceEntry *base.SentinelEntry
		methodEntry    *base.SentinelEntry
		b              *base.BlockError
	)

	interfaceEntry, b = sentinel.Entry(interfaceResourceName, sentinel.WithResourceType(base.ResTypeRPC), sentinel.WithTrafficType(base.Outbound))
	if b != nil {
		// interface blocked
		return sentinelDubboConsumerFallback(ctx, invoker, invocation, b)
	}
	ctx = context.WithValue(ctx, InterfaceEntryKey, interfaceEntry)

	methodEntry, b = sentinel.Entry(methodResourceName, sentinel.WithResourceType(base.ResTypeRPC),
		sentinel.WithTrafficType(base.Outbound), sentinel.WithArgs(invocation.Arguments()...))
	if b != nil {
		// method blocked
		return sentinelDubboConsumerFallback(ctx, invoker, invocation, b)
	}
	ctx = context.WithValue(ctx, MethodEntryKey, methodEntry)

	return invoker.Invoke(ctx, invocation)
}

func (d *SentinelConsumerFilter) OnResponse(ctx context.Context, result protocol.Result, _ protocol.Invoker, _ protocol.Invocation) protocol.Result {
	if methodEntry := ctx.Value(MethodEntryKey); methodEntry != nil {
		e := methodEntry.(*base.SentinelEntry)
		sentinel.TraceError(e, result.Error())
		e.Exit()
	}
	if interfaceEntry := ctx.Value(InterfaceEntryKey); interfaceEntry != nil {
		e := interfaceEntry.(*base.SentinelEntry)
		sentinel.TraceError(e, result.Error())
		e.Exit()
	}
	return result
}

var (
	sentinelDubboConsumerFallback = getDefaultDubboFallback()
	sentinelDubboProviderFallback = getDefaultDubboFallback()
)

type DubboFallback func(context.Context, protocol.Invoker, protocol.Invocation, *base.BlockError) protocol.Result

func SetDubboConsumerFallback(f DubboFallback) {
	sentinelDubboConsumerFallback = f
}
func SetDubboProviderFallback(f DubboFallback) {
	sentinelDubboProviderFallback = f
}
func getDefaultDubboFallback() DubboFallback {
	return func(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation, blockError *base.BlockError) protocol.Result {
		result := &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(blockError)
		return result
	}
}

const (
	SentinelProviderFilterName = "sentinel-provider"
	SentinelConsumerFilterName = "sentinel-consumer"

	DefaultProviderPrefix = "dubbo:provider:"
	DefaultConsumerPrefix = "dubbo:consumer:"

	MethodEntryKey    = "$$sentinelMethodEntry"
	InterfaceEntryKey = "$$sentinelInterfaceEntry"
)

// Currently, a ConcurrentHashMap mechanism is missing.
// All values are filled with default values first.

func getResourceName(invoker protocol.Invoker, invocation protocol.Invocation, prefix string) string {
	var (
		buf               bytes.Buffer
		interfaceResource string
	)
	buf.WriteString(prefix)
	if getInterfaceGroupAndVersionEnabled() {
		interfaceResource = getColonSeparatedKey(invoker.GetUrl())
	} else {
		interfaceResource = invoker.GetUrl().Service()
	}
	buf.WriteString(interfaceResource)
	buf.WriteString(":")
	buf.WriteString(invocation.MethodName())
	buf.WriteString("(")
	isFirst := true
	for _, v := range invocation.ParameterTypes() {
		if !isFirst {
			buf.WriteString(",")
		}
		buf.WriteString(v.Name())
		isFirst = false
	}
	buf.WriteString(")")
	return buf.String()
}

func getConsumerPrefix() string {
	return DefaultConsumerPrefix
}

func getProviderPrefix() string {
	return DefaultProviderPrefix
}

func getInterfaceGroupAndVersionEnabled() bool {
	return true
}

func getColonSeparatedKey(url common.URL) string {
	return fmt.Sprintf("%s:%s:%s",
		url.Service(),
		url.GetParam(constant.GROUP_KEY, ""),
		url.GetParam(constant.VERSION_KEY, ""))
}
