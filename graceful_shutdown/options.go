package graceful_shutdown

import "dubbo.apache.org/dubbo-go/v3/global"

type Options struct {
	shutdown *global.ShutdownConfig
}

type Option func(*Options)

func WithShutdown_Config(cfg *global.ShutdownConfig) Option {
	return func(opts *Options) {
		opts.shutdown = cfg
	}
}
