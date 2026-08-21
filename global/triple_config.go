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

package global

// TODO: Find an ideal way to separate the triple config of server and client.

// TripleConfig represents the config of triple protocol.
type TripleConfig struct {
	//
	// for server
	//
	// MaxServerSendMsgSize defines the maximum size of messages sent by server.
	// Supported units include 1mb=1000kb=1000000b and 1mib=1024kb=1048576b.
	// For more details, see https://pkg.go.dev/github.com/dustin/go-humanize#pkg-constants.
	MaxServerSendMsgSize string `yaml:"max-server-send-msg-size" json:"max-server-send-msg-size,omitempty"`

	// MaxServerRecvMsgSize defines the maximum size of messages received by server.
	MaxServerRecvMsgSize string `yaml:"max-server-recv-msg-size" json:"max-server-recv-msg-size,omitempty"`

	// Http3 holds the HTTP/3 transport configuration for server and client.
	Http3 *Http3Config `yaml:"http3" json:"http3,omitempty"`

	// Cors configures CORS for Triple protocol handlers on server.
	Cors *CorsConfig `yaml:"cors" json:"cors,omitempty"`

	// OpenAPI configures OpenAPI documentation generation for server.
	OpenAPI *OpenAPIConfig `yaml:"openapi" json:"openapi,omitempty"`

	//
	// for client
	//
	// KeepAliveInterval defines the keep-alive interval for client.
	KeepAliveInterval string `yaml:"keep-alive-interval" json:"keep-alive-interval,omitempty" property:"keep-alive-interval"`

	// KeepAliveTimeout defines the keep-alive timeout for client.
	KeepAliveTimeout string `yaml:"keep-alive-timeout" json:"keep-alive-timeout,omitempty" property:"keep-alive-timeout"`
}

// DefaultTripleConfig returns a default TripleConfig instance.
func DefaultTripleConfig() *TripleConfig {
	return &TripleConfig{
		Http3:   DefaultHttp3Config(),
		Cors:    DefaultCorsConfig(),
		OpenAPI: DefaultOpenAPIConfig(),
	}
}

// Clone a new TripleConfig
func (t *TripleConfig) Clone() *TripleConfig {
	if t == nil {
		return nil
	}

	return &TripleConfig{
		MaxServerSendMsgSize: t.MaxServerSendMsgSize,
		MaxServerRecvMsgSize: t.MaxServerRecvMsgSize,
		Http3:                t.Http3.Clone(),
		Cors:                 t.Cors.Clone(),
		OpenAPI:              t.OpenAPI.Clone(),

		KeepAliveInterval: t.KeepAliveInterval,
		KeepAliveTimeout:  t.KeepAliveTimeout,
	}
}
