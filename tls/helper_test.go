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

package tls

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestIsServerTLSValid(t *testing.T) {
	tests := []struct {
		name     string
		tlsConf  *global.TLSConfig
		expected bool
	}{
		{
			name: "Valid TLSConfig with cert and key",
			tlsConf: &global.TLSConfig{
				TLSCertFile: "cert.pem",
				TLSKeyFile:  "key.pem",
			},
			expected: true,
		},
		{
			name: "Invalid TLSConfig with empty cert and key",
			tlsConf: &global.TLSConfig{
				TLSCertFile: "",
				TLSKeyFile:  "",
			},
			expected: false,
		},
		{
			name: "Invalid TLSConfig with only cert",
			tlsConf: &global.TLSConfig{
				TLSCertFile: "cert.pem",
				TLSKeyFile:  "",
			},
			expected: false,
		},
		{
			name: "Invalid TLSConfig with only key",
			tlsConf: &global.TLSConfig{
				TLSCertFile: "",
				TLSKeyFile:  "key.pem",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsServerTLSValid(tt.tlsConf)
			assert.Equal(t, tt.expected, result, "Test case %s failed", tt.name)
		})
	}
}

func TestIsClientTLSValid(t *testing.T) {
	tests := []struct {
		name     string
		tlsConf  *global.TLSConfig
		expected bool
	}{
		{
			name: "Valid Client TLSConfig with CA cert",
			tlsConf: &global.TLSConfig{
				CACertFile: "ca.pem",
			},
			expected: true,
		},
		{
			name: "Invalid Client TLSConfig with empty CA cert",
			tlsConf: &global.TLSConfig{
				CACertFile: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsClientTLSValid(tt.tlsConf)
			assert.Equal(t, tt.expected, result, "Test case %s failed", tt.name)
		})
	}
}
