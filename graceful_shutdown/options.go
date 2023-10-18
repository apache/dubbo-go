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

package graceful_shutdown

import (
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Shutdown *global.ShutdownConfig
}

func defaultOptions() *Options {
	return &Options{
		Shutdown: global.DefaultShutdownConfig(),
	}
}

func NewOptions(opts ...Option) *Options {
	defOpts := defaultOptions()
	for _, opt := range opts {
		opt(defOpts)
	}

	return defOpts
}

type Option func(*Options)

func WithTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Shutdown.Timeout = timeout.String()
	}
}

func WithStepTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Shutdown.StepTimeout = timeout.String()
	}
}

func WithConsumerUpdateWaitTime(duration time.Duration) Option {
	return func(opts *Options) {
		opts.Shutdown.ConsumerUpdateWaitTime = duration.String()
	}
}

// todo(DMwangnima): add more specified configuration API
//func WithRejectRequestHandler(handler string) Option {
//	return func(opts *Options) {
//		opts.Shutdown.RejectRequestHandler = handler
//	}
//}

func WithoutInternalSignal() Option {
	return func(opts *Options) {
		signal := false
		opts.Shutdown.InternalSignal = &signal
	}
}

func WithOfflineRequestWindowTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Shutdown.OfflineRequestWindowTimeout = timeout.String()
	}
}

func WithRejectRequest() Option {
	return func(opts *Options) {
		opts.Shutdown.RejectRequest.Store(true)
	}
}

// ---------- For framework ----------

func SetShutdown_Config(cfg *global.ShutdownConfig) Option {
	return func(opts *Options) {
		opts.Shutdown = cfg
	}
}
