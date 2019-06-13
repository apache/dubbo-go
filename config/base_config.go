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
	"github.com/apache/dubbo-go/common/constant"
	"reflect"
	"strconv"
	"strings"
)
import (
	perrors "github.com/pkg/errors"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/config"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
)

type baseConfig struct {
	ConfigCenterConfig ConfigCenterConfig
	configCenterUrl    *common.URL
	prefix             string
	fatherConfig       interface{}
}

func (c *baseConfig) startConfigCenter(ctx context.Context) error {
	var err error
	*c.configCenterUrl, err = common.NewURL(ctx, c.ConfigCenterConfig.Address)
	if c.prepareEnvironment() != nil {
		return perrors.WithMessagef(err, "start config center error!")
	}
	c.fresh()
	return err
}

func (c *baseConfig) prepareEnvironment() error {

	factory := extension.GetConfigCenterFactory(c.ConfigCenterConfig.Protocol)
	dynamicConfig, err := factory.GetDynamicConfiguration(c.configCenterUrl)
	if err != nil {
		logger.Errorf("Get dynamic configuration error , error message is %v", err)
		return perrors.WithStack(err)
	}
	content, err := dynamicConfig.GetConfig(c.ConfigCenterConfig.ConfigFile, config_center.WithGroup(c.ConfigCenterConfig.Group))
	if err != nil {
		logger.Errorf("Get config content in dynamic configuration error , error message is %v", err)
		return perrors.WithStack(err)
	}
	mapContent, err := dynamicConfig.Parser().Parse(content)
	if err != nil {
		return perrors.WithStack(err)
	}
	config.GetEnvInstance().UpdateExternalConfigMap(mapContent)
	return nil
}

func (c *baseConfig) fresh() {
	configList := config.GetEnvInstance().Configuration(c.Prefix())
	config := configList.Front().Value.(*config.InmemoryConfiguration)
	val := reflect.Indirect(reflect.ValueOf(c.fatherConfig))
	for i := 0; i < val.NumField(); i++ {
		if key := val.Type().Field(i).Tag.Get("property"); key != "-" && key != "" {
			f := val.Field(i)
			if f.IsValid() {
				value := config.GetProperty(key)
				setValue := func(f reflect.Value) {
					if f.Kind() == reflect.Int {
						x, err := strconv.Atoi(value)
						if err != nil {
							logger.Errorf("Dynamic change the configuration in struct {%v} field {%v} error ,error message is {%v}",
								val.Type().Name(), val.Type().Field(i).Name, err)
						} else {
							if !f.OverflowInt(int64(x)) {
								f.SetInt(int64(x))
							} else {
								logger.Errorf("Dynamic change the configuration in struct {%v} field {%v} error ,error message is {%v}",
									val.Type().Name(), val.Type().Field(i).Name, perrors.Errorf("the int64 value {%v} from config center is  overflow", int64(x)))
							}
						}

					}
					if f.Kind() == reflect.String {
						f.SetString(value)
					}
					if f.Kind() == reflect.Bool {
						x, err := strconv.ParseBool(value)
						if err != nil {
							logger.Errorf("Dynamic change the configuration in struct {%v} field {%v} error ,error message is {%v}",
								val.Type().Name(), val.Type().Field(i).Name, err)
						}
						f.SetBool(x)
					}
					if f.Kind() == reflect.Float64 {
						x, err := strconv.ParseFloat(value, 64)
						if err != nil {
							logger.Errorf("Dynamic change the configuration in struct {%v} field {%v} error ,error message is {%v}",
								val.Type().Name(), val.Type().Field(i).Name, err)
						} else {
							if !f.OverflowFloat(x) {
								f.SetFloat(x)
							} else {
								logger.Errorf("Dynamic change the configuration in struct {%v} field {%v} error ,error message is {%v}",
									val.Type().Name(), val.Type().Field(i).Name, perrors.Errorf("the float64 value {%v} from config center is  overflow", x))
							}
						}
					}
				}
				setValue(f)
				if f.Kind() == reflect.Ptr {
					setValue(f.Elem())
				}
			}
		}
	}
}

func (c *baseConfig) SetPrefix(prefix string) {
	c.prefix = prefix
}
func (c *baseConfig) Prefix() string {
	if c.prefix == "" {
		return constant.DUBBO + "." + strings.ToLower(strings.Replace(reflect.Indirect(reflect.ValueOf(c.fatherConfig)).Type().Name(), "Config", "", -1))
	} else {
		return c.prefix
	}
}
func (c *baseConfig) SetFatherConfig(fatherConfig interface{}) {
	c.fatherConfig = fatherConfig
}
