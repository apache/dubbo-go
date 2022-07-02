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

// Package sentinel provides a filter when using sentinel.
// Integrate Sentinel Go MUST HAVE:
// 1. Must initialize Sentinel Go run environment, refer to https://github.com/alibaba/sentinel-golang/blob/master/api/init.go
// 2. Register rules for resources user want to guard
package sentinel

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

import (
	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/logging"

	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func init() {
	extension.SetFilter(constant.SentinelConsumerFilterKey, newSentinelConsumerFilter)
	extension.SetFilter(constant.SentinelProviderFilterKey, newSentinelProviderFilter)
	if err := logging.ResetGlobalLogger(DubboLoggerWrapper{Logger: logger.GetLogger()}); err != nil {
		logger.Errorf("[Sentinel Filter] fail to ingest dubbo logger into sentinel")
	}
}

type DubboLoggerWrapper struct {
	logger.Logger
}

func (d DubboLoggerWrapper) Debug(msg string, keysAndValues ...interface{}) {
	d.Logger.Debug(logging.AssembleMsg(logging.GlobalCallerDepth, "DEBUG", msg, nil, keysAndValues))
}

func (d DubboLoggerWrapper) DebugEnabled() bool {
	return true
}

func (d DubboLoggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	d.Logger.Info(logging.AssembleMsg(logging.GlobalCallerDepth, "INFO", msg, nil, keysAndValues))
}

func (d DubboLoggerWrapper) InfoEnabled() bool {
	return true
}

func (d DubboLoggerWrapper) Warn(msg string, keysAndValues ...interface{}) {
	d.Logger.Warn(logging.AssembleMsg(logging.GlobalCallerDepth, "WARN", msg, nil, keysAndValues))
}

func (d DubboLoggerWrapper) WarnEnabled() bool {
	return true
}

func (d DubboLoggerWrapper) Error(err error, msg string, keysAndValues ...interface{}) {
	d.Logger.Warn(logging.AssembleMsg(logging.GlobalCallerDepth, "ERROR", msg, err, keysAndValues))
}

func (d DubboLoggerWrapper) ErrorEnabled() bool {
	return true
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

var (
	providerOnce     sync.Once
	sentinelProvider *sentinelProviderFilter
)

type sentinelProviderFilter struct{}

func newSentinelProviderFilter() filter.Filter {
	if sentinelProvider == nil {
		providerOnce.Do(func() {
			sentinelProvider = &sentinelProviderFilter{}
		})
	}
	return sentinelProvider
}

func (d *sentinelProviderFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
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

func (d *sentinelProviderFilter) OnResponse(ctx context.Context, result protocol.Result, _ protocol.Invoker, _ protocol.Invocation) protocol.Result {
	sentinelExit(ctx, result)
	return result
}

var (
	consumerOnce     sync.Once
	sentinelConsumer *sentinelConsumerFilter
)

type sentinelConsumerFilter struct{}

func newSentinelConsumerFilter() filter.Filter {
	if sentinelConsumer == nil {
		consumerOnce.Do(func() {
			sentinelConsumer = &sentinelConsumerFilter{}
		})
	}
	return sentinelConsumer
}

func (d *sentinelConsumerFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
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

func (d *sentinelConsumerFilter) OnResponse(ctx context.Context, result protocol.Result, _ protocol.Invoker, _ protocol.Invocation) protocol.Result {
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
		return &protocol.RPCResult{Err: blockError}
	}
}

const (
	DefaultProviderPrefix = "dubbo:provider:"
	DefaultConsumerPrefix = "dubbo:consumer:"

	MethodEntryKey    = constant.DubboCtxKey("$$sentinelMethodEntry")
	InterfaceEntryKey = constant.DubboCtxKey("$$sentinelInterfaceEntry")
)

func getResourceName(invoker protocol.Invoker, invocation protocol.Invocation, prefix string) (interfaceResourceName, methodResourceName string) {
	var sb strings.Builder

	sb.WriteString(prefix)
	if getInterfaceGroupAndVersionEnabled() {
		interfaceResourceName = getColonSeparatedKey(invoker.GetURL())
	} else {
		interfaceResourceName = invoker.GetURL().Service()
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
		url.GetParam(constant.GroupKey, ""),
		url.GetParam(constant.VersionKey, ""))
}
