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

package client

import (
	"net/http"
	"time"
)

type RestOptions struct {
	RequestTimeout   time.Duration
	ConnectTimeout   time.Duration
	KeppAliveTimeout time.Duration
}

// Option represents a functional option for configuring RestOptions.
type Option func(*RestOptions)

// defaultRestOptions returns RestOptions with sensible defaults.
// These defaults are only used when NewRestOptions is called.
func defaultRestOptions() *RestOptions {
	return &RestOptions{
		// Keep backward compatibility: if user doesn't specify, we leave
		// zero values here and let the underlying HTTP client decide.
		// Users can explicitly configure these via the With* helpers below.
	}
}

// NewRestOptions builds a RestOptions using the functional Options pattern.
// It starts from defaultRestOptions and then applies all provided options.
func NewRestOptions(opts ...Option) *RestOptions {
	def := defaultRestOptions()
	for _, opt := range opts {
		opt(def)
	}
	return def
}

// WithRequestTimeout sets the per-request timeout.
func WithRequestTimeout(d time.Duration) Option {
	return func(o *RestOptions) {
		o.RequestTimeout = d
	}
}

// WithConnectTimeout sets the dial/connect timeout.
func WithConnectTimeout(d time.Duration) Option {
	return func(o *RestOptions) {
		o.ConnectTimeout = d
	}
}

// WithKeepAliveTimeout sets the idle connection keep-alive timeout.
func WithKeepAliveTimeout(d time.Duration) Option {
	return func(o *RestOptions) {
		o.KeppAliveTimeout = d
	}
}

type RestClientRequest struct {
	Header      http.Header
	Location    string
	Path        string
	Method      string
	PathParams  map[string]string
	QueryParams map[string]string
	Body        any
}

// RestClient user can implement this client interface to send request
type RestClient interface {
	Do(request *RestClientRequest, res any) error
}
