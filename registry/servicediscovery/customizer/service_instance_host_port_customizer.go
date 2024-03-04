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

package customizer

import (
	"strconv"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func init() {
	extension.AddCustomizers(&hostPortCustomizer{})
}

type hostPortCustomizer struct{}

// GetPriority will return 1 so that it will be invoked in front of user defining Customizer
func (e *hostPortCustomizer) GetPriority() int {
	return 1
}

// Customize calculate the revision for exported urls and then put it into instance metadata
func (e *hostPortCustomizer) Customize(instance registry.ServiceInstance) {
	if instance.GetPort() > 0 {
		return
	}
	if instance.GetServiceMetadata() == nil || len(instance.GetServiceMetadata().GetExportedServiceURLs()) == 0 {
		return
	}
	for _, url := range instance.GetServiceMetadata().GetExportedServiceURLs() {
		if ins, ok := instance.(*registry.DefaultServiceInstance); ok {
			ins.Host = url.Ip
			p, err := strconv.Atoi(url.Port)
			if err == nil {
				ins.Port = p
			}
			break
		}
	}
}
