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

package extension

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/registry"
)

var (
	registrys = make(map[string]func(config *common.URL) (registry.Registry, error))
)

func SetRegistry(name string, v func(config *common.URL) (registry.Registry, error)) {
	registrys[name] = v
}

func GetRegistry(name string, config *common.URL) (registry.Registry, error) {
	if registrys[name] == nil {
		panic("registry for " + name + " is not existing, make sure you have import the package.")
	}
	return registrys[name](config)

}
