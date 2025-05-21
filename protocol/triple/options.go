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

package triple

type ServerOptions struct {
	HTTP3 bool
}

func defaultServerOptions() *ServerOptions {
	return &ServerOptions{
		HTTP3: false,
	}
}

func (srvOpts *ServerOptions) init(opts ...ServerOption) {
	for _, opt := range opts {
		opt(srvOpts)
	}
}

type ServerOption func(*ServerOptions)

func WithServerHTTP3() ServerOption {
	return func(opts *ServerOptions) {
		opts.HTTP3 = true
	}
}

