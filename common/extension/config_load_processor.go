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
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/config/interfaces"
	"reflect"
)

const (
	LoadProcessReferenceConfigFunctionName   = "LoadProcessReferenceConfig"
	LoadProcessServiceConfigFunctionName     = "LoadProcessServiceConfig"
	AllReferencesConnectCompleteFunctionName = "AllReferencesConnectComplete"
	AllServicesListenCompleteFunctionName    = "AllServicesListenComplete"
	BeforeShutdownFunctionName               = "BeforeShutdown"
)

var (
	loadProcessors      = make(map[string]interfaces.ConfigLoadProcessor)
	loadProcessorValues = make(map[string]reflect.Value)

	referenceURL = make(map[string][]*common.URL)
	serviceURL   = make(map[string][]*common.URL)
)

// SetConfigLoadProcessor registers a ConfigLoadProcessor with the given name.
func SetConfigLoadProcessor(name string, processor interfaces.ConfigLoadProcessor) {
	loadProcessors[name] = processor
	loadProcessorValues[name] = reflect.ValueOf(processor)
}

// GetConfigLoadProcessor finds a ConfigLoadProcessor by name.
func GetConfigLoadProcessor(name string) interfaces.ConfigLoadProcessor {
	return loadProcessors[name]
}

// RemoveConfigLoadProcessor remove process from processors.
func RemoveConfigLoadProcessor(name string) {
	delete(loadProcessors, name)
	delete(loadProcessorValues, name)
}

// GetConfigLoadProcessors returns all registered instances of ConfigLoadProcessor.
func GetConfigLoadProcessors() []interfaces.ConfigLoadProcessor {
	ret := make([]interfaces.ConfigLoadProcessor, 0, len(loadProcessors))
	for _, v := range loadProcessors {
		ret = append(ret, v)
	}
	return ret
}

// GetReferenceURL returns all reference URL
func GetReferenceURL() map[string][]*common.URL {
	return referenceURL
}

// GetServiceURL returns all service URL
func GetServiceURL() map[string][]*common.URL {
	return serviceURL
}

// ResetURL remove all URL
func ResetURL() {
	for k := range referenceURL {
		referenceURL[k] = nil
		delete(referenceURL, k)
	}
	for k := range serviceURL {
		serviceURL[k] = nil
		delete(serviceURL, k)
	}
}

// emit
func emit(funcName string, val ...interface{}) {
	var values []reflect.Value
	for _, arg := range val {
		values = append(values, reflect.ValueOf(arg))
	}
	for _, p := range loadProcessorValues {
		p.MethodByName(funcName).Call(values)
	}
}

// LoadProcessReferenceConfig emit reference config load event
func LoadProcessReferenceConfig(url *common.URL, event string, errMsg *string) {
	referenceURL[event] = append(referenceURL[event], url)
	emit(LoadProcessReferenceConfigFunctionName, url, event, errMsg)
}

// LoadProcessServiceConfig emit service config load event
func LoadProcessServiceConfig(url *common.URL, event string, errMsg *string) {
	serviceURL[event] = append(serviceURL[event], url)
	emit(LoadProcessServiceConfigFunctionName, url, event, errMsg)
}

// AllReferencesConnectComplete emit all references config load complete event
func AllReferencesConnectComplete() {
	binder := interfaces.ConfigLoadProcessorURLBinder{
		Success: referenceURL[constant.HookEventReferenceConnectSuccess],
		Fail:    referenceURL[constant.HookEventReferenceConnectFail],
	}
	emit(AllReferencesConnectCompleteFunctionName, binder)
}

// AllServicesListenComplete emit all services config load complete event
func AllServicesListenComplete() {
	binder := interfaces.ConfigLoadProcessorURLBinder{
		Success: serviceURL[constant.HookEventProviderConnectSuccess],
		Fail:    serviceURL[constant.HookEventProviderConnectFail],
	}
	emit(AllServicesListenCompleteFunctionName, binder)
}

// BeforeShutdown emit before os.Exit(0)
func BeforeShutdown() {
	emit(BeforeShutdownFunctionName)
}
