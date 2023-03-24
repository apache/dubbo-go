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

package parser

import (
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/magiconair/properties"

	perrors "github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	ScopeApplication = "application"
	GeneralType      = "general"
)

// ConfigurationParser interface
type ConfigurationParser interface {
	Parse(string) (map[string]string, error)
	ParseToUrls(content string) ([]*common.URL, error)
}

// DefaultConfigurationParser for supporting properties file in config center
type DefaultConfigurationParser struct{}

// ConfiguratorConfig defines configurator config
type ConfiguratorConfig struct {
	ConfigVersion string       `yaml:"configVersion"`
	Scope         string       `yaml:"scope"`
	Key           string       `yaml:"key"`
	Enabled       bool         `yaml:"enabled"`
	Configs       []ConfigItem `yaml:"configs"`
}

// ConfigItem defines config item
type ConfigItem struct {
	Type              string            `yaml:"type"`
	Enabled           bool              `yaml:"enabled"`
	Addresses         []string          `yaml:"addresses"`
	ProviderAddresses []string          `yaml:"providerAddresses"`
	Services          []string          `yaml:"services"`
	Applications      []string          `yaml:"applications"`
	Parameters        map[string]string `yaml:"parameters"`
	Side              string            `yaml:"side"`
}

// Parse load content
func (parser *DefaultConfigurationParser) Parse(content string) (map[string]string, error) {
	pps, err := properties.LoadString(content)
	if err != nil {
		logger.Errorf("Parse the content {%v} in DefaultConfigurationParser error ,error message is {%v}", content, err)
		return nil, err
	}
	return pps.Map(), nil
}

// ParseToUrls is used to parse content to urls
func (parser *DefaultConfigurationParser) ParseToUrls(content string) ([]*common.URL, error) {
	config := ConfiguratorConfig{}
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, err
	}
	scope := config.Scope
	items := config.Configs
	var allUrls []*common.URL
	if scope == ScopeApplication {
		for _, v := range items {
			urls, err := appItemToUrls(v, config)
			if err != nil {
				return nil, err
			}
			allUrls = append(allUrls, urls...)
		}
	} else {
		for _, v := range items {
			urls, err := serviceItemToUrls(v, config)
			if err != nil {
				return nil, err
			}
			allUrls = append(allUrls, urls...)
		}
	}
	return allUrls, nil
}

// serviceItemToUrls is used to transfer item and config to urls
func serviceItemToUrls(item ConfigItem, config ConfiguratorConfig) ([]*common.URL, error) {
	addresses := item.Addresses
	if len(addresses) == 0 {
		addresses = append(addresses, constant.AnyHostValue)
	}
	var urls []*common.URL
	for _, v := range addresses {
		urlStr := constant.OverrideProtocol + "://" + v + "/"
		serviceStr, err := getServiceString(config.Key)
		if err != nil {
			return nil, perrors.WithStack(err)
		}
		urlStr = urlStr + serviceStr
		paramStr, err := getParamString(item)
		if err != nil {
			return nil, perrors.WithStack(err)
		}
		urlStr = urlStr + paramStr
		urlStr = urlStr + getEnabledString(item, config)
		urlStr = urlStr + "&category="
		urlStr = urlStr + constant.DynamicConfiguratorsCategory
		urlStr = urlStr + "&configVersion="
		urlStr = urlStr + config.ConfigVersion
		apps := item.Applications
		if len(apps) > 0 {
			for _, v := range apps {
				newUrlStr := urlStr
				newUrlStr = newUrlStr + "&application"
				newUrlStr = newUrlStr + v
				url, err := common.NewURL(newUrlStr)
				if err != nil {
					return nil, perrors.WithStack(err)
				}
				urls = append(urls, url)
			}
		} else {
			url, err := common.NewURL(urlStr)
			if err != nil {
				return nil, perrors.WithStack(err)
			}
			urls = append(urls, url)
		}
	}
	return urls, nil
}

// nolint
func appItemToUrls(item ConfigItem, config ConfiguratorConfig) ([]*common.URL, error) {
	addresses := item.Addresses
	if len(addresses) == 0 {
		addresses = append(addresses, constant.AnyHostValue)
	}
	var urls []*common.URL
	for _, v := range addresses {
		urlStr := constant.OverrideProtocol + "://" + v + "/"
		services := item.Services
		if len(services) == 0 {
			services = append(services, constant.AnyValue)
		}
		for _, vs := range services {
			serviceStr, err := getServiceString(vs)
			if err != nil {
				return nil, perrors.WithStack(err)
			}
			urlStr = urlStr + serviceStr
			paramStr, err := getParamString(item)
			if err != nil {
				return nil, perrors.WithStack(err)
			}
			urlStr = urlStr + paramStr
			urlStr = urlStr + "&application="
			urlStr = urlStr + config.Key
			urlStr = urlStr + getEnabledString(item, config)
			urlStr = urlStr + "&category="
			urlStr = urlStr + constant.AppDynamicConfiguratorsCategory
			urlStr = urlStr + "&configVersion="
			urlStr = urlStr + config.ConfigVersion
			url, err := common.NewURL(urlStr)
			if err != nil {
				return nil, perrors.WithStack(err)
			}
			urls = append(urls, url)
		}
	}
	return urls, nil
}

// getServiceString returns service string
func getServiceString(service string) (string, error) {
	if len(service) == 0 {
		return "", perrors.New("service field in configuration is null.")
	}
	var serviceStr string
	i := strings.Index(service, "/")
	if i > 0 {
		serviceStr = serviceStr + "group="
		serviceStr = serviceStr + service[0:i]
		serviceStr = serviceStr + "&"
		service = service[i+1:]
	}
	j := strings.Index(service, ":")
	if j > 0 {
		serviceStr = serviceStr + "version="
		serviceStr = serviceStr + service[j+1:]
		serviceStr = serviceStr + "&"
		service = service[0:j]
	}
	serviceStr = service + "?" + serviceStr
	return serviceStr, nil
}

// nolint
func getParamString(item ConfigItem) (string, error) {
	var retStr string
	retStr = retStr + "category="
	retStr = retStr + constant.DynamicConfiguratorsCategory
	if len(item.Side) > 0 {
		retStr = retStr + "&side="
		retStr = retStr + item.Side
	}
	params := item.Parameters
	if len(params) <= 0 {
		return "", perrors.New("Invalid configurator rule, please specify at least one parameter " +
			"you want to change in the rule.")
	}
	for k, v := range params {
		retStr += "&" + k + "=" + v
	}

	retStr += "&" + constant.OverrideProvidersKey + "=" + strings.Join(item.ProviderAddresses, ",")

	return retStr, nil
}

// getEnabledString returns enabled string
func getEnabledString(item ConfigItem, config ConfiguratorConfig) string {
	retStr := "&enabled="
	if len(item.Type) == 0 || item.Type == GeneralType {
		retStr = retStr + strconv.FormatBool(config.Enabled)
	} else {
		retStr = retStr + strconv.FormatBool(item.Enabled)
	}
	return retStr
}
