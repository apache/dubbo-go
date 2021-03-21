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
	"container/list"
	"strings"
	"sync"
)

import (
	"github.com/apache/dubbo-go/config_center"
)

// Environment
// There is dubbo.properties file and application level config center configuration which higner than normal config center in java. So in java the
// configuration sequence will be config center > application level config center > dubbo.properties > spring bean configuration.
// But in go, neither the dubbo.properties file or application level config center configuration will not support for the time being.
// We just have config center configuration which can override configuration in consumer.yaml & provider.yaml.
// But for add these features in future ,I finish the environment struct following Environment class in java.
type Environment struct {
	configCenterFirst bool
	//externalConfigs      sync.Map
	externalConfigMap    sync.Map
	appExternalConfigMap sync.Map
	dynamicConfiguration config_center.DynamicConfiguration
}

var (
	instance *Environment
	once     sync.Once
)

// GetEnvInstance gets env instance by singleton
func GetEnvInstance() *Environment {
	once.Do(func() {
		instance = &Environment{configCenterFirst: true}
	})
	return instance
}

// NewEnvInstance creates Environment instance
func NewEnvInstance() {
	instance = &Environment{configCenterFirst: true}
}

//func (env *Environment) SetConfigCenterFirst() {
//	env.configCenterFirst = true
//}

//func (env *Environment) ConfigCenterFirst() bool {
//	return env.configCenterFirst
//}

// UpdateExternalConfigMap updates env externalConfigMap field
func (env *Environment) UpdateExternalConfigMap(externalMap map[string]string) {
	for k, v := range externalMap {
		env.externalConfigMap.Store(k, v)
	}
	env.externalConfigMap.Range(func(key, value interface{}) bool {
		if _, ok := externalMap[key.(string)]; !ok {
			env.externalConfigMap.Delete(key)
		}
		return true
	})
}

// UpdateAppExternalConfigMap updates env appExternalConfigMap field
func (env *Environment) UpdateAppExternalConfigMap(externalMap map[string]string) {
	for k, v := range externalMap {
		env.appExternalConfigMap.Store(k, v)
	}
	env.appExternalConfigMap.Range(func(key, value interface{}) bool {
		if _, ok := externalMap[key.(string)]; !ok {
			env.appExternalConfigMap.Delete(key)
		}
		return true
	})
}

// Configuration puts externalConfigMap and appExternalConfigMap into list
// List represents a doubly linked list.
func (env *Environment) Configuration() *list.List {
	cfgList := list.New()
	// The sequence would be: SystemConfiguration -> AppExternalConfiguration -> ExternalConfiguration -> AbstractConfig -> PropertiesConfiguration
	cfgList.PushBack(newInmemoryConfiguration(&(env.appExternalConfigMap)))
	cfgList.PushBack(newInmemoryConfiguration(&(env.externalConfigMap)))
	return cfgList
}

// SetDynamicConfiguration sets value for dynamicConfiguration
func (env *Environment) SetDynamicConfiguration(dc config_center.DynamicConfiguration) {
	env.dynamicConfiguration = dc
}

// GetDynamicConfiguration gets dynamicConfiguration
func (env *Environment) GetDynamicConfiguration() config_center.DynamicConfiguration {
	return env.dynamicConfiguration
}

// InmemoryConfiguration stores config in memory
type InmemoryConfiguration struct {
	store *sync.Map
}

func newInmemoryConfiguration(p *sync.Map) *InmemoryConfiguration {
	return &InmemoryConfiguration{store: p}
}

// GetProperty gets value from InmemoryConfiguration instance by @key
func (conf *InmemoryConfiguration) GetProperty(key string) (bool, string) {
	if conf.store == nil {
		return false, ""
	}

	v, ok := conf.store.Load(key)
	if ok {
		return true, v.(string)
	}

	return false, ""
}

// GetSubProperty gets sub property from InmemoryConfiguration instance by @subkey
func (conf *InmemoryConfiguration) GetSubProperty(subKey string) map[string]struct{} {
	if conf.store == nil {
		return nil
	}

	properties := make(map[string]struct{})
	conf.store.Range(func(key, _ interface{}) bool {
		if idx := strings.Index(key.(string), subKey); idx >= 0 {
			after := key.(string)[idx+len(subKey):]
			if i := strings.Index(after, "."); i >= 0 {
				properties[after[0:strings.Index(after, ".")]] = struct{}{}
			}

		}
		return true
	})

	return properties
}
