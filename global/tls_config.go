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

// TLSConfig tls config
type TLSConfig struct {
	CACertFile    string `yaml:"ca-cert-file" json:"ca-cert-file" property:"ca-cert-file"`
	TLSCertFile   string `yaml:"tls-cert-file" json:"tls-cert-file" property:"tls-cert-file"`
	TLSKeyFile    string `yaml:"tls-key-file" json:"tls-key-file" property:"tls-key-file"`
	TLSServerName string `yaml:"tls-server-name" json:"tls-server-name" property:"tls-server-name"`
}

func DefaultTLSConfig() *TLSConfig {
	return nil
}

type TLSOption func(*TLSConfig)

func WithTLS_CACertFile(file string) TLSOption {
	return func(cfg *TLSConfig) {
		cfg.CACertFile = file
	}
}

func WithTLS_TLSCertFile(file string) TLSOption {
	return func(cfg *TLSConfig) {
		cfg.TLSCertFile = file
	}
}

func WithTLS_TLSKeyFile(file string) TLSOption {
	return func(cfg *TLSConfig) {
		cfg.TLSKeyFile = file
	}
}

func WithTLS_TLSServerName(name string) TLSOption {
	return func(cfg *TLSConfig) {
		cfg.TLSServerName = name
	}
}
