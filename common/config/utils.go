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

package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

import (
	"github.com/go-playground/validator/v10"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// TranslateIds string "nacos,zk" => ["nacos","zk"]
func TranslateIds(registryIds []string) []string {
	ids := make([]string, 0)
	for _, id := range registryIds {

		ids = append(ids, strings.Split(id, ",")...)
	}
	return removeDuplicateElement(ids)
}

// removeDuplicateElement remove duplicate element
func removeDuplicateElement(items []string) []string {
	result := make([]string, 0, len(items))
	temp := map[string]struct{}{}
	for _, item := range items {
		if _, ok := temp[item]; !ok && item != "" {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func Verify(s any) error {
	if err := validate.Struct(s); err != nil {
		errs := err.(validator.ValidationErrors)
		var slice []string
		for _, msg := range errs {
			slice = append(slice, msg.Error())
		}
		return errors.New(strings.Join(slice, ","))
	}
	return nil
}

// MergeObservabilityFilters resolves the configured chain and inserts automatic
// filters while preserving the order of explicitly configured filters.
func MergeObservabilityFilters(defaults, explicit, traceFilter string, metrics *global.MetricsConfig, tracing *global.OtelTraceConfig) string {
	addMetrics := metrics.Enable != nil && *metrics.Enable
	addTracing := tracing.Enable != nil && *tracing.Enable
	exemplars := metrics.RPCExemplarsEnabled()
	if exemplars {
		for name := range strings.SplitSeq(defaults+","+explicit, ",") {
			switch strings.TrimSpace(name) {
			case constant.MetricsFilterKey:
				addMetrics = false
			case traceFilter:
				addTracing = false
			}
		}
	}
	if addTracing && traceFilter == constant.OTELServerTraceKey {
		// Provider filters run inside the server span, including their logs.
		defaults = traceFilter + "," + defaults
	} else if addTracing && exemplars {
		defaults += "," + traceFilter
	}
	if addMetrics {
		defaults += "," + constant.MetricsFilterKey
	}
	if addTracing && !exemplars && traceFilter != constant.OTELServerTraceKey {
		defaults += "," + traceFilter
	}
	if explicit == "" {
		explicit, defaults = defaults, ""
	}
	chain := MergeValue(explicit, "", defaults)
	if !exemplars || addTracing == addMetrics {
		return chain
	}
	return orderAutomaticObservabilityFilter(chain, traceFilter, addTracing)
}

func orderAutomaticObservabilityFilter(chain, traceFilter string, addTracing bool) string {
	filters := strings.Split(chain, ",")
	traceIndex := slices.IndexFunc(filters, func(filter string) bool { return strings.TrimSpace(filter) == traceFilter })
	metricsIndex := slices.IndexFunc(filters, func(filter string) bool { return strings.TrimSpace(filter) == constant.MetricsFilterKey })
	if metricsIndex < 0 || traceIndex <= metricsIndex {
		return chain
	}
	// Only move the automatic filter after expanding the default placeholder.
	if addTracing {
		name := filters[traceIndex]
		copy(filters[metricsIndex+1:traceIndex+1], filters[metricsIndex:traceIndex])
		filters[metricsIndex] = name
	} else {
		name := filters[metricsIndex]
		copy(filters[metricsIndex:traceIndex], filters[metricsIndex+1:traceIndex+1])
		filters[traceIndex] = name
	}
	return strings.Join(filters, ",")
}

func MergeValue(str1, str2, def string) string {
	if str1 == "" && str2 == "" {
		return def
	}
	s1 := strings.Split(str1, ",")
	s2 := strings.Split(str2, ",")
	str := "," + strings.Join(append(s1, s2...), ",")
	defKey := strings.Contains(str, ","+constant.DefaultKey)
	if !defKey {
		str = "," + constant.DefaultKey + str
	}
	str = strings.TrimPrefix(strings.ReplaceAll(str, ","+constant.DefaultKey, ","+def), ",")
	return removeMinus(strings.Split(str, ","))
}

func removeMinus(strArr []string) string {
	if len(strArr) == 0 {
		return ""
	}
	var normalStr string
	var minusStrArr []string
	for _, v := range strArr {
		if strings.HasPrefix(v, "-") {
			minusStrArr = append(minusStrArr, v[1:])
		} else {
			normalStr += fmt.Sprintf(",%s", v)
		}
	}
	normalStr = strings.Trim(normalStr, ",")
	for _, v := range minusStrArr {
		normalStr = strings.Replace(normalStr, v, "", 1)
	}
	reg := regexp.MustCompile("[,]+")
	normalStr = reg.ReplaceAllString(strings.Trim(normalStr, ","), ",")
	return normalStr
}

func IsValid(addr string) bool {
	return addr != "" && addr != constant.NotAvailable
}

// EnsureApplicationAttribute resolves application config from the primary URL,
// fallback URLs, URL params, then the default application name.
func EnsureApplicationAttribute(url *common.URL, fallbackURLs ...*common.URL) *global.ApplicationConfig {
	urls := make([]*common.URL, 0, len(fallbackURLs)+1)
	urls = append(urls, url)
	urls = append(urls, fallbackURLs...)

	for _, candidate := range urls {
		if application, ok := applicationFromAttribute(candidate); ok {
			setApplicationAttribute(url, application)
			return application
		}
	}

	for _, candidate := range urls {
		if application := applicationFromParam(candidate); application != nil {
			setApplicationAttribute(url, application)
			return application
		}
	}

	application := &global.ApplicationConfig{Name: constant.DefaultDubboApp}
	setApplicationAttribute(url, application)
	return application
}

func setApplicationAttribute(url *common.URL, application *global.ApplicationConfig) {
	if url != nil {
		url.SetAttribute(constant.ApplicationKey, application)
	}
}

func applicationFromAttribute(url *common.URL) (*global.ApplicationConfig, bool) {
	if url == nil {
		return nil, false
	}

	applicationRaw, ok := url.GetAttribute(constant.ApplicationKey)
	if !ok {
		return nil, false
	}

	switch application := applicationRaw.(type) {
	case *global.ApplicationConfig:
		if application != nil && application.Name != "" {
			return application, true
		}
	case global.ApplicationConfig:
		if application.Name != "" {
			applicationCopy := application
			return &applicationCopy, true
		}
	case string:
		if application != "" {
			return &global.ApplicationConfig{Name: application}, true
		}
	}
	return nil, false
}

func applicationFromParam(url *common.URL) *global.ApplicationConfig {
	if url == nil {
		return nil
	}
	if applicationName := url.GetParam(constant.ApplicationKey, ""); applicationName != "" {
		return &global.ApplicationConfig{Name: applicationName}
	}
	return nil
}
