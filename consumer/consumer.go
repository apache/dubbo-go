package consumer

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	gxstrings "github.com/dubbogo/gost/strings"
	tripleConstant "github.com/dubbogo/triple/pkg/common/constant"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type consumer struct {
	invokerMap sync.Map
	opts       *Options
	rootCfg    *config.RootConfig
	cfg        *config.ConsumerConfig
}

func (con *consumer) Consume(ctx context.Context, opts ...ConsumeOption) error {
	// get a default CallOptions
	// apply CallOption
	options := con.defConsumeOpts
	for _, opt := range opts {
		opt(options)
	}
	// todo:// think about a identifier to refer invoker

	var invoker protocol.Invoker
	val, ok := con.invokerMap.Load("")
	if !ok {
		// create a new invoker and set it to invokerMap
	}
	invoker, ok = val.(protocol.Invoker)
	if !ok {
		panic("wrong type")
	}
	inv, err := callOptions2Invocation(options)
	if err != nil {
		return err
	}
	// todo: move timeout into context or invocation
	res := invoker.Invoke(ctx, inv)
	// todo: process result and error

}

func (con *consumer) Assemble(cli interface{}) error {
	// todo: using ConsumerOptions

	registeredTypeName := ""
	// todo: move logic from ConsumerConfig.Load to Assemble
	// todo: move reference config to Consume
	refConfig, ok := con.cfg.References[registeredTypeName]
	if !ok {
		// not found configuration, now new a configuration with default.
		refConfig = config.NewReferenceConfigBuilder().SetProtocol(tripleConstant.TRIPLE).Build()
		triplePBService, ok := cli.(common.TriplePBService)
		if !ok {
			logger.Errorf("Dubbo-go cannot get interface name with registeredTypeName = %s."+
				"Please run the command 'go install github.com/dubbogo/dubbogo-cli/cmd/protoc-gen-go-triple@latest' to get the latest "+
				"protoc-gen-go-triple,  and then re-generate your pb file again by this tool."+
				"If you are not using pb serialization, please set 'interfaceName' field in reference config to let dubbogo get the interface name.", registeredTypeName)
			return error()
		} else {
			// use interface name defined by pb
			refConfig.InterfaceName = triplePBService.XXX_InterfaceName()
		}
		if err := refConfig.Init(con.rootCfg); err != nil {
			logger.Errorf(fmt.Sprintf("reference with registeredTypeName = %s init failed! err: %#v", registeredTypeName, err))
			return err
		}
	}
	// todo: export id
	con.opts.id = registeredTypeName
	invoker := con.assemble(cli)
	con.invokerMap.Store(registeredTypeName, invoker)
	// todo: add a option to tell Consumer that whether we should inject invocation logic into srv
	refConfig.Implement(cli)

	var maxWait int

	if maxWaitDuration, err := time.ParseDuration(con.cfg.MaxWaitTimeForServiceDiscovery); err != nil {
		logger.Warnf("Invalid consumer max wait time for service discovery: %s, fallback to 3s", con.cfg.MaxWaitTimeForServiceDiscovery)
		maxWait = 3
	} else {
		maxWait = int(maxWaitDuration.Seconds())
	}

	// todo:// just wait for one reference

	// wait for invoker is available, if wait over default 3s, then panic
	//var count int
	//for {
	//	checkok := true
	//	for key, ref := range cfg.References {
	//		if (ref.Check != nil && *ref.Check && GetProviderService(key) == nil) ||
	//			(ref.Check == nil && cc.Check && GetProviderService(key) == nil) ||
	//			(ref.Check == nil && GetProviderService(key) == nil) { // default to true
	//
	//			if ref.invoker != nil && !ref.invoker.IsAvailable() {
	//				checkok = false
	//				count++
	//				if count > maxWait {
	//					errMsg := fmt.Sprintf("No provider available of the service %v.please check configuration.", ref.InterfaceName)
	//					logger.Error(errMsg)
	//					panic(errMsg)
	//				}
	//				time.Sleep(time.Second * 1)
	//				break
	//			}
	//			if ref.invoker == nil {
	//				logger.Warnf("The interface %s invoker not exist, may you should check your interface config.", ref.InterfaceName)
	//			}
	//		}
	//	}
	//	if checkok {
	//		break
	//	}
	//}
	return nil
}

func (con *consumer) assemble(cli interface{}) protocol.Invoker {
	// todo: move to processing logic
	// If adaptive service is enabled,
	// the cluster and load balance should be overridden to "adaptivesvc" and "p2c" respectively.
	if con.opts.AdaptiveService {
		con.opts.Cluster = constant.ClusterKeyAdaptiveService
		con.opts.Loadbalance = constant.LoadBalanceKeyP2C
	}

	// cfgURL is an interface-level invoker url, in the other words, it represents an interface.
	cfgURL := common.NewURLWithOptions(
		common.WithPath(con.opts.InterfaceName),
		common.WithProtocol(con.opts.Protocol),
		common.WithParams(con.opts.getURLMap()),
		common.WithParamsValue(constant.BeanNameKey, con.opts.id),
		common.WithParamsValue(constant.MetadataTypeKey, con.opts.metaDataType),
	)

	// todo: think about a more efficient way
	SetConsumerServiceByInterfaceName(con.opts.InterfaceName, cli)
	if con.opts.ForceTag {
		cfgURL.AddParam(constant.ForceUseTag, "true")
	}
	opts.postProcessConfig(cfgURL)

	// if mesh-enabled is set
	updateOrCreateMeshURL(rc)

	var urls []*common.URL

	// retrieving urls from config, and appending the urls to opts.urls
	if con.opts.URL != "" { // use user-specific urls
		/*
			 Two types of URL are allowed for opts.URL:
				1. direct url: server IP, that is, no need for a registry anymore
				2. registry url
			 They will be handled in different ways:
			 For example, we have a direct url and a registry url:
				1. "tri://localhost:10000" is a direct url
				2. "registry://localhost:2181" is a registry url.
			 Then, opts.URL looks like a string separated by semicolon: "tri://localhost:10000;registry://localhost:2181".
			 The result of urlStrings is a string array: []string{"tri://localhost:10000", "registry://localhost:2181"}.
		*/
		urlStrings := gxstrings.RegSplit(con.opts.URL, "\\s*[;]+\\s*")
		for _, urlStr := range urlStrings {
			serviceURL, err := common.NewURL(urlStr)
			if err != nil {
				panic(fmt.Sprintf("url configuration error,  please check your configuration, user specified URL %v refer error, error message is %v ", urlStr, err.Error()))
			}
			if serviceURL.Protocol == constant.RegistryProtocol { // serviceURL in this branch is a registry protocol
				serviceURL.SubURL = cfgURL
				urls = append(urls, serviceURL)
			} else { // serviceURL in this branch is the target endpoint IP address
				if serviceURL.Path == "" {
					serviceURL.Path = "/" + con.opts.InterfaceName
				}
				// replace params of serviceURL with params of cfgUrl
				// other stuff, e.g. IP, port, etc., are same as serviceURL
				newURL := common.MergeURL(serviceURL, cfgURL)
				newURL.AddParam("peer", "true")
				urls = append(urls, newURL)
			}
		}
	} else { // use registry configs
		// todo: move registries of rootConfig to Options
		urls = loadRegistries(con.opts.RegistryIDs, opts.rootConfig.Registries, common.CONSUMER)
		// set url to regURLs
		for _, regURL := range urls {
			regURL.SubURL = cfgURL
		}
	}

	// Get invokers according to opts.urls
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

		if con.opts.URL != "" {
			invoker = protocolwrapper.BuildInvokerChain(invoker, constant.ReferenceFilterKey)
		}

		invokers[i] = invoker
		if u.Protocol == constant.RegistryProtocol {
			regURL = u
		}
	}

	var finalInvoker protocol.Invoker
	// TODO(hxmhlt): decouple from directory, config should not depend on directory module
	if len(invokers) == 1 {
		finalInvoker = invokers[0]
		if con.opts.URL != "" {
			hitClu := constant.ClusterKeyFailover
			if u := finalInvoker.GetURL(); u != nil {
				hitClu = u.GetParam(constant.ClusterKey, constant.ClusterKeyZoneAware)
			}
			cluster, err := extension.GetCluster(hitClu)
			if err != nil {
				panic(err)
			} else {
				finalInvoker = cluster.Join(static.NewDirectory(invokers))
			}
		}
	} else {
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
			panic(err)
		} else {
			finalInvoker = cluster.Join(static.NewDirectory(invokers))
		}
	}

	// publish consumer's metadata
	publishServiceDefinition(cfgURL)
	// todo: there is no need to use proxy
	// create proxy
	//if opts.Async {
	//	callback := GetCallback(opts.id)
	//	opts.pxy = extension.GetProxyFactory(opts.rootConfig.Consumer.ProxyFactory).GetAsyncProxy(opts.invoker, callback, cfgURL)
	//} else {
	//	opts.pxy = extension.GetProxyFactory(opts.rootConfig.Consumer.ProxyFactory).GetProxy(opts.invoker, cfgURL)
	//}
	return finalInvoker
}

func callOptions2Invocation(opts *ConsumeOptions) (protocol.Invocation, error) {
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(""),
		invocation.WithArguments(nil),
		invocation.WithCallBack(nil),
		invocation.WithParameterValues(nil),
	)
	return inv, nil
}

func getURLMap(opts *Options) url.Values {
	urlMap := url.Values{}
	// first set user params
	for k, v := range opts.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.InterfaceKey, opts.InterfaceName)
	urlMap.Set(constant.TimestampKey, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.ClusterKey, opts.Cluster)
	urlMap.Set(constant.LoadbalanceKey, opts.Loadbalance)
	urlMap.Set(constant.RetriesKey, opts.Retries)
	urlMap.Set(constant.GroupKey, opts.Group)
	urlMap.Set(constant.VersionKey, opts.Version)
	urlMap.Set(constant.GenericKey, opts.Generic)
	urlMap.Set(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER))
	urlMap.Set(constant.ProvidedBy, opts.ProvidedBy)
	urlMap.Set(constant.SerializationKey, opts.Serialization)
	urlMap.Set(constant.TracingConfigKey, opts.TracingKey)

	urlMap.Set(constant.ReleaseKey, "dubbo-golang-"+constant.Version)
	urlMap.Set(constant.SideKey, (common.RoleType(common.CONSUMER)).Role())

	if len(opts.RequestTimeout) != 0 {
		urlMap.Set(constant.TimeoutKey, opts.RequestTimeout)
	}
	// getty invoke async or sync
	urlMap.Set(constant.AsyncKey, strconv.FormatBool(opts.Async))
	urlMap.Set(constant.StickyKey, strconv.FormatBool(opts.Sticky))

	// applicationConfig info
	urlMap.Set(constant.ApplicationKey, opts.rootConfig.Application.Name)
	urlMap.Set(constant.OrganizationKey, opts.rootConfig.Application.Organization)
	urlMap.Set(constant.NameKey, opts.rootConfig.Application.Name)
	urlMap.Set(constant.ModuleKey, opts.rootConfig.Application.Module)
	urlMap.Set(constant.AppVersionKey, opts.rootConfig.Application.Version)
	urlMap.Set(constant.OwnerKey, opts.rootConfig.Application.Owner)
	urlMap.Set(constant.EnvironmentKey, opts.rootConfig.Application.Environment)

	// filter
	defaultReferenceFilter := constant.DefaultReferenceFilters
	if opts.Generic != "" {
		defaultReferenceFilter = constant.GenericFilterKey + "," + defaultReferenceFilter
	}
	urlMap.Set(constant.ReferenceFilterKey, mergeValue(opts.Filter, "", defaultReferenceFilter))

	for _, v := range opts.Methods {
		urlMap.Set("methods."+v.Name+"."+constant.LoadbalanceKey, v.LoadBalance)
		urlMap.Set("methods."+v.Name+"."+constant.RetriesKey, v.Retries)
		urlMap.Set("methods."+v.Name+"."+constant.StickyKey, strconv.FormatBool(v.Sticky))
		if len(v.RequestTimeout) != 0 {
			urlMap.Set("methods."+v.Name+"."+constant.TimeoutKey, v.RequestTimeout)
		}
	}

	return urlMap
}

func mergeValue(str1, str2, def string) string {
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
	str = strings.TrimPrefix(strings.Replace(str, ","+constant.DefaultKey, ","+def, -1), ",")
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

// todo: change RegistryConfig to Options
func loadRegistries(registryIds []string, registries map[string]*RegistryConfig, roleType common.RoleType) []*common.URL {
	var registryURLs []*common.URL
	//trSlice := strings.Split(targetRegistries, ",")

	for k, registryConf := range registries {
		target := false

		// if user not config targetRegistries, default load all
		// Notice: in func "func Split(s, sep string) []string" comment:
		// if s does not contain sep and sep is not empty, SplitAfter returns
		// a slice of length 1 whose only element is s. So we have to add the
		// condition when targetRegistries string is not set (it will be "" when not set)
		if len(registryIds) == 0 || (len(registryIds) == 1 && registryIds[0] == "") {
			target = true
		} else {
			// else if user config targetRegistries
			for _, tr := range registryIds {
				if tr == k {
					target = true
					break
				}
			}
		}

		if target {
			if urls, err := registryConf.toURLs(roleType); err != nil {
				logger.Errorf("The registry id: %s url is invalid, error: %#v", k, err)
				panic(err)
			} else {
				registryURLs = append(registryURLs, urls...)
			}
		}
	}

	return registryURLs
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
