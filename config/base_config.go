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

type multiConfiger interface {
	Prefix() string
}

type BaseConfig struct {
	ConfigCenterConfig *ConfigCenterConfig `yaml:"config_center" json:"config_center,omitempty"`
	configCenterUrl    *common.URL
	prefix             string
	fatherConfig       interface{}
}

func (c *BaseConfig) startConfigCenter(ctx context.Context) error {
	url, err := common.NewURL(ctx, c.ConfigCenterConfig.Address, common.WithProtocol(c.ConfigCenterConfig.Protocol))
	if err != nil {
		return err
	}
	c.configCenterUrl = &url
	if c.prepareEnvironment() != nil {
		return perrors.WithMessagef(err, "start config center error!")
	}
	//c.fresh()
	return err
}

func (c *BaseConfig) prepareEnvironment() error {

	factory := extension.GetConfigCenterFactory(c.ConfigCenterConfig.Protocol)
	dynamicConfig, err := factory.GetDynamicConfiguration(c.configCenterUrl)
	config.GetEnvInstance().SetDynamicConfiguration(dynamicConfig)
	if err != nil {
		logger.Errorf("Get dynamic configuration error , error message is %v", err)
		return perrors.WithStack(err)
	}
	content, err := dynamicConfig.GetConfig(c.ConfigCenterConfig.ConfigFile, config_center.WithGroup(c.ConfigCenterConfig.Group))
	if err != nil {
		logger.Errorf("Get config content in dynamic configuration error , error message is %v", err)
		return perrors.WithStack(err)
	}
	var appGroup string
	var appContent string
	if providerConfig != nil && providerConfig.ApplicationConfig != nil &&
		reflect.ValueOf(c.fatherConfig).Elem().Type().Name() == "ProviderConfig" {
		appGroup = providerConfig.ApplicationConfig.Name
	} else if consumerConfig != nil && consumerConfig.ApplicationConfig != nil &&
		reflect.ValueOf(c.fatherConfig).Elem().Type().Name() == "ConsumerConfig" {
		appGroup = consumerConfig.ApplicationConfig.Name
	}

	if len(appGroup) != 0 {
		configFile := c.ConfigCenterConfig.AppConfigFile
		if len(configFile) == 0 {
			configFile = c.ConfigCenterConfig.ConfigFile
		}
		appContent, err = dynamicConfig.GetConfig(configFile, config_center.WithGroup(appGroup))
	}
	//global config file
	mapContent, err := dynamicConfig.Parser().Parse(content)
	if err != nil {
		return perrors.WithStack(err)
	}
	config.GetEnvInstance().UpdateExternalConfigMap(mapContent)

	//appGroup config file
	if len(appContent) != 0 {
		appMapConent, err := dynamicConfig.Parser().Parse(appContent)
		if err != nil {
			return perrors.WithStack(err)
		}
		config.GetEnvInstance().UpdateAppExternalConfigMap(appMapConent)
	}

	return nil
}

func getKeyPrefix(val reflect.Value) []string {
	var (
		prefix string
	)

	if val.CanAddr() {
		prefix = val.Addr().MethodByName("Prefix").Call(nil)[0].String()
	} else {
		prefix = val.MethodByName("Prefix").Call(nil)[0].String()
	}
	var retPrefixs []string

	for _, pfx := range strings.Split(prefix, "|") {

		retPrefixs = append(retPrefixs, pfx)

	}
	return retPrefixs

}
func getPtrElement(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		if v.Kind() == reflect.Ptr {
			return getPtrElement(v)
		}
	}
	return v
}
func setFieldValue(val reflect.Value, id reflect.Value, config *config.InmemoryConfiguration) {
	for i := 0; i < val.NumField(); i++ {
		if key := val.Type().Field(i).Tag.Get("property"); key != "-" && key != "" {
			f := val.Field(i)
			if f.IsValid() {
				setBaseValue := func(f reflect.Value) {

					var (
						ok    bool
						value string
						idStr string
					)

					prefixs := getKeyPrefix(val)

					if id.Kind() == reflect.String {
						idStr = id.Interface().(string)
					}

					for _, pfx := range prefixs {

						if len(pfx) > 0 {
							if len(idStr) > 0 {
								ok, value = config.GetProperty(pfx + idStr + "." + key)
							}
							if len(value) == 0 || !ok {
								ok, value = config.GetProperty(pfx + key)
							}

						} else {
							ok, value = config.GetProperty(key)
						}

						if ok {
							break
						}

					}
					if ok {
						switch f.Kind() {
						case reflect.Int64:
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
						case reflect.String:
							f.SetString(value)
						case reflect.Bool:
							x, err := strconv.ParseBool(value)
							if err != nil {
								logger.Errorf("Dynamic change the configuration in struct {%v} field {%v} error ,error message is {%v}",
									val.Type().Name(), val.Type().Field(i).Name, err)
							}
							f.SetBool(x)
						case reflect.Float64:
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
						default:
							logger.Warnf("The kind of field {%v} is not supported ", f.Kind().String())
						}

					}

				}

				if f.Kind() == reflect.Ptr {
					f = getPtrElement(f)
					if f.Kind() == reflect.Struct {
						setFieldValue(f, reflect.Value{}, config)
					} else {
						setBaseValue(f)
					}
				}

				if f.Kind() == reflect.Struct {
					setFieldValue(f, reflect.Value{}, config)
				}
				if f.Kind() == reflect.Slice {
					for i := 0; i < f.Len(); i++ {
						e := f.Index(i)
						if e.Kind() == reflect.Ptr {
							e = getPtrElement(e)
							if e.Kind() == reflect.Struct {
								setFieldValue(e, reflect.Value{}, config)
							} else {
								setBaseValue(e)
							}
						}

					}

				}
				if f.Kind() == reflect.Map {

					if f.Type().Elem().Kind() == reflect.Ptr {
						//initiate config
						s := reflect.New(f.Type().Elem().Elem())
						prefix := s.MethodByName("Prefix").Call(nil)[0].String()
						for _, pfx := range strings.Split(prefix, "|") {
							m := config.GetSubProperty(pfx)
							if m != nil {
								for k := range m {
									f.SetMapIndex(reflect.ValueOf(k), reflect.New(f.Type().Elem().Elem()))
								}
							}

						}

					}

					//iter := f.MapRange()

					for _, k := range f.MapKeys() {
						v := f.MapIndex(k)
						switch v.Kind() {
						case reflect.Ptr:
							v = getPtrElement(v)
							if v.Kind() == reflect.Struct {
								setFieldValue(v, k, config)
							} else {
								setBaseValue(v)
							}
						case reflect.Int64, reflect.String, reflect.Bool, reflect.Float64:
							setBaseValue(v)
						default:
							logger.Warnf("The kind of field {%v} is not supported ", v.Kind().String())
						}
					}
				}
				setBaseValue(f)

			}
		}
	}
}
func (c *BaseConfig) fresh() {
	configList := config.GetEnvInstance().Configuration()
	for element := configList.Front(); element != nil; element = element.Next() {
		config := element.Value.(*config.InmemoryConfiguration)
		c.freshInternalConfig(config)
	}
}

func (c *BaseConfig) freshInternalConfig(config *config.InmemoryConfiguration) {
	//reflect to init struct
	tp := reflect.ValueOf(c.fatherConfig).Elem().Type()
	initializeStruct(tp, reflect.ValueOf(c.fatherConfig).Elem())

	val := reflect.Indirect(reflect.ValueOf(c.fatherConfig))
	setFieldValue(val, reflect.Value{}, config)
}

func (c *BaseConfig) SetFatherConfig(fatherConfig interface{}) {
	c.fatherConfig = fatherConfig
}

func initializeStruct(t reflect.Type, v reflect.Value) {
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			ft := t.Field(i)

			if ft.Tag.Get("property") != "" {
				switch ft.Type.Kind() {
				case reflect.Map:
					if f.IsNil() {
						f.Set(reflect.MakeMap(ft.Type))
					}
				case reflect.Slice:
					if f.IsNil() {
						f.Set(reflect.MakeSlice(ft.Type, 0, 0))
					}
				case reflect.Chan:
					if f.IsNil() {
						f.Set(reflect.MakeChan(ft.Type, 0))
					}
				case reflect.Struct:
					if f.IsNil() {
						initializeStruct(ft.Type, f)
					}
				case reflect.Ptr:
					if f.IsNil() {
						fv := reflect.New(ft.Type.Elem())
						initializeStruct(ft.Type.Elem(), fv.Elem())
						f.Set(fv)
					}
				default:
				}
			}

		}
	}

}
