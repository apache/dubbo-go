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

package config

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

	// TODO: add more params about http3
}
