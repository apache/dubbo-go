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
	"crypto/sha512"
	"fmt"
	"net/url"
	"sort"
	"strconv"
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
	App                   string `json:"app,omitempty" hessian:"app"`
	Revision              string `json:"revision,omitempty" hessian:"revision"`
	Tag                   string
	Services              map[string]*ServiceInfo  `json:"services,omitempty" hessian:"services"`
	exportedServiceURLs   map[string][]*common.URL `hessian:"-"` // server exported service urls
	subscribedServiceURLs map[string][]*common.URL `hessian:"-"` // client subscribed service urls
}

func NewAppMetadataInfo(app string) *MetadataInfo {
	return NewMetadataInfo(app, "")
}

func NewMetadataInfo(app, tag string) *MetadataInfo {
	return &MetadataInfo{
		App:                   app,
		Tag:                   tag,
		Services:              make(map[string]*ServiceInfo),
		exportedServiceURLs:   make(map[string][]*common.URL),
		subscribedServiceURLs: make(map[string][]*common.URL),
	}
}

func NewMetadataInfoWithParams(app string, revision string, services map[string]*ServiceInfo) *MetadataInfo {
	return &MetadataInfo{
		App:                   app,
		Revision:              revision,
		Services:              services,
		exportedServiceURLs:   make(map[string][]*common.URL),
		subscribedServiceURLs: make(map[string][]*common.URL),
	}
}

func (info *MetadataInfo) JavaClassName() string {
	return "org.apache.dubbo.metadata.MetadataInfo"
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
			if u.URLEqual(url) {
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
	removeUrl(info.exportedServiceURLs, url)
	if replacement := info.findExportedServiceURL(service.GetMatchKey()); replacement != nil {
		info.Services[service.GetMatchKey()] = NewServiceInfoWithURL(replacement)
		return
	}
	delete(info.Services, service.GetMatchKey())
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

func (info *MetadataInfo) ReplaceExportedServices(urls []*common.URL) {
	info.Services = make(map[string]*ServiceInfo)
	info.exportedServiceURLs = make(map[string][]*common.URL)
	for _, serviceURL := range urls {
		info.AddService(serviceURL)
	}
}

// CalAndGetRevision calculates and updates the revision for this MetadataInfo.
// The revision is derived from the canonical ServiceInfo representation,
// ensuring strong binding between revision and serialized metadata content.
// Aligned with Java dubbo MetadataInfo.calAndGetRevision().
func (info *MetadataInfo) CalAndGetRevision() string {
	info.Revision = CalRevision(info.App, info.Services)
	return info.Revision
}

func (info *MetadataInfo) findExportedServiceURL(matchKey string) *common.URL {
	for _, urls := range info.exportedServiceURLs {
		for _, serviceURL := range urls {
			if NewServiceInfoWithURL(serviceURL).GetMatchKey() == matchKey {
				return serviceURL
			}
		}
	}
	return nil
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

	ServiceKey string      `json:"-" hessian:"-"`
	MatchKey   string      `json:"-" hessian:"-"`
	URL        *common.URL `json:"-" hessian:"-"`
}

func NewServiceInfoWithURL(url *common.URL) *ServiceInfo {
	service := NewServiceInfo(url.Service(), url.Group(), url.Version(), url.Protocol, url.Path, nil)
	service.Port, _ = strconv.Atoi(url.Port)
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
	p[constant.MethodsKey] = strings.Join(url.Methods, ",")
	service.Params = p
	return service
}

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

func (si *ServiceInfo) JavaClassName() string {
	return "org.apache.dubbo.metadata.MetadataInfo$ServiceInfo"
}

func (si *ServiceInfo) GetMethods() []string {
	s := si.Params[constant.MethodsKey]
	return strings.Split(s, ",")
}

func (si *ServiceInfo) GetParams() url.Values {
	v := url.Values{}
	methods := gxset.NewSet()
	if methodNames, ok := si.Params[constant.MethodsKey]; ok {
		for _, method := range strings.Split(methodNames, ",") {
			methods.Add(method)
		}
	}
	for k, p := range si.Params {
		ms := strings.Index(k, ".")
		if ms > 0 && methods.Contains(k[0:ms]) {
			v.Set("methods."+k, p)
		} else {
			v.Set(k, p)
		}
	}
	return v
}

func (si *ServiceInfo) GetMatchKey() string {
	if si.MatchKey != "" {
		return si.MatchKey
	}
	serviceKey := si.GetServiceKey()
	si.MatchKey = common.MatchKey(serviceKey, si.Protocol)
	return si.MatchKey
}

func (si *ServiceInfo) GetServiceKey() string {
	if si.ServiceKey != "" {
		return si.ServiceKey
	}
	si.ServiceKey = common.ServiceKey(si.Name, si.Group, si.Version)
	return si.ServiceKey
}

// toDescString returns a deterministic string representation of ServiceInfo
// for revision calculation. Aligned with Java dubbo ServiceInfo.toDescString().
//
// Format: name|group|version|protocol|port|path|params|methods
//
// Empty fields use "" as placeholder to keep separator count stable.
// Params are sorted by key alphabetically, joined as k=v&k=v.
// The "methods" key is excluded from params and appended separately.
// Methods are sorted alphabetically and comma-joined.
// No escaping is performed on param values (aligned with Java behavior).
func (si *ServiceInfo) toDescString() string {
	var b strings.Builder

	b.WriteString(si.Name)
	b.WriteByte('|')
	b.WriteString(si.Group)
	b.WriteByte('|')
	b.WriteString(si.Version)
	b.WriteByte('|')
	b.WriteString(si.Protocol)
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(si.Port))
	b.WriteByte('|')
	b.WriteString(si.Path)
	b.WriteByte('|')

	// params: sorted keys, exclude methods key
	keys := make([]string, 0, len(si.Params))
	for k := range si.Params {
		if k == constant.MethodsKey {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(si.Params[k])
	}

	b.WriteByte('|')

	// methods: sorted alphabetically, comma-joined
	if methodsStr, ok := si.Params[constant.MethodsKey]; ok && len(methodsStr) > 0 {
		methods := strings.Split(methodsStr, ",")
		sort.Strings(methods)
		for i, m := range methods {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(m)
		}
	}

	return b.String()
}

// CalRevision calculates a deterministic revision string from canonical ServiceInfo objects.
// Returns "0" if services is empty (aligned with Java EMPTY_REVISION).
// Services are sorted by matchKey before serialization to ensure deterministic output.
// The revision is a SHA-512 hex digest of: app + sorted toDescString of each ServiceInfo.
func CalRevision(app string, services map[string]*ServiceInfo) string {
	if len(services) == 0 {
		return "0"
	}

	// collect and sort matchKeys for deterministic iteration
	matchKeys := make([]string, 0, len(services))
	for mk := range services {
		matchKeys = append(matchKeys, mk)
	}
	sort.Strings(matchKeys)

	var b strings.Builder
	b.WriteString(app)
	for _, mk := range matchKeys {
		b.WriteString(services[mk].toDescString())
	}

	sum := sha512.Sum512([]byte(b.String()))
	return fmt.Sprintf("%x", sum)
}
