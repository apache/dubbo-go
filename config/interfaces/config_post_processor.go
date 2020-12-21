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

// PostConfigurable is a marker to indicate the instance can be post-configured.
type PostConfigurable interface{}

// ConfigPostProcessor is an extension to give users a chance to customize configs against ReferenceConfig and
// ServiceConfig during deployment time.
type ConfigPostProcessor interface {
	// PostProcessReferenceConfig customizes ReferenceConfig's params. The 'target' parameter must be a instance of
	// ReferenceConfig. The implementation must assert the passed in instance is a ReferenceConfig.
	PostProcessReferenceConfig(target PostConfigurable)

	// PostProcessServiceConfig customizes ServiceConfig's params. The 'target' parameter must be an instance of
	// ServiceConfig. The implementation must assert the passed in instance is a ServiceConfig.
	PostProcessServiceConfig(target PostConfigurable)
}
