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

package common

import (
	"fmt"
	"github.com/apache/dubbo-go/common/constant"
	gxset "github.com/dubbogo/gost/container/set"
	"go.uber.org/atomic"
	"hash/crc32"
	"net/url"
	"sort"
	"strings"
)

var IncludeKeys = gxset.NewSet(
	constant.APPLICATION_KEY,
	constant.GROUP_KEY, constant.TIMESTAMP_KEY, constant.SERIALIZATION_KEY, constant.CLUSTER_KEY,
	constant.LOADBALANCE_KEY, constant.PATH_KEY, constant.TIMEOUT_KEY,
	constant.TOKEN_KEY, constant.VERSION_KEY, constant.WARMUP_KEY,
	constant.WEIGHT_KEY, constant.RELEASE_KEY)

type MetadataInfo struct {
	App      string                  `json:"app"`
	Revision string                  `json:"revision"`
	Services map[string]*ServiceInfo `json:"services"`

	reported *atomic.Bool `json:"-"`
}

func NewMetadataInfWithApp(app string) *MetadataInfo {
	return NewMetadataInfo(app, "", make(map[string]*ServiceInfo))
}

func NewMetadataInfo(app string, revision string, services map[string]*ServiceInfo) *MetadataInfo {
	return &MetadataInfo{
		App:      app,
		Revision: revision,
		Services: services,
		reported: atomic.NewBool(false),
	}
}

// CalAndGetRevision is different from Dubbo because golang doesn't support overload
// so that we could use interface + method name as identifier and ignore the method params
// per my understanding, it's enough because Dubbo actually ignore the url params.
// please refer org.apache.dubbo.common.URL#toParameterString(java.lang.String...)
func (mi *MetadataInfo) CalAndGetRevision() string {
	if mi.Revision != "" && mi.reported.Load() {
		return mi.Revision
	}
	if len(mi.Services) == 0 {
		return "0"
	}
	candidates := make([]string, 8)

	for _, s := range mi.Services {
		sk := s.serviceKey
		ms := s.url.Methods
		if len(ms) == 0 {
			candidates = append(candidates, sk)
		} else {
			for _, m := range ms {
				// methods are part of candidates
				candidates = append(candidates, sk+constant.KEY_SEPARATOR+m)
			}
		}

		// append url params if we need it
	}
	sort.Strings(candidates)

	// it's nearly impossible to be overflow
	res := uint64(0)
	for _, c := range candidates {
		res += uint64(crc32.ChecksumIEEE([]byte(c)))
	}
	mi.Revision = fmt.Sprint(res)
	return mi.Revision

}

func (mi *MetadataInfo) HasReported() bool {
	return mi.reported.Load()
}

func (mi *MetadataInfo) MarkReported() {
	mi.reported.CAS(false, true)
}

func (mi *MetadataInfo) AddService(service *ServiceInfo) {
	if service == nil {
		return
	}
	mi.Services[service.GetMatchKey()] = service
}

func (mi *MetadataInfo) RemoveService(service *ServiceInfo) {
	if service == nil {
		return
	}
	delete(mi.Services, service.matchKey)
}

type ServiceInfo struct {
	Name     string            `json:"name"`
	Group    string            `json:"group"`
	Version  string            `json:"version"`
	Protocol string            `json:"protocol"`
	Path     string            `json:"path"`
	Params   map[string]string `json:"params"`

	serviceKey string `json:"-"`
	matchKey   string `json:"-"`
	url        *URL   `json:"-"`
}

func NewServiceInfoWithUrl(url *URL) *ServiceInfo {
	service := NewServiceInfo(url.Service(), url.Group(), url.Version(), url.Protocol, url.Path, nil)
	service.url = url
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

func NewServiceInfo(name string, group string, version string, protocol string, path string, params map[string]string) *ServiceInfo {
	serviceKey := ServiceKey(name, group, version)
	matchKey := MatchKey(serviceKey, protocol)
	return &ServiceInfo{
		Name:       name,
		Group:      group,
		Version:    version,
		Protocol:   protocol,
		Path:       path,
		Params:     params,
		serviceKey: serviceKey,
		matchKey:   matchKey,
	}
}

func (si *ServiceInfo) GetMethods() []string {
	if si.Params[constant.METHODS_KEY] != "" {
		s := si.Params[constant.METHODS_KEY]
		return strings.Split(s, ",")
	}
	methods := make([]string, 8)
	for k, _ := range si.Params {
		ms := strings.Index(k, ".")
		if ms > 0 {
			methods = append(methods, k[0:ms])
		}
	}
	return methods
}

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

func (si *ServiceInfo) GetMatchKey() string {
	if si.matchKey != "" {
		return si.matchKey
	}
	serviceKey := si.GetServiceKey()
	si.matchKey = MatchKey(serviceKey, si.Protocol)
	return si.matchKey
}

func (si *ServiceInfo) GetServiceKey() string {
	if si.serviceKey != "" {
		return si.serviceKey
	}
	si.serviceKey = ServiceKey(si.Name, si.Group, si.Version)
	return si.serviceKey
}
