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

package interfaces

import (
	"github.com/apache/dubbo-go/common"
)

// ConfigPostProcessor is an extension to give users a chance to customize configs against ReferenceConfig and
// ServiceConfig during deployment time.
type ConfigPostProcessor interface {
	// PostProcessReferenceConfig customizes ReferenceConfig's params.
	// PostProcessReferenceConfig emit on refer reference (GetParam(constant.HOOK_EVENT_PARAM_KEY): before-reference-connect, reference-connect-success, reference-connect-fail)
	PostProcessReferenceConfig(*common.URL)

	// PostProcessServiceConfig customizes ServiceConfig's params.
	// PostProcessServiceConfig emit on export service (GetParam(constant.HOOK_EVENT_PARAM_KEY): before-service-listen, service-listen-success, service-listen-fail)
	PostProcessServiceConfig(*common.URL)
}

// ConfigLoaderHook is extends ConfigPostProcessor
type ConfigLoaderHook interface {
	ConfigPostProcessor

	// AllReferencesConnectComplete emit on all references export complete
	AllReferencesConnectComplete()

	// AllServicesListenComplete emit on all services export complete
	AllServicesListenComplete()

	// BeforeShutdown emit on before shutdown
	BeforeShutdown()
}
