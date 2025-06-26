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
	"strconv"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ProtocolConfig is protocol configuration
type ProtocolConfig struct {
	Name string `yaml:"name" json:"name,omitempty" property:"name"`
	Ip   string `yaml:"ip"  json:"ip,omitempty" property:"ip"`
	Port string `yaml:"port" json:"port,omitempty" property:"port"`

	// TODO: maybe Params is useless, find a ideal way to config dubbo protocol, ref: TripleConfig.
	Params any `yaml:"params" json:"params,omitempty" property:"params"`

	TripleConfig *TripleConfig `yaml:"triple" json:"triple,omitempty" property:"triple"`

	// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
	//
	// MaxServerSendMsgSize max size of server send message, 1mb=1000kb=1000000b 1mib=1024kb=1048576b.
	// more detail to see https://pkg.go.dev/github.com/dustin/go-humanize#pkg-constants
	// Deprecated：use TripleConfig
	MaxServerSendMsgSize string `yaml:"max-server-send-msg-size" json:"max-server-send-msg-size,omitempty"`
	// TODO: remove MaxServerSendMsgSize and MaxServerRecvMsgSize when version 4.0.0
	//
	// MaxServerRecvMsgSize max size of server receive message
	// Deprecated：use TripleConfig
	MaxServerRecvMsgSize string `default:"4mib" yaml:"max-server-recv-msg-size" json:"max-server-recv-msg-size,omitempty"`
}

// DefaultProtocolConfig returns a default ProtocolConfig instance.
func DefaultProtocolConfig() *ProtocolConfig {
	return &ProtocolConfig{
		Name:         constant.TriProtocol,
		Port:         strconv.Itoa(constant.DefaultPort),
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
