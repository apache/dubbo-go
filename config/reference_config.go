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
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/failover"
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/zoneaware"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

import (
	"github.com/creasty/defaults"

	gxstrings "github.com/dubbogo/gost/strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/proxy"
	"dubbo.apache.org/dubbo-go/v3/config/generic"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
)

// ReferenceConfig is the configuration of service consumer
type ReferenceConfig struct {
	pxy            *proxy.Proxy
	id             string
	InterfaceName  string            `required:"true"  yaml:"interface"  json:"interface,omitempty" property:"interface"`
	Check          *bool             `default:"false" yaml:"check"  json:"check,omitempty" property:"check"`
	URL            string            `yaml:"url"  json:"url,omitempty" property:"url"`
	Filter         string            `yaml:"filter" json:"filter,omitempty" property:"filter"`
	Protocol       string            `default:"dubbo"  yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	RegistryIDs    []string          `yaml:"registryIDs"  json:"registryIDs,omitempty"  property:"registryIDs"`
	Cluster        string            `yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	Loadbalance    string            `yaml:"loadbalance"  json:"loadbalance,omitempty" property:"loadbalance"`
	Retries        string            `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	Group          string            `yaml:"group"  json:"group,omitempty" property:"group"`
	Version        string            `yaml:"version"  json:"version,omitempty" property:"version"`
	Serialization  string            `yaml:"serialization" json:"serialization" property:"serialization"`
	ProvidedBy     string            `yaml:"provided_by"  json:"provided_by,omitempty" property:"provided_by"`
	Methods        []*MethodConfig   `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	Async          bool              `yaml:"async"  json:"async,omitempty" property:"async"`
	Params         map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	invoker        protocol.Invoker
	urls           []*common.URL
	Generic        string `yaml:"generic"  json:"generic,omitempty" property:"generic"`
	Sticky         bool   `yaml:"sticky"   json:"sticky,omitempty" property:"sticky"`
	RequestTimeout string `yaml:"timeout"  json:"timeout,omitempty" property:"timeout"`
	ForceTag       bool   `yaml:"force.tag"  json:"force.tag,omitempty" property:"force.tag"`

	rootConfig   *RootConfig
	metaDataType string
}

// nolint
func (rc *ReferenceConfig) Prefix() string {
	return constant.ReferenceConfigPrefix + rc.InterfaceName + "."
}

func (rc *ReferenceConfig) Init(root *RootConfig) error {
	for _, method := range rc.Methods {
		if err := method.Init(); err != nil {
			return err
		}
	}
	if err := defaults.Set(rc); err != nil {
		return err
	}
	rc.rootConfig = root
	if root.Application != nil {
		rc.metaDataType = root.Application.MetadataType
	}
	if rc.Cluster == "" {
		rc.Cluster = "failover"
	}
	rc.RegistryIDs = translateRegistryIds(rc.RegistryIDs)
	if len(rc.RegistryIDs) <= 0 {
		rc.RegistryIDs = root.Consumer.RegistryIDs
	}
	return verify(rc)
}

// Refer ...
func (rc *ReferenceConfig) Refer(srv interface{}) {
	cfgURL := common.NewURLWithOptions(
		common.WithPath(rc.InterfaceName),
		common.WithProtocol(rc.Protocol),
		common.WithParams(rc.getURLMap()),
		common.WithParamsValue(constant.BEAN_NAME_KEY, rc.id),
		common.WithParamsValue(constant.METADATATYPE_KEY, rc.metaDataType),
	)
	SetConsumerServiceByInterfaceName(rc.InterfaceName, srv)
	if rc.ForceTag {
		cfgURL.AddParam(constant.ForceUseTag, "true")
	}
	rc.postProcessConfig(cfgURL)

	// retrieving urls from config, and appending the urls to rc.urls
	if rc.URL != "" { // use user-specific urls
		/*
		 Two types of URL are allowed for rc.URL: direct url and registry url, they will be handled in different ways.
		 For example, "tri://localhost:10000" is a direct url, and "registry://localhost:2181" is a registry url.
		 rc.URL: "tri://localhost:10000;tri://localhost:10001;registry://localhost:2181",
		 urlStrings = []string{"tri://localhost:10000", "tri://localhost:10001", "registry://localhost:2181"}.
		*/
		urlStrings := gxstrings.RegSplit(rc.URL, "\\s*[;]+\\s*")
		for _, urlStr := range urlStrings {
			serviceURL, err := common.NewURL(urlStr)
			if err != nil {
				panic(fmt.Sprintf("url configuration error,  please check your configuration, user specified URL %v refer error, error message is %v ", urlStr, err.Error()))
			}
			if serviceURL.Protocol == constant.REGISTRY_PROTOCOL { // URL stands for a registry protocol
				serviceURL.SubURL = cfgURL
				rc.urls = append(rc.urls, serviceURL)
			} else { // URL stands for a direct address
				if serviceURL.Path == "" {
					serviceURL.Path = "/" + rc.InterfaceName
				}
				// merge URL param with cfgURL, others are same as serviceURL
				newURL := common.MergeURL(serviceURL, cfgURL)
				rc.urls = append(rc.urls, newURL)
			}
		}
	} else { // use registry configs
		rc.urls = loadRegistries(rc.RegistryIDs, rc.rootConfig.Registries, common.CONSUMER)
		// set url to regURLs
		for _, regURL := range rc.urls {
			regURL.SubURL = cfgURL
		}
	}

	// Get invokers according to rc.urls
	var (
		invoker protocol.Invoker
		regURL  *common.URL
	)
	invokers := make([]protocol.Invoker, len(rc.urls))
	for i, u := range rc.urls {
		if u.Protocol == constant.SERVICE_REGISTRY_PROTOCOL {
			invoker = extension.GetProtocol("registry").Refer(u)
		} else {
			invoker = extension.GetProtocol(u.Protocol).Refer(u)
		}

		if rc.URL != "" {
			invoker = protocolwrapper.BuildInvokerChain(invoker, constant.REFERENCE_FILTER_KEY)
		}

		invokers[i] = invoker
		if u.Protocol == constant.REGISTRY_PROTOCOL {
			regURL = u
		}
	}

	// TODO(hxmhlt): decouple from directory, config should not depend on directory module
	if len(invokers) == 1 {
		rc.invoker = invokers[0]
		if rc.URL != "" {
			hitClu := failover.Key
			if u := rc.invoker.GetURL(); u != nil {
				hitClu = u.GetParam(constant.CLUSTER_KEY, zoneaware.Key)
			}
			rc.invoker = extension.GetCluster(hitClu).Join(static.NewDirectory(invokers))
		}
	} else {
		var hitClu string
		if regURL != nil {
			// for multi-subscription scenario, use 'zone-aware' policy by default
			hitClu = zoneaware.Key
		} else {
			// not a registry url, must be direct invoke.
			hitClu = failover.Key
			if u := invokers[0].GetURL(); u != nil {
				hitClu = u.GetParam(constant.CLUSTER_KEY, zoneaware.Key)
			}
		}
		rc.invoker = extension.GetCluster(hitClu).Join(static.NewDirectory(invokers))
	}

	// publish consumer's metadata
	publishServiceDefinition(cfgURL)
	// create proxy
	if rc.Async {
		callback := GetCallback(rc.id)
		rc.pxy = extension.GetProxyFactory(rc.rootConfig.Consumer.ProxyFactory).GetAsyncProxy(rc.invoker, callback, cfgURL)
	} else {
		rc.pxy = extension.GetProxyFactory(rc.rootConfig.Consumer.ProxyFactory).GetProxy(rc.invoker, cfgURL)
	}
}

// Implement
// @v is service provider implemented RPCService
func (rc *ReferenceConfig) Implement(v common.RPCService) {
	rc.pxy.Implement(v)
}

// GetRPCService gets RPCService from proxy
func (rc *ReferenceConfig) GetRPCService() common.RPCService {
	return rc.pxy.Get()
}

// GetProxy gets proxy
func (rc *ReferenceConfig) GetProxy() *proxy.Proxy {
	return rc.pxy
}

func (rc *ReferenceConfig) getURLMap() url.Values {
	urlMap := url.Values{}
	// first set user params
	for k, v := range rc.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.INTERFACE_KEY, rc.InterfaceName)
	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, rc.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, rc.Loadbalance)
	urlMap.Set(constant.RETRIES_KEY, rc.Retries)
	urlMap.Set(constant.GROUP_KEY, rc.Group)
	urlMap.Set(constant.VERSION_KEY, rc.Version)
	urlMap.Set(constant.GENERIC_KEY, rc.Generic)
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.CONSUMER))
	urlMap.Set(constant.PROVIDED_BY, rc.ProvidedBy)
	urlMap.Set(constant.SERIALIZATION_KEY, rc.Serialization)

	urlMap.Set(constant.RELEASE_KEY, "dubbo-golang-"+constant.Version)
	urlMap.Set(constant.SIDE_KEY, (common.RoleType(common.CONSUMER)).Role())

	if len(rc.RequestTimeout) != 0 {
		urlMap.Set(constant.TIMEOUT_KEY, rc.RequestTimeout)
	}
	// getty invoke async or sync
	urlMap.Set(constant.ASYNC_KEY, strconv.FormatBool(rc.Async))
	urlMap.Set(constant.STICKY_KEY, strconv.FormatBool(rc.Sticky))

	// applicationConfig info
	urlMap.Set(constant.APPLICATION_KEY, rc.rootConfig.Application.Name)
	urlMap.Set(constant.ORGANIZATION_KEY, rc.rootConfig.Application.Organization)
	urlMap.Set(constant.NAME_KEY, rc.rootConfig.Application.Name)
	urlMap.Set(constant.MODULE_KEY, rc.rootConfig.Application.Module)
	urlMap.Set(constant.APP_VERSION_KEY, rc.rootConfig.Application.Version)
	urlMap.Set(constant.OWNER_KEY, rc.rootConfig.Application.Owner)
	urlMap.Set(constant.ENVIRONMENT_KEY, rc.rootConfig.Application.Environment)

	// filter
	defaultReferenceFilter := constant.DEFAULT_REFERENCE_FILTERS
	if rc.Generic != "" {
		defaultReferenceFilter = constant.GENERIC_REFERENCE_FILTERS + "," + defaultReferenceFilter
	}
	urlMap.Set(constant.REFERENCE_FILTER_KEY, mergeValue(rc.rootConfig.Consumer.Filter, "", defaultReferenceFilter))

	for _, v := range rc.Methods {
		urlMap.Set("methods."+v.Name+"."+constant.LOADBALANCE_KEY, v.LoadBalance)
		urlMap.Set("methods."+v.Name+"."+constant.RETRIES_KEY, v.Retries)
		urlMap.Set("methods."+v.Name+"."+constant.STICKY_KEY, strconv.FormatBool(v.Sticky))
		if len(v.RequestTimeout) != 0 {
			urlMap.Set("methods."+v.Name+"."+constant.TIMEOUT_KEY, v.RequestTimeout)
		}
	}

	return urlMap
}

// GenericLoad ...
func (rc *ReferenceConfig) GenericLoad(id string) {
	genericService := generic.NewGenericService(rc.id)
	SetConsumerService(genericService)
	rc.id = id
	rc.Refer(genericService)
	rc.Implement(genericService)
}

// GetInvoker get invoker from ReferenceConfig
func (rc *ReferenceConfig) GetInvoker() protocol.Invoker {
	return rc.invoker
}

// postProcessConfig asks registered ConfigPostProcessor to post-process the current ReferenceConfig.
func (rc *ReferenceConfig) postProcessConfig(url *common.URL) {
	for _, p := range extension.GetConfigPostProcessors() {
		p.PostProcessReferenceConfig(url)
	}
}

//////////////////////////////////// reference config api

// newEmptyReferenceConfig returns empty ReferenceConfig
func newEmptyReferenceConfig() *ReferenceConfig {
	newReferenceConfig := &ReferenceConfig{}
	newReferenceConfig.Methods = make([]*MethodConfig, 0, 8)
	newReferenceConfig.Params = make(map[string]string, 8)
	return newReferenceConfig
}

type ReferenceConfigBuilder struct {
	referenceConfig *ReferenceConfig
}

func NewReferenceConfigBuilder() *ReferenceConfigBuilder {
	return &ReferenceConfigBuilder{referenceConfig: newEmptyReferenceConfig()}
}

func (pcb *ReferenceConfigBuilder) SetInterface(interfaceName string) *ReferenceConfigBuilder {
	pcb.referenceConfig.InterfaceName = interfaceName
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetRegistryIDs(registryIDs ...string) *ReferenceConfigBuilder {
	pcb.referenceConfig.RegistryIDs = registryIDs
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetGeneric(generic bool) *ReferenceConfigBuilder {
	if generic {
		pcb.referenceConfig.Generic = "true"
	} else {
		pcb.referenceConfig.Generic = "false"
	}
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetCluster(cluster string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Cluster = cluster
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetSerialization(serialization string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Serialization = serialization
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetProtocol(protocol string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Protocol = protocol
	return pcb
}

func (pcb *ReferenceConfigBuilder) Build() *ReferenceConfig {
	return pcb.referenceConfig
}
