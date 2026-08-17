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

package router

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// Options contains the router configuration built by router options.
type Options struct {
	// Router holds the configuration modified by Option values.
	Router *global.RouterConfig
}

func defaultOptions() *Options {
	return &Options{
		Router: global.DefaultRouterConfig(),
	}
}

// NewOptions returns router options initialized with the default router configuration.
func NewOptions(opts ...Option) *Options {
	defOpts := defaultOptions()
	for _, opt := range opts {
		opt(defOpts)
	}
	return defOpts
}

// Option modifies router options.
type Option func(*Options)

// WithScope sets the rule scope, such as service or application.
func WithScope(scope string) Option {
	return func(opts *Options) {
		opts.Router.Scope = scope
	}
}

// WithKey sets the service or application key to which the rule applies.
func WithKey(key string) Option {
	return func(opts *Options) {
		opts.Router.Key = key
	}
}

// WithForce sets whether the rule should be enforced when it produces no matching provider.
func WithForce(force bool) Option {
	return func(opts *Options) {
		opts.Router.Force = &force
	}
}

// WithRuntime sets whether the rule is evaluated at runtime.
func WithRuntime(runtime bool) Option {
	return func(opts *Options) {
		opts.Router.Runtime = &runtime
	}
}

// WithEnabled sets whether the router rule is enabled.
func WithEnabled(enabled bool) Option {
	return func(opts *Options) {
		opts.Router.Enabled = &enabled
	}
}

// WithValid records whether the router rule passed validation.
func WithValid(valid bool) Option {
	return func(opts *Options) {
		opts.Router.Valid = &valid
	}
}

// WithPriority sets the rule priority. Lower values run before higher values.
func WithPriority(priority int) Option {
	return func(opts *Options) {
		opts.Router.Priority = priority
	}
}

// WithConditions sets the condition expressions used by a condition router.
func WithConditions(conditions []string) Option {
	return func(opts *Options) {
		opts.Router.Conditions = conditions
	}
}

// WithTags sets the tag definitions used by a tag router.
func WithTags(tags []global.Tag) Option {
	return func(opts *Options) {
		opts.Router.Tags = tags
	}
}

// WithScript sets the script body used by a script router.
func WithScript(script string) Option {
	return func(opts *Options) {
		opts.Router.Script = script
	}
}

// WithScriptType sets the script language used to evaluate the script body.
func WithScriptType(scriptType string) Option {
	return func(opts *Options) {
		opts.Router.ScriptType = scriptType
	}
}
