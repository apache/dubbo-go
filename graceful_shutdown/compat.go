package graceful_shutdown

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"go.uber.org/atomic"
)

func compatShutdownConfig(c *global.ShutdownConfig) *config.ShutdownConfig {
	if c == nil {
		return nil
	}
	cfg := &config.ShutdownConfig{
		Timeout:                     c.Timeout,
		StepTimeout:                 c.StepTimeout,
		ConsumerUpdateWaitTime:      c.ConsumerUpdateWaitTime,
		RejectRequestHandler:        c.RejectRequestHandler,
		InternalSignal:              c.InternalSignal,
		OfflineRequestWindowTimeout: c.OfflineRequestWindowTimeout,
		RejectRequest:               atomic.Bool{},
	}
	cfg.RejectRequest.Store(c.RejectRequest.Load())
	return cfg
}
