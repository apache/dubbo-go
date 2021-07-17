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

package application

import (
	"errors"
	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// Config is a configuration for current application, whether the application is a provider or a consumer
type Config struct {
	Organization string `default:"dubbo.io" yaml:"organization" json:"organization,omitempty" property:"organization"`
	Name         string `default:"dubbo.io" yaml:"name" json:"name,omitempty" property:"name"`
	Module       string `default:"sample" yaml:"module" json:"module,omitempty" property:"module"`
	Version      string `default:"0.0.1" yaml:"version" json:"version,omitempty" property:"version"`
	Owner        string `default:"dubbo-go" yaml:"owner" json:"owner,omitempty" property:"owner"`
	Environment  string `default:"dev" yaml:"environment" json:"environment,omitempty" property:"environment"`
	// the metadata type. remote or local
	MetadataType string `default:"local" yaml:"metadataType" json:"metadataType,omitempty" property:"metadataType"`
}

// Prefix dubbo.application
func (Config) Prefix() string {
	return constant.DUBBO + ".application"
}

func (c *Config) SetDefault() error {
	return defaults.Set(c)
}

func (c *Config) Validate(valid *validator.Validate) error {
	if err := valid.Struct(c); err != nil {
		errs := err.(validator.ValidationErrors)
		var slice []string
		for _, msg := range errs {
			slice = append(slice, msg.Error())
		}
		return errors.New(strings.Join(slice, ","))
	}
	return nil
}

// UnmarshalYAML unmarshal the Config by @unmarshal function
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain Config
	return unmarshal((*plain)(c))
}
