package config

import (
	"reflect"
)

type LoaderHook interface {
	emit() bool
	emitWithParams(params interface{}) bool
}

func getLoaderHookType(hook LoaderHook) string {
	return reflect.TypeOf(hook).Elem().String()
}

func getLoaderHookParams(t LoaderHook) interface{} {
	return reflect.ValueOf(t).Elem().FieldByName("HookParams").Elem().Interface()
}

func AddLoaderHooks(hooks ...LoaderHook) {
	if len(hooks) > 0 {
		if loaderHooks == nil {
			loaderHooks = map[string][]LoaderHook{}
		}
		for _, hook := range hooks {
			hookType := getLoaderHookType(hook)
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
			hookType := getLoaderHookType(rmHook)
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
		hookType := getLoaderHookType(t)
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
