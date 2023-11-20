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

func DefaultApplicationConfig() *ApplicationConfig {
	// return a new config without setting any field means there is not any default value for initialization
	return &ApplicationConfig{}
}

type ApplicationOption func(*ApplicationConfig)

func WithApplication_Organization(organization string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Organization = organization
	}
}

func WithApplication_Name(name string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Name = name
	}
}

func WithApplication_Module(module string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Module = module
	}
}

func WithApplication_Group(group string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Group = group
	}
}

func WithApplication_Version(version string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Version = version
	}
}

func WithApplication_Owner(owner string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Owner = owner
	}
}

func WithApplication_Environment(environment string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Environment = environment
	}
}

func WithApplication_MetadataType(metadataType string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.MetadataType = metadataType
	}
}

func WithApplication_Tag(tag string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Tag = tag
	}
}
