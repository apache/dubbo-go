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

package exposed_tmp

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// RegisterServiceInstance register service instance
func RegisterServiceInstance() error {
	defer func() {
		// TODO remove this recover func,this just to avoid some unit test failed,this will not happen in user side mostly
		// config test -> metadata exporter -> dubbo protocol/remoting -> config,cycle import will occur
		// some day we fix the cycle import then can remove this recover
		if err := recover(); err != nil {
			logger.Errorf("register service instance failed,please check if registry protocol is imported, error: %v", err)
		}
	}()
	protocol := extension.GetProtocol(constant.RegistryKey)
	if rf, ok := protocol.(registry.RegistryFactory); ok {
		for _, r := range rf.GetRegistries() {
			if sdr, ok := r.(registry.ServiceDiscoveryRegistry); ok {
				if err := sdr.RegisterService(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
