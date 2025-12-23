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

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestNewTLSConfigBuilder(t *testing.T) {
	config := NewTLSConfigBuilder().
		SetCACertFile("ca_cert_file").
		SetTLSKeyFile("tls_key_file").
		SetTLSServerName("tls_server_name").
		SetTLSCertFile("tls_cert_file").
		Build()
	assert.Equal(t, "ca_cert_file", config.CACertFile)
	assert.Equal(t, "tls_cert_file", config.TLSCertFile)
	assert.Equal(t, "tls_server_name", config.TLSServerName)
	assert.Equal(t, "tls_key_file", config.TLSKeyFile)
	assert.Equal(t, constant.TLSConfigPrefix, config.Prefix())

}
