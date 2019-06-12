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
	"context"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	perrors "github.com/pkg/errors"
)
import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config_center"
)

type baseConfig struct {
	// application
	ApplicationConfig  ApplicationConfig `yaml:"application_config" json:"application_config,omitempty"`
	ConfigCenterConfig ConfigCenterConfig
	configCenterUrl    *common.URL
}

func (c *baseConfig) startConfigCenter(ctx context.Context) {
	var err error
	*c.configCenterUrl, err = common.NewURL(ctx, c.ConfigCenterConfig.Address)
	c.prepareEnvironment()

}

func (c *baseConfig) prepareEnvironment() error {

	factory := extension.GetConfigCenterFactory(c.ConfigCenterConfig.Protocol)
	dynamicConfig, err := factory.GetDynamicConfiguration(c.configCenterUrl)
	if err != nil {
		logger.Errorf("get dynamic configuration error , error message is %v", err)
		return err
	}
	content, err := dynamicConfig.GetConfig(c.ConfigCenterConfig.ConfigFile, config_center.WithGroup(c.ConfigCenterConfig.Group))
	if err != nil {
		logger.Errorf("Get config content in dynamic configuration error , error message is %v", err)
		return err
	}
}
