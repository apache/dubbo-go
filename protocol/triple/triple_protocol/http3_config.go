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

package triple_protocol

import (
	"fmt"
	"time"
)

import (
	"github.com/quic-go/quic-go"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

func newQUICConfig(http3Config *global.Http3Config) (*quic.Config, error) {
	quicConfig := &quic.Config{}
	if http3Config == nil {
		return quicConfig, nil
	}

	if http3Config.KeepAlivePeriod != "" {
		keepAlivePeriod, err := time.ParseDuration(http3Config.KeepAlivePeriod)
		if err != nil {
			return nil, fmt.Errorf("invalid http3 keep-alive-period %q: %w", http3Config.KeepAlivePeriod, err)
		}
		quicConfig.KeepAlivePeriod = keepAlivePeriod
	}

	if http3Config.MaxIdleTimeout != "" {
		maxIdleTimeout, err := time.ParseDuration(http3Config.MaxIdleTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid http3 max-idle-timeout %q: %w", http3Config.MaxIdleTimeout, err)
		}
		quicConfig.MaxIdleTimeout = maxIdleTimeout
	}

	// Preserve quic-go defaults when these fields are left unset in config.
	if http3Config.MaxIncomingStreams != 0 {
		quicConfig.MaxIncomingStreams = http3Config.MaxIncomingStreams
	}
	if http3Config.MaxIncomingUniStreams != 0 {
		quicConfig.MaxIncomingUniStreams = http3Config.MaxIncomingUniStreams
	}

	return quicConfig, nil
}
