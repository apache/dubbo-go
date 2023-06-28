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

package prometheus

import (
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
	"github.com/prometheus/client_golang/prometheus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

var (
	labelNames             = []string{applicationNameKey, groupKey, hostnameKey, interfaceKey, ipKey, methodKey, versionKey}
	defaultHistogramBucket = []float64{10, 50, 100, 200, 500, 1000, 10000}
)

func buildLabels(url *common.URL) prometheus.Labels {
	return prometheus.Labels{
		applicationNameKey: url.GetParam(constant.ApplicationKey, ""),
		groupKey:           url.Group(),
		hostnameKey:        "not implemented yet",
		interfaceKey:       url.Service(),
		ipKey:              common.GetLocalIp(),
		versionKey:         url.GetParam(constant.AppVersionKey, ""),
		methodKey:          url.GetParam(constant.MethodKey, ""),
	}
}

// return the role of the application, provider or consumer, if the url is not a valid one, return empty string
func getRole(url *common.URL) (role string) {
	if isProvider(url) {
		role = providerField
	} else if isConsumer(url) {
		role = consumerField
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
