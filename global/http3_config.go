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

import "encoding/json"

// Http3Config represents the config of http3
type Http3Config struct {
	// Whether to enable HTTP/3 support.
	// When set to true, both HTTP/2 and HTTP/3 servers will be started simultaneously.
	// When set to false, only HTTP/2 server will be started.
	// The default value is false.
	Enable bool `yaml:"enable" json:"enable,omitempty"`

	// Whether to enable HTTP/3 negotiation.
	// If set to true, HTTP/2 alt-svc negotiation will be enabled,
	// allowing clients to negotiate between HTTP/2 and HTTP/3.
	// If set to false, HTTP/2 alt-svc negotiation will be skipped,
	// Clients cannot discover HTTP/3 via Alt-Svc.
	// The default value is true.
	// ref: https://quic-go.net/docs/http3/server/#advertising-http3-via-alt-svc
	Negotiation bool `yaml:"negotiation" json:"negotiation,omitempty"`

	// KeepAlivePeriod defines how often to send keep-alive packets.
	KeepAlivePeriod string `yaml:"keep-alive-period" json:"keep-alive-period,omitempty"`

	// MaxIdleTimeout defines the maximum idle timeout for QUIC connections.
	MaxIdleTimeout string `yaml:"max-idle-timeout" json:"max-idle-timeout,omitempty"`

	// MaxIncomingStreams defines the maximum number of concurrent bidirectional streams.
	MaxIncomingStreams int64 `yaml:"max-incoming-streams" json:"max-incoming-streams,omitempty"`

	// MaxIncomingUniStreams defines the maximum number of concurrent unidirectional streams.
	MaxIncomingUniStreams int64 `yaml:"max-incoming-uni-streams" json:"max-incoming-uni-streams,omitempty"`
}

func (t *Http3Config) UnmarshalJSON(data []byte) error {
	type canonicalJSON struct {
		Enable                *bool   `json:"enable"`
		Negotiation           *bool   `json:"negotiation"`
		KeepAlivePeriod       *string `json:"keep-alive-period"`
		MaxIdleTimeout        *string `json:"max-idle-timeout"`
		MaxIncomingStreams    *int64  `json:"max-incoming-streams"`
		MaxIncomingUniStreams *int64  `json:"max-incoming-uni-streams"`
	}
	type compatJSON struct {
		KeepAlivePeriod       *string `json:"keepAlivePeriod"`
		MaxIdleTimeout        *string `json:"maxIdleTimeout"`
		MaxIncomingStreams    *int64  `json:"maxIncomingStreams"`
		MaxIncomingUniStreams *int64  `json:"maxIncomingUniStreams"`
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var canonical canonicalJSON
	if err := json.Unmarshal(data, &canonical); err != nil {
		return err
	}

	var compat compatJSON
	if err := json.Unmarshal(data, &compat); err != nil {
		return err
	}

	if canonical.Enable != nil {
		t.Enable = *canonical.Enable
	}
	if canonical.Negotiation != nil {
		t.Negotiation = *canonical.Negotiation
	}
	if _, ok := raw["keep-alive-period"]; ok {
		if canonical.KeepAlivePeriod != nil {
			t.KeepAlivePeriod = *canonical.KeepAlivePeriod
		}
	} else if compat.KeepAlivePeriod != nil {
		t.KeepAlivePeriod = *compat.KeepAlivePeriod
	}
	if _, ok := raw["max-idle-timeout"]; ok {
		if canonical.MaxIdleTimeout != nil {
			t.MaxIdleTimeout = *canonical.MaxIdleTimeout
		}
	} else if compat.MaxIdleTimeout != nil {
		t.MaxIdleTimeout = *compat.MaxIdleTimeout
	}
	if _, ok := raw["max-incoming-streams"]; ok {
		if canonical.MaxIncomingStreams != nil {
			t.MaxIncomingStreams = *canonical.MaxIncomingStreams
		}
	} else if compat.MaxIncomingStreams != nil {
		t.MaxIncomingStreams = *compat.MaxIncomingStreams
	}
	if _, ok := raw["max-incoming-uni-streams"]; ok {
		if canonical.MaxIncomingUniStreams != nil {
			t.MaxIncomingUniStreams = *canonical.MaxIncomingUniStreams
		}
	} else if compat.MaxIncomingUniStreams != nil {
		t.MaxIncomingUniStreams = *compat.MaxIncomingUniStreams
	}

	return nil
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
