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

// There is dubbo.properties file and application level config center configuration which higner than normal config center in java. So in java the
// configuration sequence will be config center > application level config center > dubbo.properties > spring bean configuration.
// But in go, neither the dubbo.properties file or application level config center configuration will not support for the time being.
// We just have config center configuration which can override configuration in consumer.yaml & provider.yaml.
// But for add these features in future ,I finish the environment struct following Environment class in java.
type Environment struct {
	configCenterFirst bool
	externalConfigs   sync.Map
	externalConfigMap sync.Map
}

var (
	instance *Environment
	once     sync.Once
)

func GetEnvInstance() *Environment {
	once.Do(func() {
		instance = &Environment{configCenterFirst: true}
	})
	return instance
}

//func (env *Environment) SetConfigCenterFirst() {
//	env.configCenterFirst = true
//}

//func (env *Environment) ConfigCenterFirst() bool {
//	return env.configCenterFirst
//}

func (env *Environment) UpdateExternalConfigMap(externalMap map[string]string) {
	for k, v := range externalMap {
		env.externalConfigMap.Store(k, v)
	}
}

func (env *Environment) Configuration() *list.List {
	list := list.New()
	memConf := newInmemoryConfiguration()
	memConf.setProperties(env.externalConfigMap)
	list.PushBack(memConf)
	return list
}

type InmemoryConfiguration struct {
	store sync.Map
}

func newInmemoryConfiguration() *InmemoryConfiguration {
	return &InmemoryConfiguration{}
}
func (conf *InmemoryConfiguration) setProperties(p sync.Map) {
	conf.store = p
}

func (conf *InmemoryConfiguration) GetProperty(key string) (bool, string) {
	v, ok := conf.store.Load(key)
	if ok {
		return true, v.(string)
	}
	return false, ""

}

func (conf *InmemoryConfiguration) GetSubProperty(subKey string) map[string]struct{} {
	properties := make(map[string]struct{})
	conf.store.Range(func(key, value interface{}) bool {
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
