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

type (
	// ServerConfig
	ServerConfig struct {
	}

	// ClientConfig
	ClientConfig struct {
		// content type, more information refer by https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#requests
		ContentType string `default:"application/grpc+proto" yaml:"content_type" json:"content_type,omitempty"`
	}
)

// GetDefaultClientConfig ...
func GetDefaultClientConfig() ClientConfig {
	return ClientConfig{
		ContentType: "application/grpc+proto",
	}
}

// GetDefaultServerConfig ...
func GetDefaultServerConfig() ServerConfig {
	return ServerConfig{
	}
}

func (c *ClientConfig) Validate() error {
	return nil
}

func (c *ServerConfig) Validate() error {
	return nil
}
