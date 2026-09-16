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

package http3config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/dustin/go-humanize"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/quicvarint"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// NewQUICConfig maps HTTP/3 transport configuration over optional default QUIC settings.
func NewQUICConfig(http3Config *global.Http3Config, defaults *quic.Config) (*quic.Config, error) {
	quicConfig := &quic.Config{}
	if defaults != nil {
		quicConfigCopy := *defaults
		quicConfig = &quicConfigCopy
	}
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

	parseReceiveWindow := func(name, value string) (uint64, error) {
		// Parse option-generated decimal values exactly before accepting humanized sizes.
		window, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
		if err != nil {
			window, err = humanize.ParseBytes(value)
		}
		if err != nil {
			return 0, fmt.Errorf("invalid http3 %s %q: %w", name, value, err)
		}
		return window, nil
	}

	initialStreamReceiveWindow := quicConfig.InitialStreamReceiveWindow
	maxStreamReceiveWindow := quicConfig.MaxStreamReceiveWindow
	initialConnectionReceiveWindow := quicConfig.InitialConnectionReceiveWindow
	maxConnectionReceiveWindow := quicConfig.MaxConnectionReceiveWindow
	var err error
	if http3Config.InitialStreamReceiveWindow != "" {
		initialStreamReceiveWindow, err = parseReceiveWindow("initial-stream-receive-window", http3Config.InitialStreamReceiveWindow)
		if err != nil {
			return nil, err
		}
	}
	if http3Config.MaxStreamReceiveWindow != "" {
		maxStreamReceiveWindow, err = parseReceiveWindow("max-stream-receive-window", http3Config.MaxStreamReceiveWindow)
		if err != nil {
			return nil, err
		}
	}
	if http3Config.InitialConnectionReceiveWindow != "" {
		initialConnectionReceiveWindow, err = parseReceiveWindow("initial-connection-receive-window", http3Config.InitialConnectionReceiveWindow)
		if err != nil {
			return nil, err
		}
	}
	if http3Config.MaxConnectionReceiveWindow != "" {
		maxConnectionReceiveWindow, err = parseReceiveWindow("max-connection-receive-window", http3Config.MaxConnectionReceiveWindow)
		if err != nil {
			return nil, err
		}
	}

	for _, receiveWindow := range []struct {
		name  string
		value uint64
	}{
		{"initial-stream-receive-window", initialStreamReceiveWindow},
		{"max-stream-receive-window", maxStreamReceiveWindow},
		{"initial-connection-receive-window", initialConnectionReceiveWindow},
		{"max-connection-receive-window", maxConnectionReceiveWindow},
	} {
		if receiveWindow.value > quicvarint.Max {
			return nil, fmt.Errorf("invalid http3 %s: value %d exceeds QUIC varint maximum %d", receiveWindow.name, receiveWindow.value, quicvarint.Max)
		}
	}
	if initialStreamReceiveWindow != 0 && maxStreamReceiveWindow != 0 && initialStreamReceiveWindow > maxStreamReceiveWindow {
		return nil, fmt.Errorf("invalid http3 receive windows: initial-stream-receive-window %d exceeds max-stream-receive-window %d", initialStreamReceiveWindow, maxStreamReceiveWindow)
	}
	if initialConnectionReceiveWindow != 0 && maxConnectionReceiveWindow != 0 && initialConnectionReceiveWindow > maxConnectionReceiveWindow {
		return nil, fmt.Errorf("invalid http3 receive windows: initial-connection-receive-window %d exceeds max-connection-receive-window %d", initialConnectionReceiveWindow, maxConnectionReceiveWindow)
	}

	quicConfig.InitialStreamReceiveWindow = initialStreamReceiveWindow
	quicConfig.MaxStreamReceiveWindow = maxStreamReceiveWindow
	quicConfig.InitialConnectionReceiveWindow = initialConnectionReceiveWindow
	quicConfig.MaxConnectionReceiveWindow = maxConnectionReceiveWindow

	return quicConfig, nil
}
