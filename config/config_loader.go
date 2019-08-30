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
	"fmt"
	"log"
	"os"
	"time"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

var (
	consumerConfig *ConsumerConfig
	providerConfig *ProviderConfig
	maxWait        = 3
)

// loaded consumer & provider config from xxx.yml, and log config from xxx.xml
// Namely: dubbo.consumer.xml & dubbo.provider.xml in java dubbo
func init() {
	var (
		confConFile, confProFile string
	)

	confConFile = os.Getenv(constant.CONF_CONSUMER_FILE_PATH)
	confProFile = os.Getenv(constant.CONF_PROVIDER_FILE_PATH)
	if errCon := ConsumerInit(confConFile); errCon != nil {
		log.Printf("[consumerInit] %#v", errCon)
		consumerConfig = nil
	}
	if errPro := ProviderInit(confProFile); errPro != nil {
		log.Printf("[providerInit] %#v", errPro)
		providerConfig = nil
	}
}

// Dubbo Init
func Load() {
	// reference config
	if consumerConfig == nil {
		logger.Warnf("consumerConfig is nil!")
	} else {
		if err := configCenterRefreshConsumer(); err != nil {
			logger.Errorf("[consumer config center refresh] %#v", err)
		}
		for key, ref := range consumerConfig.References {
			if ref.Generic {
				genericService := NewGenericService(key)
				SetConsumerService(genericService)
			}
			rpcService := GetConsumerService(key)
			if rpcService == nil {
				logger.Warnf("%s does not exist!", key)
				continue
			}
			ref.id = key
			ref.Refer()
			ref.Implement(rpcService)
		}
		//wait for invoker is available, if wait over default 3s, then panic
		var count int
		checkok := true
		for {
			for _, refconfig := range consumerConfig.References {
				if (refconfig.Check != nil && *refconfig.Check) ||
					(refconfig.Check == nil && consumerConfig.Check != nil && *consumerConfig.Check) ||
					(refconfig.Check == nil && consumerConfig.Check == nil) { //default to true

					if refconfig.invoker != nil &&
						!refconfig.invoker.IsAvailable() {
						checkok = false
						count++
						if count > maxWait {
							panic(fmt.Sprintf("Failed to check the status of the service %v . No provider available for the service to the consumer use dubbo version %v", refconfig.InterfaceName, constant.Version))
						}
						time.Sleep(time.Second * 1)
						break
					}
					if refconfig.invoker == nil {
						logger.Warnf("The interface %s invoker not exist , may you should check your interface config.", refconfig.InterfaceName)
					}
				}
			}
			if checkok {
				break
			}
			checkok = true
		}
	}

	// service config
	if providerConfig == nil {
		logger.Warnf("providerConfig is nil!")
	} else {
		if err := configCenterRefreshProvider(); err != nil {
			logger.Errorf("[provider config center refresh] %#v", err)
		}
		for key, svs := range providerConfig.Services {
			rpcService := GetProviderService(key)
			if rpcService == nil {
				logger.Warnf("%s does not exist!", key)
				continue
			}
			svs.id = key
			svs.Implement(rpcService)
			if err := svs.Export(); err != nil {
				panic(fmt.Sprintf("service %s export failed! ", key))
			}
		}
	}
}

// get rpc service for consumer
func GetRPCService(name string) common.RPCService {
	return consumerConfig.References[name].GetRPCService()
}

// create rpc service for consumer
func RPCService(service common.RPCService) {
	consumerConfig.References[service.Reference()].Implement(service)
}
