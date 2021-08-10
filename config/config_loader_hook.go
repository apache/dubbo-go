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

package config

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"fmt"
	"reflect"
)

const (
	hookParams = constant.HookParams
)

type LoaderHook interface {
	emit() bool
	// if you want to use emitWithParams, you must add HookParams field and make emit return false
	emitWithParams(params interface{}) bool
}

func getLoaderHookType(t LoaderHook) reflect.Type {
	hookType := reflect.TypeOf(t)
	if reflect.Ptr == hookType.Kind() {
		hookType = hookType.Elem()
	}
	return hookType
}

func getLoaderHookTypeString(t LoaderHook) string {
	return getLoaderHookType(t).String()
}

func getLoaderHookInfo(v reflect.Value) reflect.Value {
	if reflect.Ptr == v.Kind() {
		v = v.Elem()
	}
	return v
}

func getLoaderHookParams(t LoaderHook) interface{} {
	hookType := getLoaderHookType(t)
	_, existed := hookType.FieldByName(hookParams)
	if !existed {
		panic(fmt.Sprintf("Hook field [%s] not existed for %s", hookParams, hookType.Name()))
	}
	hookParams := getLoaderHookInfo(reflect.ValueOf(t)).FieldByName(hookParams)
	return getLoaderHookInfo(hookParams).Interface()
}

func AddLoaderHooks(hooks ...LoaderHook) {
	if len(hooks) > 0 {
		if loaderHooks == nil {
			loaderHooks = map[string][]LoaderHook{}
		}
		for _, hook := range hooks {
			hookType := getLoaderHookTypeString(hook)
			var hooks []LoaderHook
			if hs, ok := loaderHooks[hookType]; ok {
				hooks = hs
			}
			if hooks == nil {
				hooks = []LoaderHook{}
			}
			loaderHooks[hookType] = append(hooks, hook)
		}
	}
}

func RemoveLoaderHooks(hooks ...LoaderHook) {
	if loaderHooks != nil && len(loaderHooks) > 0 {
		for _, rmHook := range hooks {
			hookType := getLoaderHookTypeString(rmHook)
			if hooks, ok := loaderHooks[hookType]; ok {
				for index, targetHook := range hooks {
					if rmHook == targetHook {
						newHooks := append(hooks[:index], hooks[index+1:]...)
						if len(newHooks) == 0 {
							delete(loaderHooks, hookType)
						} else {
							loaderHooks[hookType] = newHooks
						}
						break
					}
				}
			}
		}
	}
}

func emitHook(t LoaderHook) {
	if loaderHooks != nil && len(loaderHooks) > 0 {
		hookType := getLoaderHookTypeString(t)
		if hooks, ok := loaderHooks[hookType]; ok {
			for _, hook := range hooks {
				if !hook.emit() {
					hook.emitWithParams(getLoaderHookParams(t))
				}
			}
		}
	}
}

// emit on before shutdown
type BeforeShutdownHookEvent func()

type beforeShutdownHook struct {
	onEvent func()
}

func (h *beforeShutdownHook) emit() bool {
	h.onEvent()
	return true
}

func (h *beforeShutdownHook) emitWithParams(params interface{}) bool {
	return params == nil
}

func NewBeforeShutdownHook(onEvent BeforeShutdownHookEvent) *beforeShutdownHook {
	return &beforeShutdownHook{
		onEvent,
	}
}

// Consumer Hooks
// emit on before export consumer
type BeforeConsumerConnectHookEvent func(info *ConsumerConnectInfo)

// emit on consumer export success
type ConsumerConnectSuccessHookEvent func(info *ConsumerConnectInfo)

// emit on consumer export fail
type ConsumerConnectFailHookEvent func(info *ConsumerConnectFailInfo)

// emit on all consumers export complete
type AllConsumersConnectCompleteHookEvent func()

type beforeConsumerConnectHook struct {
	onEvent BeforeConsumerConnectHookEvent

	HookParams *ConsumerConnectInfo
}

type consumerConnectSuccessHook struct {
	onEvent ConsumerConnectSuccessHookEvent

	HookParams *ConsumerConnectInfo
}

type consumerConnectFailHook struct {
	onEvent ConsumerConnectFailHookEvent

	HookParams *ConsumerConnectFailInfo
}

type allConsumersConnectCompleteHook struct {
	onEvent func()
}

type ConsumerConnectInfo struct {
	id            string
	protocol      string
	registry      string
	interfaceName string
	group         string
	version       string
}

type ConsumerConnectFailInfo struct {
	ConsumerConnectInfo

	message string
}

func (h *beforeConsumerConnectHook) emit() bool {
	return false
}

func (h *beforeConsumerConnectHook) emitWithParams(params interface{}) bool {
	info := params.(ConsumerConnectInfo)
	h.onEvent(&info)
	return true
}

func (h *consumerConnectSuccessHook) emit() bool {
	return false
}

func (h *consumerConnectSuccessHook) emitWithParams(params interface{}) bool {
	info := params.(ConsumerConnectInfo)
	h.onEvent(&info)
	return true
}

func (h *consumerConnectFailHook) emit() bool {
	return false
}

func (h *consumerConnectFailHook) emitWithParams(params interface{}) bool {
	info := params.(ConsumerConnectFailInfo)
	h.onEvent(&info)
	return true
}

func (h *allConsumersConnectCompleteHook) emit() bool {
	h.onEvent()
	return true
}

func (h *allConsumersConnectCompleteHook) emitWithParams(params interface{}) bool {
	return params == nil
}

func NewBeforeConsumerConnectHook(onEvent BeforeConsumerConnectHookEvent) *beforeConsumerConnectHook {
	hook := &beforeConsumerConnectHook{}
	hook.onEvent = onEvent
	return hook
}

func NewConsumerConnectSuccessHook(onEvent ConsumerConnectSuccessHookEvent) *consumerConnectSuccessHook {
	hook := &consumerConnectSuccessHook{}
	hook.onEvent = onEvent
	return hook
}

func NewConsumerConnectFailHook(onEvent ConsumerConnectFailHookEvent) *consumerConnectFailHook {
	hook := &consumerConnectFailHook{}
	hook.onEvent = onEvent
	return hook
}

func NewAllConsumersConnectCompleteHook(onEvent AllConsumersConnectCompleteHookEvent) *allConsumersConnectCompleteHook {
	return &allConsumersConnectCompleteHook{
		onEvent,
	}
}

// Provider Hooks
// emit on before export provider
type BeforeProviderConnectHookEvent func(info *ProviderConnectInfo)

// emit on provider export success
type ProviderConnectSuccessHookEvent func(info *ProviderConnectInfo)

// emit on provider export fail
type ProviderConnectFailHookEvent func(info *ProviderConnectFailInfo)

// emit on all providers export complete
type AllProvidersConnectCompleteHookEvent func()

type beforeProviderConnectHook struct {
	onEvent BeforeProviderConnectHookEvent

	HookParams *ProviderConnectInfo
}

type providerConnectSuccessHook struct {
	onEvent ProviderConnectSuccessHookEvent

	HookParams *ProviderConnectInfo
}

type providerConnectFailHook struct {
	onEvent ProviderConnectFailHookEvent

	HookParams *ProviderConnectFailInfo
}

type allProvidersConnectCompleteHook struct {
	onEvent func()
}

type ProviderConnectInfo struct {
	id            string
	protocol      string
	registry      string
	interfaceName string
	group         string
	version       string
}

type ProviderConnectFailInfo struct {
	ProviderConnectInfo

	message string
}

func (h *beforeProviderConnectHook) emit() bool {
	return false
}

func (h *beforeProviderConnectHook) emitWithParams(params interface{}) bool {
	info := params.(ProviderConnectInfo)
	h.onEvent(&info)
	return true
}

func (h *providerConnectSuccessHook) emit() bool {
	return false
}

func (h *providerConnectSuccessHook) emitWithParams(params interface{}) bool {
	info := params.(ProviderConnectInfo)
	h.onEvent(&info)
	return true
}

func (h *providerConnectFailHook) emit() bool {
	return false
}

func (h *providerConnectFailHook) emitWithParams(params interface{}) bool {
	info := params.(ProviderConnectFailInfo)
	h.onEvent(&info)
	return true
}

func (h *allProvidersConnectCompleteHook) emit() bool {
	h.onEvent()
	return true
}

func (h *allProvidersConnectCompleteHook) emitWithParams(params interface{}) bool {
	return params == nil
}

func NewBeforeProviderConnectHook(onEvent BeforeProviderConnectHookEvent) *beforeProviderConnectHook {
	hook := &beforeProviderConnectHook{}
	hook.onEvent = onEvent
	return hook
}

func NewProviderConnectSuccessHook(onEvent ProviderConnectSuccessHookEvent) *providerConnectSuccessHook {
	hook := &providerConnectSuccessHook{}
	hook.onEvent = onEvent
	return hook
}

func NewProviderConnectFailHook(onEvent ProviderConnectFailHookEvent) *providerConnectFailHook {
	hook := &providerConnectFailHook{}
	hook.onEvent = onEvent
	return hook
}

func NewAllProvidersConnectCompleteHook(onEvent AllProvidersConnectCompleteHookEvent) *allProvidersConnectCompleteHook {
	return &allProvidersConnectCompleteHook{
		onEvent,
	}
}
