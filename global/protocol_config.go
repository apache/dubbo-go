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

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ProtocolConfig represents the config of protocol.
type ProtocolConfig struct {
	// Name defines the protocol name for server.
	Name string `yaml:"name" json:"name,omitempty" property:"name"`

	// Ip defines the listening IP address for server.
	Ip string `yaml:"ip"  json:"ip,omitempty" property:"ip"`

	// Port defines the listening port for server.
	Port string `yaml:"port" json:"port,omitempty" property:"port"`

	// TODO: maybe Params is useless, find a ideal way to config dubbo protocol, ref: TripleConfig.
	// Params defines additional protocol parameters for server.
	Params any `yaml:"params" json:"params,omitempty" property:"params"`

	// TripleConfig holds the Triple protocol configuration for server.
	TripleConfig *TripleConfig `yaml:"triple" json:"triple,omitempty" property:"triple"`

	// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
	//
	// MaxServerSendMsgSize defines the maximum size of messages sent by server.
	// Supported units include 1mb=1000kb=1000000b and 1mib=1024kb=1048576b.
	// For more details, see https://pkg.go.dev/github.com/dustin/go-humanize#pkg-constants.
	//
	// Deprecated: use "ClientProtocolConfig.TripleConfig.MaxServerSendMsgSize" or in config tag "protocol_config/triple/max-server-send-msg-size" instead
	MaxServerSendMsgSize string `yaml:"max-server-send-msg-size" json:"max-server-send-msg-size,omitempty"`

	// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
	//
	// MaxServerRecvMsgSize defines the maximum size of messages received by server.
	//
	// Deprecated: use "ClientProtocolConfig.TripleConfig.MaxServerRecvMsgSize" or in config tag "protocol_config/triple/max-server-recv-msg-size" instead
	MaxServerRecvMsgSize string `default:"4mib" yaml:"max-server-recv-msg-size" json:"max-server-recv-msg-size,omitempty"`
}

// DefaultProtocolConfig returns a default ProtocolConfig instance.
func DefaultProtocolConfig() *ProtocolConfig {
	return &ProtocolConfig{
		Name:         constant.TriProtocol,
		Port:         constant.DefaultTripleProtocolPort,
		TripleConfig: DefaultTripleConfig(),
	}
}

// Clone a new ProtocolConfig
func (c *ProtocolConfig) Clone() *ProtocolConfig {
	if c == nil {
		return nil
	}

	return &ProtocolConfig{
		Name:                 c.Name,
		Ip:                   c.Ip,
		Port:                 c.Port,
		Params:               c.Params,
		TripleConfig:         c.TripleConfig.Clone(),
		MaxServerSendMsgSize: c.MaxServerSendMsgSize,
		MaxServerRecvMsgSize: c.MaxServerRecvMsgSize,
	}
}
