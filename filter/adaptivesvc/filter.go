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

// Package adaptivesvc providers AdaptiveService filter.
package adaptivesvc

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	adaptiveServiceProviderFilterOnce sync.Once
	instance                          filter.Filter

	ErrAdaptiveSvcInterrupted = fmt.Errorf("adaptive service interrupted")
	ErrUpdaterNotFound        = fmt.Errorf("updater not found")
	ErrUnexpectedUpdaterType  = fmt.Errorf("unexpected updater type")
)

func init() {
	extension.SetFilter(constant.AdaptiveServiceProviderFilterKey, newAdaptiveServiceProviderFilter)
}

// adaptiveServiceProviderFilter is for adaptive service on the provider side.
type adaptiveServiceProviderFilter struct{}

func newAdaptiveServiceProviderFilter() filter.Filter {
	if instance == nil {
		adaptiveServiceProviderFilterOnce.Do(func() {
			instance = &adaptiveServiceProviderFilter{}
		})
	}
	return instance
}

func (f *adaptiveServiceProviderFilter) Invoke(ctx context.Context, invoker protocol.Invoker,
	invocation protocol.Invocation) protocol.Result {
	if invocation.GetAttachmentWithDefaultValue(constant.AdaptiveServiceEnabledKey, "") !=
		constant.AdaptiveServiceIsEnabled {
		// the adaptive service is enabled on the invocation
		return invoker.Invoke(ctx, invocation)
	}

	l, err := limiterMapperSingleton.getMethodLimiter(invoker.GetURL(), invocation.MethodName())
	if err != nil {
		if errors.Is(err, ErrLimiterNotFoundOnMapper) {
			// limiter is not found on the mapper, just create
			// a new limiter
			if l, err = limiterMapperSingleton.newAndSetMethodLimiter(invoker.GetURL(),
				invocation.MethodName(), limiter.HillClimbingLimiter); err != nil {
				return &protocol.RPCResult{Err: wrapErrAdaptiveSvcInterrupted(err)}
			}
		} else {
			// unexpected errors
			return &protocol.RPCResult{Err: wrapErrAdaptiveSvcInterrupted(err)}
		}
	}

	updater, err := l.Acquire()
	if err != nil {
		return &protocol.RPCResult{Err: wrapErrAdaptiveSvcInterrupted(err)}
	}

	invocation.SetAttribute(constant.AdaptiveServiceUpdaterKey, updater)
	return invoker.Invoke(ctx, invocation)
}

func (f *adaptiveServiceProviderFilter) OnResponse(_ context.Context, result protocol.Result, invoker protocol.Invoker,
	invocation protocol.Invocation) protocol.Result {
	var asEnabled string
	asEnabledIface := result.Attachment(constant.AdaptiveServiceEnabledKey, nil)
	if asEnabledIface != nil {
		if str, strOK := asEnabledIface.(string); strOK {
			asEnabled = str
		} else if strArr, strArrOK := asEnabledIface.([]string); strArrOK && len(strArr) > 0 {
			asEnabled = strArr[0]
		}
	}
	if asEnabled != constant.AdaptiveServiceIsEnabled {
		// the adaptive service is enabled on the invocation
		return result
	}

	if isErrAdaptiveSvcInterrupted(result.Error()) {
		// If the Invoke method of the adaptiveServiceProviderFilter returns an error,
		// the OnResponse of the adaptiveServiceProviderFilter should not be performed.
		return result
	}

	// get updater from the attributes
	updaterIface, _ := invocation.GetAttribute(constant.AdaptiveServiceUpdaterKey)
	if updaterIface == nil {
		logger.Errorf("[adasvc filter] The updater is not found on the attributes: %#v",
			invocation.Attributes())
		return &protocol.RPCResult{Err: ErrUpdaterNotFound}
	}
	updater, ok := updaterIface.(limiter.Updater)
	if !ok {
		logger.Errorf("[adasvc filter] The type of the updater is not unexpected, we got %#v", updaterIface)
		return &protocol.RPCResult{Err: ErrUnexpectedUpdaterType}
	}

	err := updater.DoUpdate()
	if err != nil {
		logger.Errorf("[adasvc filter] The DoUpdate method was failed, err: %s.", err)
		return &protocol.RPCResult{Err: err}
	}

	// get limiter for the mapper
	l, err := limiterMapperSingleton.getMethodLimiter(invoker.GetURL(), invocation.MethodName())
	if err != nil {
		logger.Errorf("[adasvc filter] The method limiter for \"%s\" is not found.", invocation.MethodName())
		return &protocol.RPCResult{Err: err}
	}

	// set attachments to inform consumer of provider status
	result.AddAttachment(constant.AdaptiveServiceRemainingKey, fmt.Sprintf("%d", l.Remaining()))
	result.AddAttachment(constant.AdaptiveServiceInflightKey, fmt.Sprintf("%d", l.Inflight()))
	logger.Debugf("[adasvc filter] The attachments are set, %s: %d, %s: %d.",
		constant.AdaptiveServiceRemainingKey, l.Remaining(),
		constant.AdaptiveServiceInflightKey, l.Inflight())

	return result
}

func wrapErrAdaptiveSvcInterrupted(customizedErr interface{}) error {
	return fmt.Errorf("%w: %v", ErrAdaptiveSvcInterrupted, customizedErr)
}

func isErrAdaptiveSvcInterrupted(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), ErrAdaptiveSvcInterrupted.Error())
}
