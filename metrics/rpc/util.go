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
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

// buildLabels will build the labels for the rpc metrics
func buildLabels(url *common.URL, invocation base.Invocation) map[string]string {
	applicationName, applicationVersion := metricApplication(url)
	return map[string]string{
		constant.TagApplicationName:    applicationName,
		constant.TagApplicationVersion: applicationVersion,
		constant.TagHostname:           common.GetLocalHostName(),
		constant.TagIp:                 common.GetLocalIp(),
		constant.TagInterface:          url.Service(),
		constant.TagMethod:             invocation.MethodName(),
		constant.TagGroup:              url.Group(),
		constant.TagVersion:            url.GetParam(constant.VersionKey, ""),
	}
}

// metricApplication keeps consumer metrics on the local application. A
// discovered provider URL retains the remote application in its params, while
// the reference URL carries the local application config in its attributes.
func metricApplication(url *common.URL) (string, string) {
	if isConsumer(url) {
		raw, _ := url.GetAttribute(constant.ApplicationKey)
		switch application := raw.(type) {
		case *global.ApplicationConfig:
			if application != nil && application.Name != "" {
				return application.Name, application.Version
			}
		case global.ApplicationConfig:
			if application.Name != "" {
				return application.Name, application.Version
			}
		}
	}
	return url.GetParam(constant.ApplicationKey, ""), url.GetParam(constant.AppVersionKey, "")
}

// getRole will get the application role from the url
func getRole(url *common.URL) (role string) {
	if isProvider(url) {
		role = constant.SideProvider
	} else if isConsumer(url) {
		role = constant.SideConsumer
	} else {
		logger.Warnf("[Metrics][RPC] the url belongs neither the consumer nor the provider, "+
			"so the invocation will be ignored. url=%s", url.String())
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
