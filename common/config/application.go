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

// todo(DMwangnima): think about the location of this type of configuration.

// ApplicationConfig is a configuration for current applicationConfig, whether the applicationConfig is a provider or a consumer
type ApplicationConfig struct {
	Organization string `default:"dubbo-go" yaml:"organization" json:"organization,omitempty" property:"organization"`
	Name         string `default:"dubbo.io" yaml:"name" json:"name,omitempty" property:"name"`
	Module       string `default:"sample" yaml:"module" json:"module,omitempty" property:"module"`
	Group        string `yaml:"group" json:"group,omitempty" property:"module"`
	Version      string `yaml:"version" json:"version,omitempty" property:"version"`
	Owner        string `default:"dubbo-go" yaml:"owner" json:"owner,omitempty" property:"owner"`
	Environment  string `yaml:"environment" json:"environment,omitempty" property:"environment"`
	// the metadata type. remote or local
	MetadataType string `default:"local" yaml:"metadata-type" json:"metadataType,omitempty" property:"metadataType"`
	Tag          string `yaml:"tag" json:"tag,omitempty" property:"tag"`
}

type ApplicationOption func(*ApplicationConfig)

func WithOrganization(organization string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Organization = organization
	}
}

func WithName(name string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Name = name
	}
}

func WithModule(module string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Module = module
	}
}

func WithGroup(group string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Group = group
	}
}

func WithVersion(version string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Version = version
	}
}

func WithOwner(owner string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Owner = owner
	}
}

func WithEnvironment(environment string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Environment = environment
	}
}

func WithMetadataType(metadataType string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.MetadataType = metadataType
	}
}

func WithTag(tag string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Tag = tag
	}
}
