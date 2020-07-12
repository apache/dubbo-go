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
	gxset "github.com/dubbogo/gost/container/set"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/yaml"
)

var (
	routerURLSet = gxset.NewSet()
)

// RouterInit Load config file to init router config
func RouterInit(confRouterFile string) error {
	fileRouterFactories := extension.GetFileRouterFactories()
	bytes, err := yaml.LoadYMLConfig(confRouterFile)
	if err != nil {
		return perrors.Errorf("ioutil.ReadFile(file:%s) = error:%v", confRouterFile, perrors.WithStack(err))
	}
	logger.Warnf("get fileRouterFactories len{%+v})", len(fileRouterFactories))
	for k, factory := range fileRouterFactories {
		r, e := factory.NewFileRouter(bytes)
		if e == nil {
			url := r.URL()
			routerURLSet.Add(&url)
			return nil
		}
		logger.Warnf("router config type %s create fail {%v}\n", k, e)
	}
	return perrors.Errorf("no file router exists for parse %s , implement router.FIleRouterFactory please.", confRouterFile)
}

func GetRouterURLSet() *gxset.HashSet {
	return routerURLSet
}
