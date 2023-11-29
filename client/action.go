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

package client

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
	gxstrings "github.com/dubbogo/gost/strings"

	constant2 "github.com/dubbogo/triple/pkg/common/constant"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/common"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/proxy"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func updateOrCreateMeshURL(opts *ReferenceOptions) {
	ref := opts.Reference
	con := opts.Consumer

	if ref.URL != "" {
		logger.Infof("URL specified explicitly %v", ref.URL)
	}

	if !con.MeshEnabled {
		return
	}
	if ref.Protocol != constant2.TRIPLE {
		panic(fmt.Sprintf("Mesh mode enabled, Triple protocol expected but %v protocol found!", ref.Protocol))
	}
	if ref.ProvidedBy == "" {
		panic("Mesh mode enabled, provided-by should not be empty!")
	}

	podNamespace := getEnv(constant.PodNamespaceEnvKey, constant.DefaultNamespace)
	clusterDomain := getEnv(constant.ClusterDomainKey, constant.DefaultClusterDomain)

	var meshPort int
	if ref.MeshProviderPort > 0 {
		meshPort = ref.MeshProviderPort
	} else {
		meshPort = constant.DefaultMeshPort
	}

	ref.URL = "tri://" + ref.ProvidedBy + "." + podNamespace + constant.SVC + clusterDomain + ":" + strconv.Itoa(meshPort)
}

// ReferWithService retrieves invokers from urls.
func (refOpts *ReferenceOptions) ReferWithService(srv common.RPCService) {
	refOpts.refer(srv, nil)
}

func (refOpts *ReferenceOptions) ReferWithInfo(info *ClientInfo) {
	refOpts.refer(nil, info)
}

func (refOpts *ReferenceOptions) ReferWithServiceAndInfo(srv common.RPCService, info *ClientInfo) {
	refOpts.refer(srv, info)
}

func (refOpts *ReferenceOptions) refer(srv common.RPCService, info *ClientInfo) {
	ref := refOpts.Reference
	con := refOpts.Consumer

	var methods []string
	if info != nil {
		ref.InterfaceName = info.InterfaceName
		methods = info.MethodNames
		refOpts.id = info.InterfaceName
		refOpts.info = info
	} else if srv != nil {
		refOpts.id = common.GetReference(srv)
	} else {
		refOpts.id = ref.InterfaceName
	}
	// If adaptive service is enabled,
	// the cluster and load balance should be overridden to "adaptivesvc" and "p2c" respectively.
	if con != nil && con.AdaptiveService {
		ref.Cluster = constant.ClusterKeyAdaptiveService
		ref.Loadbalance = constant.LoadBalanceKeyP2C
	}

	// cfgURL is an interface-level invoker url, in the other words, it represents an interface.
	cfgURL := common.NewURLWithOptions(
		common.WithPath(ref.InterfaceName),
		common.WithProtocol(ref.Protocol),
		common.WithMethods(methods),
		common.WithParams(refOpts.getURLMap()),
		common.WithParamsValue(constant.BeanNameKey, refOpts.id),
		common.WithParamsValue(constant.MetadataTypeKey, refOpts.metaDataType),
	)
	if info != nil {
		cfgURL.SetAttribute(constant.ClientInfoKey, info)
	}

	if ref.ForceTag {
		cfgURL.AddParam(constant.ForceUseTag, "true")
	}
	refOpts.postProcessConfig(cfgURL)

	// if mesh-enabled is set
	updateOrCreateMeshURL(refOpts)

	// retrieving urls from config, and appending the urls to refOpts.urls
	urls, err := processURL(ref, refOpts.registriesCompat, cfgURL)
	if err != nil {
		panic(err)
	}

	// build invoker according to urls
	invoker, err := buildInvoker(urls, ref)
	if err != nil {
		panic(err)
	}
	refOpts.urls = urls
	refOpts.invoker = invoker

	// publish consumer's metadata
	publishServiceDefinition(cfgURL)
	// create proxy
	if info == nil && srv != nil {
		if ref.Async {
			var callback common.CallbackResponse
			if asyncSrv, ok := srv.(common.AsyncCallbackService); ok {
				callback = asyncSrv.CallBack
			}
			refOpts.pxy = extension.GetProxyFactory(con.ProxyFactory).GetAsyncProxy(refOpts.invoker, callback, cfgURL)
		} else {
			refOpts.pxy = extension.GetProxyFactory(con.ProxyFactory).GetProxy(refOpts.invoker, cfgURL)
		}
		refOpts.pxy.Implement(srv)
	}
	// this protocol would be destroyed in graceful_shutdown
	// please refer to (https://github.com/apache/dubbo-go/issues/2429)
	graceful_shutdown.RegisterProtocol(ref.Protocol)
}

func processURL(ref *global.ReferenceConfig, regsCompat map[string]*config.RegistryConfig, cfgURL *common.URL) ([]*common.URL, error) {
	var urls []*common.URL
	if ref.URL != "" { // use user-specific urls
		/*
			 Two types of URL are allowed for refOpts.URL:
				1. direct url: server IP, that is, no need for a registry anymore
				2. registry url
			 They will be handled in different ways:
			 For example, we have a direct url and a registry url:
				1. "tri://localhost:10000" is a direct url
				2. "registry://localhost:2181" is a registry url.
			 Then, refOpts.URL looks like a string separated by semicolon: "tri://localhost:10000;registry://localhost:2181".
			 The result of urlStrings is a string array: []string{"tri://localhost:10000", "registry://localhost:2181"}.
		*/
		urlStrings := gxstrings.RegSplit(ref.URL, "\\s*[;]+\\s*")
		for _, urlStr := range urlStrings {
			serviceURL, err := common.NewURL(urlStr, common.WithProtocol(ref.Protocol))
			if err != nil {
				return nil, fmt.Errorf(fmt.Sprintf("url configuration error,  please check your configuration, user specified URL %v refer error, error message is %v ", urlStr, err.Error()))
			}
			if serviceURL.Protocol == constant.RegistryProtocol { // serviceURL in this branch is a registry protocol
				serviceURL.SubURL = cfgURL
				urls = append(urls, serviceURL)
			} else { // serviceURL in this branch is the target endpoint IP address
				if serviceURL.Path == "" {
					serviceURL.Path = "/" + ref.InterfaceName
				}
				// replace params of serviceURL with params of cfgUrl
				// other stuff, e.g. IP, port, etc., are same as serviceURL
				newURL := serviceURL.MergeURL(cfgURL)
				newURL.AddParam("peer", "true")
				urls = append(urls, newURL)
			}
		}
	} else { // use registry configs
		urls = config.LoadRegistries(ref.RegistryIDs, regsCompat, common.CONSUMER)
		// set url to regURLs
		for _, regURL := range urls {
			regURL.SubURL = cfgURL
		}
	}
	return urls, nil
}

func buildInvoker(urls []*common.URL, ref *global.ReferenceConfig) (protocol.Invoker, error) {
	var (
		invoker protocol.Invoker
		regURL  *common.URL
	)
	invokers := make([]protocol.Invoker, len(urls))
	for i, u := range urls {
		if u.Protocol == constant.ServiceRegistryProtocol {
			invoker = extension.GetProtocol(constant.RegistryProtocol).Refer(u)
		} else {
			invoker = extension.GetProtocol(u.Protocol).Refer(u)
		}

		if ref.URL != "" {
			invoker = protocolwrapper.BuildInvokerChain(invoker, constant.ReferenceFilterKey)
		}

		if u.Protocol == constant.RegistryProtocol {
			regURL = u
		}
		invokers[i] = invoker
	}

	var resInvoker protocol.Invoker
	// TODO(hxmhlt): decouple from directory, config should not depend on directory module
	if len(invokers) == 1 {
		resInvoker = invokers[0]
		if ref.URL != "" {
			hitClu := constant.ClusterKeyFailover
			if u := resInvoker.GetURL(); u != nil {
				hitClu = u.GetParam(constant.ClusterKey, constant.ClusterKeyZoneAware)
			}
			cluster, err := extension.GetCluster(hitClu)
			if err != nil {
				return nil, err
			}
			resInvoker = cluster.Join(static.NewDirectory(invokers))
		}
		return resInvoker, nil
	}

	var hitClu string
	if regURL != nil {
		// for multi-subscription scenario, use 'zone-aware' policy by default
		hitClu = constant.ClusterKeyZoneAware
	} else {
		// not a registry url, must be direct invoke.
		hitClu = constant.ClusterKeyFailover
		if u := invokers[0].GetURL(); u != nil {
			hitClu = u.GetParam(constant.ClusterKey, constant.ClusterKeyZoneAware)
		}
	}
	cluster, err := extension.GetCluster(hitClu)
	if err != nil {
		return nil, err
	}
	resInvoker = cluster.Join(static.NewDirectory(invokers))

	return resInvoker, nil
}

func publishServiceDefinition(url *common.URL) {
	localService, err := extension.GetLocalMetadataService(constant.DefaultKey)
	if err != nil {
		logger.Warnf("get local metadata service failed, please check if you have imported _ \"dubbo.apache.org/dubbo-go/v3/metadata/service/local\"")
		return
	}
	localService.PublishServiceDefinition(url)
	if url.GetParam(constant.MetadataTypeKey, "") != constant.RemoteMetadataStorageType {
		return
	}
	if remoteMetadataService, err := extension.GetRemoteMetadataService(); err == nil && remoteMetadataService != nil {
		remoteMetadataService.PublishServiceDefinition(url)
	}
}

func (refOpts *ReferenceOptions) CheckAvailable() bool {
	ref := refOpts.Reference
	if refOpts.invoker == nil {
		logger.Warnf("The interface %s invoker not exist, may you should check your interface config.", ref.InterfaceName)
		return false
	}
	if !refOpts.invoker.IsAvailable() {
		return false
	}
	return true
}

// Implement
// @v is service provider implemented RPCService
func (refOpts *ReferenceOptions) Implement(v common.RPCService) {
	if refOpts.pxy != nil {
		refOpts.pxy.Implement(v)
	} else if refOpts.info != nil {
		refOpts.info.ConnectionInjectFunc(v, &Connection{
			refOpts: refOpts,
		})
	}
}

// GetRPCService gets RPCService from proxy
func (refOpts *ReferenceOptions) GetRPCService() common.RPCService {
	return refOpts.pxy.Get()
}

// GetProxy gets proxy
func (refOpts *ReferenceOptions) GetProxy() *proxy.Proxy {
	return refOpts.pxy
}

func (refOpts *ReferenceOptions) getURLMap() url.Values {
	ref := refOpts.Reference
	app := refOpts.applicationCompat
	metrics := refOpts.Metrics
	tracing := refOpts.Otel.TracingConfig

	urlMap := url.Values{}
	// first set user params
	for k, v := range ref.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.InterfaceKey, ref.InterfaceName)
	urlMap.Set(constant.TimestampKey, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.ClusterKey, ref.Cluster)
	urlMap.Set(constant.LoadbalanceKey, ref.Loadbalance)
	urlMap.Set(constant.RetriesKey, ref.Retries)
	urlMap.Set(constant.GroupKey, ref.Group)
	urlMap.Set(constant.VersionKey, ref.Version)
	urlMap.Set(constant.GenericKey, ref.Generic)
	urlMap.Set(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER))
	urlMap.Set(constant.ProvidedBy, ref.ProvidedBy)
	urlMap.Set(constant.SerializationKey, ref.Serialization)
	urlMap.Set(constant.TracingConfigKey, ref.TracingKey)

	urlMap.Set(constant.ReleaseKey, "dubbo-golang-"+constant.Version)
	urlMap.Set(constant.SideKey, (common.RoleType(common.CONSUMER)).Role())

	if len(ref.RequestTimeout) != 0 {
		urlMap.Set(constant.TimeoutKey, ref.RequestTimeout)
	}
	// getty invoke async or sync
	urlMap.Set(constant.AsyncKey, strconv.FormatBool(ref.Async))
	urlMap.Set(constant.StickyKey, strconv.FormatBool(ref.Sticky))

	// applicationConfig info
	if app != nil {
		urlMap.Set(constant.ApplicationKey, app.Name)
		urlMap.Set(constant.OrganizationKey, app.Organization)
		urlMap.Set(constant.NameKey, app.Name)
		urlMap.Set(constant.ModuleKey, app.Module)
		urlMap.Set(constant.AppVersionKey, app.Version)
		urlMap.Set(constant.OwnerKey, app.Owner)
		urlMap.Set(constant.EnvironmentKey, app.Environment)
	}

	// filter
	defaultReferenceFilter := constant.DefaultReferenceFilters
	if ref.Generic != "" {
		defaultReferenceFilter = constant.GenericFilterKey + "," + defaultReferenceFilter
	}
	if metrics.Enable != nil && *metrics.Enable {
		defaultReferenceFilter += fmt.Sprintf(",%s", constant.MetricsFilterKey)
	}
	if tracing.Enable != nil && *tracing.Enable {
		defaultReferenceFilter += fmt.Sprintf(",%s", constant.OTELClientTraceKey)
	}
	urlMap.Set(constant.ReferenceFilterKey, commonCfg.MergeValue(ref.Filter, "", defaultReferenceFilter))

	for _, v := range ref.Methods {
		urlMap.Set("methods."+v.Name+"."+constant.LoadbalanceKey, v.LoadBalance)
		urlMap.Set("methods."+v.Name+"."+constant.RetriesKey, v.Retries)
		urlMap.Set("methods."+v.Name+"."+constant.StickyKey, strconv.FormatBool(v.Sticky))
		if len(v.RequestTimeout) != 0 {
			urlMap.Set("methods."+v.Name+"."+constant.TimeoutKey, v.RequestTimeout)
		}
	}

	return urlMap
}

// todo: figure this out
//// GenericLoad ...
//func (opts *ReferenceOptions) GenericLoad(id string) {
//	genericService := generic.NewGenericService(opts.id)
//	config.SetConsumerService(genericService)
//	opts.id = id
//	opts.Refer(genericService)
//	opts.Implement(genericService)
//}

// GetInvoker get invoker from ReferenceConfigs
func (refOpts *ReferenceOptions) GetInvoker() protocol.Invoker {
	return refOpts.invoker
}

// postProcessConfig asks registered ConfigPostProcessor to post-process the current ReferenceConfigs.
func (refOpts *ReferenceOptions) postProcessConfig(url *common.URL) {
	for _, p := range extension.GetConfigPostProcessors() {
		p.PostProcessReferenceConfig(url)
	}
}
