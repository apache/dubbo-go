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

var instance *Environment

var once sync.Once

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

func (env *Environment) Configuration(prefix string) *list.List {
	list := list.New()
	memConf := newInmemoryConfiguration(prefix)
	memConf.setProperties(env.externalConfigMap)
	list.PushBack(memConf)
	return list
}

type InmemoryConfiguration struct {
	prefix string
	store  sync.Map
}

func newInmemoryConfiguration(prefix string) *InmemoryConfiguration {
	return &InmemoryConfiguration{prefix: prefix}
}
func (conf *InmemoryConfiguration) setProperties(p sync.Map) {
	conf.store = p
}

func (conf *InmemoryConfiguration) GetProperty(key string) string {
	v, ok := conf.store.Load(conf.prefix + "." + key)
	if ok {
		return v.(string)
	} else {
		return ""
	}
}
