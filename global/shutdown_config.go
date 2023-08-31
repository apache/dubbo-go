package global

import (
	"github.com/creasty/defaults"
	"go.uber.org/atomic"
)

// ShutdownConfig is used as configuration for graceful shutdown
type ShutdownConfig struct {
	/*
	 * Total timeout. Even though we don't release all resources,
	 * the applicationConfig will shutdown if the costing time is over this configuration. The unit is ms.
	 * default value is 60 * 1000 ms = 1 minutes
	 * In general, it should be bigger than 3 * StepTimeout.
	 */
	Timeout string `default:"60s" yaml:"timeout" json:"timeout,omitempty" property:"timeout"`
	/*
	 * the timeout on each step. You should evaluate the response time of request
	 * and the time that client noticed that server shutdown.
	 * For example, if your client will received the notification within 10s when you start to close server,
	 * and the 99.9% requests will return response in 2s, so the StepTimeout will be bigger than(10+2) * 1000ms,
	 * maybe (10 + 2*3) * 1000ms is a good choice.
	 */
	StepTimeout string `default:"3s" yaml:"step-timeout" json:"step.timeout,omitempty" property:"step.timeout"`

	/*
	 * ConsumerUpdateWaitTime means when provider is shutting down, after the unregister, time to wait for client to
	 * update invokers. During this time, incoming invocation can be treated normally.
	 */
	ConsumerUpdateWaitTime string `default:"3s" yaml:"consumer-update-wait-time" json:"consumerUpdate.waitTIme,omitempty" property:"consumerUpdate.waitTIme"`
	// when we try to shutdown the applicationConfig, we will reject the new requests. In most cases, you don't need to configure this.
	RejectRequestHandler string `yaml:"reject-handler" json:"reject-handler,omitempty" property:"reject_handler"`
	// internal listen kill signal，the default is true.
	InternalSignal *bool `default:"true" yaml:"internal-signal" json:"internal.signal,omitempty" property:"internal.signal"`
	// offline request window length
	OfflineRequestWindowTimeout string `yaml:"offline-request-window-timeout" json:"offlineRequestWindowTimeout,omitempty" property:"offlineRequestWindowTimeout"`
	// true -> new request will be rejected.
	RejectRequest atomic.Bool
	// active invocation
	ConsumerActiveCount atomic.Int32
	ProviderActiveCount atomic.Int32

	// provider last received request timestamp
	ProviderLastReceivedRequestTime atomic.Time
}

func DefaultShutdownConfig() *ShutdownConfig {
	cfg := &ShutdownConfig{}
	defaults.MustSet(cfg)

	return cfg
}

type ShutdownOption func(*ShutdownConfig)

// ---------- ShutdownOption ----------

func WithShutdown_Timeout(timeout string) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.Timeout = timeout
	}
}

func WithShutdown_StepTimeout(timeout string) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.StepTimeout = timeout
	}
}

func WithShutdown_ConsumerUpdateWaitTime(duration string) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.ConsumerUpdateWaitTime = duration
	}
}

func WithShutdown_RejectRequestHandler(handler string) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.RejectRequestHandler = handler
	}
}

func WithShutdown_InternalSignal(signal bool) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.InternalSignal = &signal
	}
}

func WithShutdown_OfflineRequestWindowTimeout(timeout string) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.OfflineRequestWindowTimeout = timeout
	}
}

func WithShutdown_RejectRequest(flag bool) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.RejectRequest.Store(flag)
	}
}
