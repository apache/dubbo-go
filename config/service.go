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
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

var (
	// conServicesLock is used to guard conServices map.
	conServicesLock = sync.Mutex{}
	conServices     = map[string]common.RPCService{} // service name -> service

	// proServicesLock is used to guard proServices map
	proServicesLock = sync.Mutex{}
	proServices     = map[string]common.RPCService{} // service name -> service

	// interfaceNameConServicesLock is used to guard interfaceNameConServices map
	interfaceNameConServicesLock = sync.Mutex{}
	interfaceNameConServices     = map[string]common.RPCService{} // interfaceName -> service
)

// SetConsumerService is called by init() of implement of RPCService
func SetConsumerService(service common.RPCService) {
	ref := common.GetReference(service)
	conServicesLock.Lock()
	defer conServicesLock.Unlock()
	conServices[ref] = service
}

// SetProviderService is called by init() of implement of RPCService
func SetProviderService(service common.RPCService) {
	ref := common.GetReference(service)
	proServicesLock.Lock()
	defer proServicesLock.Unlock()
	proServices[ref] = service
}

// GetConsumerService gets ConsumerService by @name
func GetConsumerService(name string) common.RPCService {
	conServicesLock.Lock()
	defer conServicesLock.Unlock()
	return conServices[name]
}

// GetProviderService gets ProviderService by @name
func GetProviderService(name string) common.RPCService {
	proServicesLock.Lock()
	defer proServicesLock.Unlock()
	return proServices[name]
}

// SetConsumerServiceByInterfaceName is used by pb serialization
func SetConsumerServiceByInterfaceName(interfaceName string, srv common.RPCService) {
	interfaceNameConServicesLock.Lock()
	defer interfaceNameConServicesLock.Unlock()
	interfaceNameConServices[interfaceName] = srv
}

// GetConsumerServiceByInterfaceName is used by pb serialization
func GetConsumerServiceByInterfaceName(interfaceName string) common.RPCService {
	interfaceNameConServicesLock.Lock()
	defer interfaceNameConServicesLock.Unlock()
	return interfaceNameConServices[interfaceName]
}

// GetCallback gets CallbackResponse by @name
func GetCallback(name string) func(response common.CallbackResponse) {
	service := GetConsumerService(name)
	if sv, ok := service.(common.AsyncCallbackService); ok {
		return sv.CallBack
	}
	return nil
}
