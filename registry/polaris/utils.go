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

package polaris

import (
	"fmt"
	"strconv"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type PolarisInstanceInfo struct {
	instance registry.ServiceInstance
	url      *common.URL
}

func getInstanceKey(namespace string, instance registry.ServiceInstance) string {
	return fmt.Sprintf("%s-%s-%s-%d", namespace, instance.GetServiceName(), instance.GetHost(), instance.GetPort())
}

// just copy from dubbo-go for nacos
func getCategory(url *common.URL) string {
	role, _ := strconv.Atoi(url.GetParam(constant.RegistryRoleKey, strconv.Itoa(constant.PolarisDefaultRoleType)))
	category := common.DubboNodes[role]
	return category
}
