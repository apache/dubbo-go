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

package report

// MetadataReportConfig is app level configuration
type MetadataReportConfig struct {
	Protocol  string `required:"true"  yaml:"protocol"  json:"protocol,omitempty"`
	Address   string `required:"true" yaml:"address" json:"address"`
	Username  string `yaml:"username" json:"username,omitempty"`
	Password  string `yaml:"password" json:"password,omitempty"`
	Timeout   string `yaml:"timeout" json:"timeout,omitempty"`
	Group     string `yaml:"group" json:"group,omitempty"`
	Namespace string `yaml:"namespace" json:"namespace,omitempty"`
	// metadataType of this application is defined by application config, local or remote
	metadataType string
}

func DefaultMetadataReportConfig() *MetadataReportConfig {
	// return a new config without setting any field means there is not any default value for initialization
	return &MetadataReportConfig{}
}

type MetadataReportOption func(*MetadataReportConfig)

func WithProtocol(protocol string) MetadataReportOption {
	return func(cfg *MetadataReportConfig) {
		cfg.Protocol = protocol
	}
}

func WithAddress(address string) MetadataReportOption {
	return func(cfg *MetadataReportConfig) {
		cfg.Address = address
	}
}

func WithUsername(username string) MetadataReportOption {
	return func(cfg *MetadataReportConfig) {
		cfg.Username = username
	}
}

func WithPassword(password string) MetadataReportOption {
	return func(cfg *MetadataReportConfig) {
		cfg.Password = password
	}
}

func WithTimeout(timeout string) MetadataReportOption {
	return func(cfg *MetadataReportConfig) {
		cfg.Timeout = timeout
	}
}

func WithGroup(group string) MetadataReportOption {
	return func(cfg *MetadataReportConfig) {
		cfg.Group = group
	}
}

func WithNamespace(namespace string) MetadataReportOption {
	return func(cfg *MetadataReportConfig) {
		cfg.Namespace = namespace
	}
}
