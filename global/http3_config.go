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

// Http3Config represents the config of http3 protocol.
type Http3Config struct {
	// Enable defines whether HTTP/3 support is enabled for server and client.
	// When true, a valid TLS configuration is required; otherwise, initialization fails.
	// With valid TLS, server starts both HTTP/2 and HTTP/3,
	// and client uses a dual HTTP/2 and HTTP/3 transport.
	// When false, server starts only HTTP/2, and client uses an HTTP/2 transport.
	// The default value is false.
	Enable bool `yaml:"enable" json:"enable,omitempty"`

	// Negotiation defines whether HTTP/3 negotiation is enabled for server.
	// When true, HTTP/3 is advertised through Alt-Svc,
	// allowing clients to negotiate between HTTP/2 and HTTP/3.
	// When false, HTTP/3 is not advertised through Alt-Svc.
	// The default value is true.
	// For more details, see https://quic-go.net/docs/http3/server/#advertising-http3-via-alt-svc.
	Negotiation bool `yaml:"negotiation" json:"negotiation,omitempty"`

	// KeepAlivePeriod defines how often keep-alive packets are sent for server and client.
	KeepAlivePeriod string `yaml:"keep-alive-period" json:"keep-alive-period,omitempty"`

	// MaxIdleTimeout defines the maximum idle timeout for QUIC connections on server and client.
	MaxIdleTimeout string `yaml:"max-idle-timeout" json:"max-idle-timeout,omitempty"`

	// MaxIncomingStreams defines the maximum number of concurrent bidirectional streams accepted by server and client.
	MaxIncomingStreams int64 `yaml:"max-incoming-streams" json:"max-incoming-streams,omitempty"`

	// MaxIncomingUniStreams defines the maximum number of concurrent unidirectional streams accepted by server and client.
	MaxIncomingUniStreams int64 `yaml:"max-incoming-uni-streams" json:"max-incoming-uni-streams,omitempty"`
}

// DefaultHttp3Config returns a default Http3Config instance.
func DefaultHttp3Config() *Http3Config {
	return &Http3Config{
		Enable:                false,
		Negotiation:           true,
		KeepAlivePeriod:       "",
		MaxIdleTimeout:        "",
		MaxIncomingStreams:    0,
		MaxIncomingUniStreams: 0,
	}
}

// Clone a new Http3Config
func (t *Http3Config) Clone() *Http3Config {
	if t == nil {
		return nil
	}

	return &Http3Config{
		Enable:                t.Enable,
		Negotiation:           t.Negotiation,
		KeepAlivePeriod:       t.KeepAlivePeriod,
		MaxIdleTimeout:        t.MaxIdleTimeout,
		MaxIncomingStreams:    t.MaxIncomingStreams,
		MaxIncomingUniStreams: t.MaxIncomingUniStreams,
	}
}
