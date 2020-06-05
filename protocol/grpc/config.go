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

package grpc

import (
	perrors "github.com/pkg/errors"
)

type (
	// ServerConfig
	ServerConfig struct {
	}

	// ClientConfig
	ClientConfig struct {
		// content type, more information refer by https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#requests
		ContentSubType string `default:"proto" yaml:"content_sub_type" json:"content_sub_type,omitempty"`
	}
)

// GetDefaultClientConfig ...
func GetDefaultClientConfig() ClientConfig {
	return ClientConfig{
		ContentSubType: CODEC_PROTO,
	}
}

// GetDefaultServerConfig ...
func GetDefaultServerConfig() ServerConfig {
	return ServerConfig{}
}

// GetClientConfig ...
func GetClientConfig() ClientConfig {
	return ClientConfig{}
}

// clientConfig Validate ...
func (c *ClientConfig) Validate() error {
	if c.ContentSubType != CODEC_JSON && c.ContentSubType != CODEC_PROTO {
		return perrors.Errorf(" dubbo-go grpc codec currently only support proto、json, %s isn't supported,"+
			" please check protocol content_sub_type config", c.ContentSubType)
	}
	return nil
}

// severConfig Validate ...
func (c *ServerConfig) Validate() error {
	return nil
}
