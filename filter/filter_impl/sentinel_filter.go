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
	"context"
	"fmt"
	"strings"
)

import (
	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/logging"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

func init() {
	extension.SetFilter(SentinelProviderFilterName, GetSentinelProviderFilter)
	extension.SetFilter(SentinelConsumerFilterName, GetSentinelConsumerFilter)

	if err := sentinel.InitDefault(); err != nil {
		logger.Errorf("[Sentinel Filter] fail to initialize Sentinel")
	}
	if err := logging.ResetGlobalLogger(DubboLoggerWrapper{Logger: logger.GetLogger()}); err != nil {
		logger.Errorf("[Sentinel Filter] fail to ingest dubbo logger into sentinel")
	}
}

type DubboLoggerWrapper struct {
	logger.Logger
}

func (d DubboLoggerWrapper) Fatal(v ...interface{}) {
	d.Logger.Error(v...)
}

func (d DubboLoggerWrapper) Fatalf(format string, v ...interface{}) {
	d.Logger.Errorf(format, v...)
}

func (d DubboLoggerWrapper) Panic(v ...interface{}) {
	d.Logger.Error(v...)
}

func (d DubboLoggerWrapper) Panicf(format string, v ...interface{}) {
	d.Logger.Errorf(format, v...)
}

func GetSentinelConsumerFilter() filter.Filter {
	return &SentinelConsumerFilter{}
}

func GetSentinelProviderFilter() filter.Filter {
	return &SentinelProviderFilter{}
}

func sentinelExit(ctx context.Context, result protocol.Result) {
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
}

type SentinelProviderFilter struct{}

func (d *SentinelProviderFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	interfaceResourceName, methodResourceName := getResourceName(invoker, invocation, getProviderPrefix())

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

	methodEntry, b = sentinel.Entry(methodResourceName,
		sentinel.WithResourceType(base.ResTypeRPC),
		sentinel.WithTrafficType(base.Inbound),
		sentinel.WithArgs(invocation.Arguments()...))
	if b != nil {
		// method blocked
		return sentinelDubboProviderFallback(ctx, invoker, invocation, b)
	}
	ctx = context.WithValue(ctx, MethodEntryKey, methodEntry)
	return invoker.Invoke(ctx, invocation)
}

func (d *SentinelProviderFilter) OnResponse(ctx context.Context, result protocol.Result, _ protocol.Invoker, _ protocol.Invocation) protocol.Result {
	sentinelExit(ctx, result)
	return result
}

type SentinelConsumerFilter struct{}

func (d *SentinelConsumerFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	interfaceResourceName, methodResourceName := getResourceName(invoker, invocation, getConsumerPrefix())
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
	sentinelExit(ctx, result)
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

func getResourceName(invoker protocol.Invoker, invocation protocol.Invocation, prefix string) (interfaceResourceName, methodResourceName string) {
	var sb strings.Builder

	sb.WriteString(prefix)
	if getInterfaceGroupAndVersionEnabled() {
		interfaceResourceName = getColonSeparatedKey(invoker.GetUrl())
	} else {
		interfaceResourceName = invoker.GetUrl().Service()
	}
	sb.WriteString(interfaceResourceName)
	sb.WriteString(":")
	sb.WriteString(invocation.MethodName())
	sb.WriteString("(")
	isFirst := true
	for _, v := range invocation.ParameterTypes() {
		if !isFirst {
			sb.WriteString(",")
		}
		sb.WriteString(v.Name())
		isFirst = false
	}
	sb.WriteString(")")
	methodResourceName = sb.String()
	return
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

func getColonSeparatedKey(url *common.URL) string {
	return fmt.Sprintf("%s:%s:%s",
		url.Service(),
		url.GetParam(constant.GROUP_KEY, ""),
		url.GetParam(constant.VERSION_KEY, ""))
}
