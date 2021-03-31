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
	"bytes"
	"reflect"
	"strconv"
	"strings"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/config"
	"github.com/apache/dubbo-go/common/logger"
)

// BaseConfig is the common configuration for provider and consumer
type BaseConfig struct {
	ConfigCenterConfig *ConfigCenterConfig `yaml:"config_center" json:"config_center,omitempty"`

	// since 1.5.0 version
	Remotes              map[string]*RemoteConfig           `yaml:"remote" json:"remote,omitempty"`
	ServiceDiscoveries   map[string]*ServiceDiscoveryConfig `yaml:"service_discovery" json:"service_discovery,omitempty"`
	MetadataReportConfig *MetadataReportConfig              `yaml:"metadata_report" json:"metadata_report,omitempty" property:"metadata_report"`

	// application config
	ApplicationConfig *ApplicationConfig `yaml:"application" json:"application,omitempty" property:"application"`

	//prefix              string
	fatherConfig        interface{}
	EventDispatcherType string        `default:"direct" yaml:"event_dispatcher_type" json:"event_dispatcher_type,omitempty"`
	MetricConfig        *MetricConfig `yaml:"metrics" json:"metrics,omitempty"`
	fileStream          *bytes.Buffer

	// cache file used to store the current used configurations.
	CacheFile string `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
}

// nolint
func (c *BaseConfig) GetServiceDiscoveries(name string) (config *ServiceDiscoveryConfig, ok bool) {
	config, ok = c.ServiceDiscoveries[name]
	return
}

// GetRemoteConfig will return the remote's config with the name if found
func (c *BaseConfig) GetRemoteConfig(name string) (config *RemoteConfig, ok bool) {
	config, ok = c.Remotes[name]
	return
}

func getKeyPrefix(val reflect.Value) []string {
	var (
		prefix string
	)
	configPrefixMethod := "Prefix"
	if val.CanAddr() {
		prefix = val.Addr().MethodByName(configPrefixMethod).Call(nil)[0].String()
	} else {
		prefix = val.MethodByName(configPrefixMethod).Call(nil)[0].String()
	}

	return strings.Split(prefix, "|")
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

					prefixes := getKeyPrefix(val)

					if id.Kind() == reflect.String {
						idStr = id.Interface().(string)
					}

					for _, pfx := range prefixes {

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
						errMsg := func(structName string, fieldName string, errorDetails error) {
							logger.Errorf("Dynamic change the configuration in struct {%v} field {%v} error ,error message is {%v}",
								structName, fieldName, errorDetails)
						}
						switch f.Kind() {
						case reflect.Int64:
							x, err := strconv.Atoi(value)
							if err != nil {
								errMsg(val.Type().Name(), val.Type().Field(i).Name, err)
							} else {
								if !f.OverflowInt(int64(x)) {
									f.SetInt(int64(x))
								} else {
									errMsg(val.Type().Name(), val.Type().Field(i).Name, perrors.Errorf("the int64 value {%v} from config center is  overflow", int64(x)))
								}
							}
						case reflect.String:
							f.SetString(value)
						case reflect.Bool:
							x, err := strconv.ParseBool(value)
							if err != nil {
								errMsg(val.Type().Name(), val.Type().Field(i).Name, err)
							}
							f.SetBool(x)
						case reflect.Float64:
							x, err := strconv.ParseFloat(value, 64)
							if err != nil {
								errMsg(val.Type().Name(), val.Type().Field(i).Name, err)
							} else {
								if !f.OverflowFloat(x) {
									f.SetFloat(x)
								} else {
									errMsg(val.Type().Name(), val.Type().Field(i).Name, perrors.Errorf("the float64 value {%v} from config center is  overflow", x))
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
						// initiate config
						s := reflect.New(f.Type().Elem().Elem())
						prefix := s.MethodByName("Prefix").Call(nil)[0].String()
						for _, pfx := range strings.Split(prefix, "|") {
							m := config.GetSubProperty(pfx)
							for k := range m {
								f.SetMapIndex(reflect.ValueOf(k), reflect.New(f.Type().Elem().Elem()))
							}
						}

					}

					// iter := f.MapRange()

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
	for element := configList.Back(); element != nil; element = element.Prev() {
		cfg := element.Value.(*config.InmemoryConfiguration)
		c.freshInternalConfig(cfg)
	}
}

func (c *BaseConfig) freshInternalConfig(config *config.InmemoryConfiguration) {
	// reflect to init struct
	tp := reflect.ValueOf(c.fatherConfig).Elem().Type()
	initializeStruct(tp, reflect.ValueOf(c.fatherConfig).Elem())

	val := reflect.Indirect(reflect.ValueOf(c.fatherConfig))
	setFieldValue(val, reflect.Value{}, config)
}

// SetFatherConfig sets father config by @fatherConfig
func (c *BaseConfig) SetFatherConfig(fatherConfig interface{}) {
	c.fatherConfig = fatherConfig
}

func initializeStruct(t reflect.Type, v reflect.Value) {
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := t.Field(i)

		if ft.Tag.Get("property") == "" {
			continue
		}
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
