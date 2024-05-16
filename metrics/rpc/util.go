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

package rpc

import (
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// buildLabels will build the labels for the rpc metrics
func buildLabels(url *common.URL, invocation protocol.Invocation) map[string]string {
	return map[string]string{
		constant.TagApplicationName:    url.GetParam(constant.ApplicationKey, ""),
		constant.TagApplicationVersion: url.GetParam(constant.AppVersionKey, ""),
		constant.TagHostname:           common.GetLocalHostName(),
		constant.TagIp:                 common.GetLocalIp(),
		constant.TagInterface:          url.Service(),
		constant.TagMethod:             invocation.MethodName(),
		constant.TagGroup:              url.Group(),
		constant.TagVersion:            url.GetParam(constant.VersionKey, ""),
	}
}

// getRole will get the application role from the url
func getRole(url *common.URL) (role string) {
	if isProvider(url) {
		role = constant.SideProvider
	} else if isConsumer(url) {
		role = constant.SideConsumer
	} else {
		logger.Warnf("The url belongs neither the consumer nor the provider, "+
			"so the invocation will be ignored. url: %s", url.String())
	}
	return
}

// isProvider shows whether this url represents the application received the request as server
func isProvider(url *common.URL) bool {
	role := url.GetParam(constant.RegistryRoleKey, "")
	return strings.EqualFold(role, strconv.Itoa(common.PROVIDER))
}

// isConsumer shows whether this url represents the application sent then request as client
func isConsumer(url *common.URL) bool {
	role := url.GetParam(constant.RegistryRoleKey, "")
	return strings.EqualFold(role, strconv.Itoa(common.CONSUMER))
}
