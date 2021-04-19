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

package curator_discovery

// ServiceInstance which define in curator-x-discovery, please refer to
// https://github.com/apache/curator/blob/master/curator-x-discovery/src/main/java/org/apache/curator/x/discovery/ServiceInstance.java
type ServiceInstance struct {
	Name                string      `json:"name,omitempty"`
	ID                  string      `json:"id,omitempty"`
	Address             string      `json:"address,omitempty"`
	Port                int         `json:"port,omitempty"`
	Payload             interface{} `json:"payload,omitempty"`
	RegistrationTimeUTC int64       `json:"registrationTimeUTC,omitempty"`
}
