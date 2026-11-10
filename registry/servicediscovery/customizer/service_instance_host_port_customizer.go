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

// hostPortCustomizer fills the host and port of a DefaultServiceInstance from
// its exported service URLs.
type hostPortCustomizer struct{}

// GetPriority will return 1 so that it will be invoked in front of user defining Customizer
func (e *hostPortCustomizer) GetPriority() int {
	return 1
}

// Customize sets the host and port of the instance from the first exported
// service URL, so that the instance carries a reachable address.
// It only applies to *registry.DefaultServiceInstance, and does nothing when
// the port is already set or when the instance has no exported service URLs.
// An unparsable port leaves the port unchanged, while the host is still set.
func (e *hostPortCustomizer) Customize(instance registry.ServiceInstance) {
	if instance.GetPort() > 0 { // has set, avoid reset
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
