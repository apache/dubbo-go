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

// Package hystrix provides hystrix filter.
// To use hystrix, you need to configure commands using hystrix-go API:
//
//	import "github.com/afex/hystrix-go/hystrix"
//
//	// Resource name format: dubbo:consumer:InterfaceName:group:version:Method(param1,param2)
//	// Example: dubbo:consumer:com.example.GreetService:::Greet(string,string)
//	hystrix.ConfigureCommand("dubbo:consumer:com.example.GreetService:::Greet(string,string)", hystrix.CommandConfig{
//	    Timeout:                1000,
//	    MaxConcurrentRequests:  20,
//	    RequestVolumeThreshold: 20,
//	    SleepWindow:            5000,
//	    ErrorPercentThreshold:  50,
//	})
package hystrix

import (
	"context"
	"fmt"
	"strings"
)

import (
	"github.com/afex/hystrix-go/hystrix"

	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

const (
	// HYSTRIX is the key used in filter configuration.
	HYSTRIX = "hystrix"
)

func init() {
	extension.SetFilter(constant.HystrixConsumerFilterKey, newFilterConsumer)
	extension.SetFilter(constant.HystrixProviderFilterKey, newFilterProvider)
}

// FilterError implements error interface
type FilterError struct {
	err           error
	failByHystrix bool
}

func (hfError *FilterError) Error() string {
	return hfError.err.Error()
}

// FailByHystrix returns whether the fails causing by Hystrix
func (hfError *FilterError) FailByHystrix() bool {
	return hfError.failByHystrix
}

// NewHystrixFilterError return a FilterError instance
func NewHystrixFilterError(err error, failByHystrix bool) error {
	return &FilterError{
		err:           err,
		failByHystrix: failByHystrix,
	}
}

// Filter for Hystrix
type Filter struct {
	COrP bool // true for consumer, false for provider
}

// Invoke is an implementation of filter, provides Hystrix pattern latency and fault tolerance
func (f *Filter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	cmdName := getResourceName(invoker, invocation, f.COrP)

	var res result.Result
	err := hystrix.Do(cmdName, func() error {
		res = invoker.Invoke(ctx, invocation)
		return res.Error()
	}, func(err error) error {
		// Circuit is open, return fallback error
		_, ok := err.(hystrix.CircuitError)
		logger.Debugf("[Hystrix Filter] Circuit opened for %s, failed by hystrix: %v", cmdName, ok)
		res = &result.RPCResult{}
		res.SetResult(nil)
		res.SetError(NewHystrixFilterError(err, ok))
		return err
	})

	if err != nil {
		return res
	}
	return res
}

// OnResponse dummy process, returns the result directly
func (f *Filter) OnResponse(ctx context.Context, result result.Result, invoker base.Invoker, invocation base.Invocation) result.Result {
	return result
}

// newFilterConsumer returns Filter instance for consumer
func newFilterConsumer() filter.Filter {
	return &Filter{COrP: true}
}

// newFilterProvider returns Filter instance for provider
func newFilterProvider() filter.Filter {
	return &Filter{COrP: false}
}

const (
	DefaultProviderPrefix = "dubbo:provider:"
	DefaultConsumerPrefix = "dubbo:consumer:"
)

func getResourceName(invoker base.Invoker, invocation base.Invocation, isConsumer bool) string {
	var sb strings.Builder

	if isConsumer {
		sb.WriteString(DefaultConsumerPrefix)
	} else {
		sb.WriteString(DefaultProviderPrefix)
	}

	// Format: interface:group:version
	sb.WriteString(getColonSeparatedKey(invoker.GetURL()))
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

	return sb.String()
}

func getColonSeparatedKey(url *common.URL) string {
	return fmt.Sprintf("%s:%s:%s",
		url.Service(),
		url.GetParam(constant.GroupKey, ""),
		url.GetParam(constant.VersionKey, ""))
}
