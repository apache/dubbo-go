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

package info

import (
	"fmt"
	"hash/crc32"
	"net/url"
	"sort"
	"strings"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	gxset "github.com/dubbogo/gost/container/set"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func init() {
	hessian.RegisterPOJO(&MetadataInfo{})
	hessian.RegisterPOJO(&ServiceInfo{})
}

var IncludeKeys = gxset.NewSet(
	constant.ApplicationKey,
	constant.GroupKey,
	constant.TimestampKey,
	constant.SerializationKey,
	constant.ClusterKey,
	constant.LoadbalanceKey,
	constant.PathKey,
	constant.TimeoutKey,
	constant.TokenKey,
	constant.VersionKey,
	constant.WarmupKey,
	constant.WeightKey,
	constant.ReleaseKey)

// MetadataInfo the metadata information of instance
type MetadataInfo struct {
	Reported              bool                     `json:"-"`
	App                   string                   `json:"app,omitempty" hessian:"app"`
	Revision              string                   `json:"revision,omitempty" hessian:"revision"`
	Services              map[string]*ServiceInfo  `json:"services,omitempty" hessian:"services"`
	exportedServiceURLs   map[string][]*common.URL // server exported service urls
	subscribedServiceURLs map[string][]*common.URL // client subscribed service urls
}

// nolint
func NewMetadataInfWithApp(app string) *MetadataInfo {
	return NewMetadataInfo(app, "", make(map[string]*ServiceInfo))
}

// nolint
func NewMetadataInfo(app string, revision string, services map[string]*ServiceInfo) *MetadataInfo {
	return &MetadataInfo{
		App:                   app,
		Revision:              revision,
		Services:              services,
		Reported:              false,
		exportedServiceURLs:   make(map[string][]*common.URL),
		subscribedServiceURLs: make(map[string][]*common.URL),
	}
}

// nolint
func (info *MetadataInfo) JavaClassName() string {
	return "org.apache.dubbo.metadata.MetadataInfo"
}

// CalAndGetRevision is different from Dubbo because golang doesn't support overload
// so that we should use interface + method name as identifier and ignore the method params
// in my opinion, it's enough because Dubbo actually ignore the URL params.
// please refer org.apache.dubbo.common.URL#toParameterString(java.lang.String...)
func (info *MetadataInfo) CalAndGetRevision() string {
	if info.Revision != "" && info.Reported {
		return info.Revision
	}
	if len(info.Services) == 0 {
		return "0"
	}
	candidates := make([]string, 0, 8)

	for _, s := range info.Services {
		iface := s.URL.GetParam(constant.InterfaceKey, "")
		ms := s.URL.Methods
		if len(ms) == 0 {
			candidates = append(candidates, iface)
		} else {
			for _, m := range ms {
				// methods are part of candidates
				candidates = append(candidates, iface+constant.KeySeparator+m)
			}
		}

		// append URL params if we need it
	}
	sort.Strings(candidates)

	// it's nearly impossible to be overflow
	res := uint64(0)
	for _, c := range candidates {
		res += uint64(crc32.ChecksumIEEE([]byte(c)))
	}
	info.Revision = fmt.Sprint(res)
	return info.Revision

}

// nolint
func (info *MetadataInfo) HasReported() bool {
	return info.Reported
}

// nolint
func (info *MetadataInfo) MarkReported() {
	info.Reported = true
}

// AddService add provider service info to MetadataInfo
func (info *MetadataInfo) AddService(url *common.URL) {
	service := NewServiceInfoWithURL(url)
	info.Services[service.GetMatchKey()] = service
	addUrl(info.exportedServiceURLs, url)
	if info.App == "" {
		info.App = url.GetParam(constant.ApplicationKey, "")
	}
}

func addUrl(m map[string][]*common.URL, url *common.URL) {
	if _, ok := m[url.ServiceKey()]; !ok {
		m[url.ServiceKey()] = make([]*common.URL, 0)
	}
	m[url.ServiceKey()] = append(m[url.ServiceKey()], url)
}

func removeUrl(m map[string][]*common.URL, url *common.URL) {
	if urls, ok := m[url.ServiceKey()]; ok {
		for i, u := range urls {
			if u == url {
				m[url.ServiceKey()] = deleteItem(urls, i)
				break
			}
		}
		if len(m[url.ServiceKey()]) == 0 {
			delete(m, url.ServiceKey())
		}
	}
}

func deleteItem(slice []*common.URL, index int) []*common.URL {
	copy(slice[index:], slice[index+1:])
	slice = slice[:len(slice)-1]
	return slice
}

func (info *MetadataInfo) RemoveService(url *common.URL) {
	service := NewServiceInfoWithURL(url)
	delete(info.Services, service.GetMatchKey())
	removeUrl(info.exportedServiceURLs, url)
}

// AddSubscribeURL client subscribe a service url
func (info *MetadataInfo) AddSubscribeURL(url *common.URL) {
	addUrl(info.subscribedServiceURLs, url)
}

// RemoveSubscribeURL client unsubscribe a service url
func (info *MetadataInfo) RemoveSubscribeURL(url *common.URL) {
	removeUrl(info.subscribedServiceURLs, url)
}

func (info *MetadataInfo) GetExportedServiceURLs() []*common.URL {
	res := make([]*common.URL, 0)
	for _, urls := range info.exportedServiceURLs {
		res = append(res, urls...)
	}
	return res
}

func (info *MetadataInfo) GetSubscribedURLs() []*common.URL {
	res := make([]*common.URL, 0)
	for _, urls := range info.subscribedServiceURLs {
		res = append(res, urls...)
	}
	return res
}

// ServiceInfo the information of service
type ServiceInfo struct {
	Name     string            `json:"name,omitempty" hessian:"name"`
	Group    string            `json:"group,omitempty" hessian:"group"`
	Version  string            `json:"version,omitempty" hessian:"version"`
	Protocol string            `json:"protocol,omitempty" hessian:"protocol"`
	Port     int               `json:"port,omitempty" hessian:"port"`
	Path     string            `json:"path,omitempty" hessian:"path"`
	Params   map[string]string `json:"params,omitempty" hessian:"params"`

	ServiceKey string      `json:"-"`
	MatchKey   string      `json:"-"`
	URL        *common.URL `json:"-"`
}

// nolint
func NewServiceInfoWithURL(url *common.URL) *ServiceInfo {
	service := NewServiceInfo(url.Service(), url.Group(), url.Version(), url.Protocol, url.Path, nil)
	service.URL = url
	// TODO includeKeys load dynamic
	p := make(map[string]string, 8)
	for _, keyInter := range IncludeKeys.Values() {
		key := keyInter.(string)
		value := url.GetParam(key, "")
		if len(value) != 0 {
			p[key] = value
		}
		for _, method := range url.Methods {
			value = url.GetMethodParam(method, key, "")
			if len(value) != 0 {
				p[method+"."+key] = value
			}
		}
	}
	service.Params = p
	return service
}

// nolint
func NewServiceInfo(name, group, version, protocol, path string, params map[string]string) *ServiceInfo {
	serviceKey := common.ServiceKey(name, group, version)
	matchKey := common.MatchKey(serviceKey, protocol)
	return &ServiceInfo{
		Name:       name,
		Group:      group,
		Version:    version,
		Protocol:   protocol,
		Path:       strings.TrimPrefix(path, "/"),
		Params:     params,
		ServiceKey: serviceKey,
		MatchKey:   matchKey,
	}
}

// nolint
func (si *ServiceInfo) JavaClassName() string {
	return "org.apache.dubbo.metadata.MetadataInfo$ServiceInfo"
}

// nolint
func (si *ServiceInfo) GetMethods() []string {
	if si.Params[constant.MethodsKey] != "" {
		s := si.Params[constant.MethodsKey]
		return strings.Split(s, ",")
	}
	methods := make([]string, 0, 8)
	for k, _ := range si.Params {
		ms := strings.Index(k, ".")
		if ms > 0 {
			methods = append(methods, k[0:ms])
		}
	}
	return methods
}

// nolint
func (si *ServiceInfo) GetParams() url.Values {
	v := url.Values{}
	for k, p := range si.Params {
		ms := strings.Index(k, ".")
		if ms > 0 {
			v.Set("methods."+k, p)
		} else {
			v.Set(k, p)
		}
	}
	return v
}

// nolint
func (si *ServiceInfo) GetMatchKey() string {
	if si.MatchKey != "" {
		return si.MatchKey
	}
	serviceKey := si.GetServiceKey()
	si.MatchKey = common.MatchKey(serviceKey, si.Protocol)
	return si.MatchKey
}

// nolint
func (si *ServiceInfo) GetServiceKey() string {
	if si.ServiceKey != "" {
		return si.ServiceKey
	}
	si.ServiceKey = common.ServiceKey(si.Name, si.Group, si.Version)
	return si.ServiceKey
}
