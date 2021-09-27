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
	"github.com/creasty/defaults"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ApplicationConfig is a configuration for current applicationConfig, whether the applicationConfig is a provider or a consumer
type ApplicationConfig struct {
	Organization string `default:"dubbo-go" yaml:"organization" json:"organization,omitempty" property:"organization"`
	Name         string `default:"dubbo.io" yaml:"name" json:"name,omitempty" property:"name"`
	Module       string `default:"sample" yaml:"module" json:"module,omitempty" property:"module"`
	Version      string `default:"0.0.1" yaml:"version" json:"version,omitempty" property:"version"`
	Owner        string `default:"dubbo-go" yaml:"owner" json:"owner,omitempty" property:"owner"`
	Environment  string `default:"dev" yaml:"environment" json:"environment,omitempty" property:"environment"`
	// the metadata type. remote or local
	MetadataType string `default:"local" yaml:"metadata-type" json:"metadataType,omitempty" property:"metadataType"`
}

// Prefix dubbo.application
func (ApplicationConfig) Prefix() string {
	return constant.ApplicationConfigPrefix
}

// Init  application config and set default value
func (ac *ApplicationConfig) Init() error {
	if ac == nil {
		return errors.New("application is null")
	}
	if err := ac.check(); err != nil {
		return err
	}
	return nil
}

func GetApplicationInstance(opts ...ApplicationConfigOpt) *ApplicationConfig {
	ac := &ApplicationConfig{}
	for _, opt := range opts {
		opt(ac)
	}
	return ac
}

func (ac *ApplicationConfig) check() error {
	if err := defaults.Set(ac); err != nil {
		return err
	}
	return verify(ac)
}

type ApplicationConfigOpt func(config *ApplicationConfig)

func WithOrganization(organization string) ApplicationConfigOpt {
	return func(ac *ApplicationConfig) {
		ac.Organization = organization
	}
}

func WithName(name string) ApplicationConfigOpt {
	return func(ac *ApplicationConfig) {
		ac.Name = name
	}
}

func WithModule(module string) ApplicationConfigOpt {
	return func(ac *ApplicationConfig) {
		ac.Module = module
	}
}

func WithVersion(version string) ApplicationConfigOpt {
	return func(ac *ApplicationConfig) {
		ac.Version = version
	}
}

func WithOwner(owner string) ApplicationConfigOpt {
	return func(ac *ApplicationConfig) {
		ac.Owner = owner
	}
}

func WithEnvironment(env string) ApplicationConfigOpt {
	return func(ac *ApplicationConfig) {
		ac.Environment = env
	}
}

func WithMetadataType(metadataType string) ApplicationConfigOpt {
	return func(ac *ApplicationConfig) {
		ac.MetadataType = metadataType
	}
}

type ApplicationConfigBuilder struct {
	*ApplicationConfig
}

func (acb *ApplicationConfigBuilder) Organization(organization string) *ApplicationConfigBuilder {
	acb.ApplicationConfig.Organization = organization
	return acb
}

func (acb *ApplicationConfigBuilder) Name(name string) *ApplicationConfigBuilder {
	acb.ApplicationConfig.Name = name
	return acb
}

func (acb *ApplicationConfigBuilder) Module(module string) *ApplicationConfigBuilder {
	acb.ApplicationConfig.Module = module
	return acb
}

func (acb *ApplicationConfigBuilder) Version(version string) *ApplicationConfigBuilder {
	acb.ApplicationConfig.Version = version
	return acb
}

func (acb *ApplicationConfigBuilder) Owner(owner string) *ApplicationConfigBuilder {
	acb.ApplicationConfig.Owner = owner
	return acb
}

func (acb *ApplicationConfigBuilder) Environment(environment string) *ApplicationConfigBuilder {
	acb.ApplicationConfig.Environment = environment
	return acb
}

func (acb *ApplicationConfigBuilder) MetadataType(metadataType string) *ApplicationConfigBuilder {
	acb.ApplicationConfig.MetadataType = metadataType
	return acb
}

func (acb *ApplicationConfigBuilder) Build() *ApplicationConfig {
	if err := acb.Init(); err != nil {
		panic(err)
	}
	return acb.ApplicationConfig
}
