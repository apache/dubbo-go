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
	"fmt"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
)

var metadataServiceProxyFactoryMap = make(map[string]func() service.MetadataServiceProxyFactory, 2)

type MetadataServiceProxyFactoryFunc func() service.MetadataServiceProxyFactory

// SetMetadataServiceProxyFactory store the name-creator pair
func SetMetadataServiceProxyFactory(name string, creator MetadataServiceProxyFactoryFunc) {
	metadataServiceProxyFactoryMap[name] = creator
}

// GetMetadataServiceProxyFactory will create an instance.
// it will panic if the factory with name not found
func GetMetadataServiceProxyFactory(name string) service.MetadataServiceProxyFactory {
	if name == "" {
		name = constant.DEFAULT_KEY
	}
	if f, ok := metadataServiceProxyFactoryMap[name]; ok {
		return f()
	}
	panic(fmt.Sprintf("could not find the metadata service factory creator for name: %s, "+
		"please check whether you have imported relative packages, "+
		"local - dubbo.apache.org/dubbo-go/v3/metadata/service/local", name))
}
