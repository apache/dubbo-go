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
	// ServerConfig currently is empty struct,for future expansion
	ServerConfig struct{}

	// ClientConfig wrap client call parameters
	ClientConfig struct {
		// content type, more information refer by https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#requests
		ContentSubType string `default:"proto" yaml:"content_sub_type" json:"content_sub_type,omitempty"`
	}
)

// GetDefaultClientConfig return grpc client default call options
func GetDefaultClientConfig() ClientConfig {
	return ClientConfig{
		ContentSubType: codecProto,
	}
}

// GetDefaultServerConfig currently return empty struct,for future expansion
func GetDefaultServerConfig() ServerConfig {
	return ServerConfig{}
}

// GetClientConfig return grpc client custom call options
func GetClientConfig() ClientConfig {
	return ClientConfig{}
}

// Validate check if custom config encoding is supported in dubbo grpc
func (c *ClientConfig) Validate() error {
	if c.ContentSubType != codecJson && c.ContentSubType != codecProto {
		return perrors.Errorf(" dubbo-go grpc codec currently only support proto、json, %s isn't supported,"+
			" please check protocol content_sub_type config", c.ContentSubType)
	}
	return nil
}

// Validate currently return empty struct,for future expansion
func (c *ServerConfig) Validate() error {
	return nil
}
