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

package logger

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Options struct {
	Logger *global.LoggerConfig
}

func defaultOptions() *Options {
	return &Options{Logger: global.DefaultLoggerConfig()}
}

func NewOptions(opts ...Option) *Options {
	Options := defaultOptions()
	for _, opt := range opts {
		opt(Options)
	}
	return Options
}

type Option func(*Options)

func WithLogrus() Option {
	return func(opts *Options) {
		opts.Logger.Driver = "logrus"
	}
}

func WithZap() Option {
	return func(opts *Options) {
		opts.Logger.Driver = "zap"
	}
}

func WithLevel(level string) Option {
	return func(opts *Options) {
		opts.Logger.Level = level
	}
}

func WithFormat(format string) Option {
	return func(opts *Options) {
		opts.Logger.Format = format
	}
}

func WithAppender(appender string) Option {
	return func(opts *Options) {
		opts.Logger.Appender = appender
	}
}

func WithFileName(name string) Option {
	return func(opts *Options) {
		opts.Logger.File.Name = name
	}
}

func WithFileMaxSize(size int) Option {
	return func(opts *Options) {
		opts.Logger.File.MaxSize = size
	}
}

func WithFileMaxBackups(backups int) Option {
	return func(opts *Options) {
		opts.Logger.File.MaxBackups = backups
	}
}

func WithFileMaxAge(age int) Option {
	return func(opts *Options) {
		opts.Logger.File.MaxAge = age
	}
}

func WithFileCompress() Option {
	return func(opts *Options) {
		b := true
		opts.Logger.File.Compress = &b
	}
}
