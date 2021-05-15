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
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"log"
)

type LoaderInitOption interface {
	init()
	apply()
}

type optionFunc struct {
	initFunc  func()
	applyFunc func()
}

func (f *optionFunc) init() {
	f.initFunc()
}

func (f *optionFunc) apply() {
	f.applyFunc()
}

func ConsumerInitOption(confConFile string) LoaderInitOption {
	return consumerInitOption(confConFile, false)
}

func ConsumerMustInitOption(confConFile string) LoaderInitOption {
	return consumerInitOption(confConFile, true)
}

func consumerInitOption(confConFile string, must bool) LoaderInitOption {
	return &optionFunc{
		func() {
			if consumerConfig != nil && !must {
				return
			}
			if errCon := ConsumerInit(confConFile); errCon != nil {
				log.Printf("[consumerInit] %#v", errCon)
				consumerConfig = nil
			} else if confBaseFile == "" {
				// Check if there are some important key fields missing,
				// if so, we set a default value for it
				setDefaultValue(consumerConfig)
				// Even though baseConfig has been initialized, we override it
				// because we think read from config file is correct config
				baseConfig = &consumerConfig.BaseConfig
			}
		},
		func() {
			loadConsumerConfig()
		},
	}
}

func ProviderInitOption(confProFile string) LoaderInitOption {
	return providerInitOption(confProFile, false)
}

func ProviderMustInitOption(confProFile string) LoaderInitOption {
	return providerInitOption(confProFile, true)
}

func providerInitOption(confProFile string, must bool) LoaderInitOption {
	return &optionFunc{
		func() {
			if providerConfig != nil && !must {
				return
			}
			if errPro := ProviderInit(confProFile); errPro != nil {
				log.Printf("[providerInit] %#v", errPro)
				providerConfig = nil
			} else if confBaseFile == "" {
				// Check if there are some important key fields missing,
				// if so, we set a default value for it
				setDefaultValue(providerConfig)
				// Even though baseConfig has been initialized, we override it
				// because we think read from config file is correct config
				baseConfig = &providerConfig.BaseConfig
			}
		},
		func() {
			loadProviderConfig()
		},
	}
}

func RouterInitOption(crf string) LoaderInitOption {
	return &optionFunc{
		func() {
			confRouterFile = crf
		},
		func() {
			initRouter()
		},
	}
}

func BaseInitOption(cbf string) LoaderInitOption {
	return &optionFunc{
		func() {
			if cbf == "" {
				return
			}
			confBaseFile = cbf
			if err := BaseInit(cbf); err != nil {
				log.Printf("[BaseInit] %#v", err)
				baseConfig = nil
			}
		},
		func() {
			// init the global event dispatcher
			extension.SetAndInitGlobalDispatcher(GetBaseConfig().EventDispatcherType)

			// start the metadata report if config set
			if err := startMetadataReport(GetApplicationConfig().MetadataType, GetBaseConfig().MetadataReportConfig); err != nil {
				logger.Errorf("Provider starts metadata report error, and the error is {%#v}", err)
				return
			}
		},
	}
}
