package consumer

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	gxstrings "github.com/dubbogo/gost/strings"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Consumer struct {
	invokerMap sync.Map
	opts       *Options
}

func (con *Consumer) consume(ctx context.Context, paramsRawVals []interface{}, interfaceName, methodName, consumeType string, opts ...ConsumeOption) (protocol.Result, error) {
	// get a default CallOptions
	// apply CallOption
	options := newDefaultConsumeOptions()
	for _, opt := range opts {
		opt(options)
	}

	var invoker protocol.Invoker
	val, ok := con.invokerMap.Load(interfaceName)
	if !ok {
		return nil, fmt.Errorf("%s interface is not be assembled", interfaceName)
	}
	invoker, ok = val.(protocol.Invoker)
	if !ok {
		panic(fmt.Sprintf("Consumer retrieves %s interface invoker failed", interfaceName))
	}
	inv, err := generateInvocation(methodName, paramsRawVals, consumeType, options)
	if err != nil {
		return nil, err
	}
	// todo: move timeout into context or invocation
	return invoker.Invoke(ctx, inv), nil

}

func (con *Consumer) ConsumeUnary(ctx context.Context, req, resp interface{}, interfaceName, methodName string, opts ...ConsumeOption) error {
	res, err := con.consume(ctx, []interface{}{req, resp}, interfaceName, methodName, constant.ConsumeUnary, opts...)
	if err != nil {
		return err
	}
	return res.Error()
}

func (con *Consumer) ConsumeClientStream(ctx context.Context, interfaceName, methodName string, opts ...ConsumeOption) (interface{}, error) {
	res, err := con.consume(ctx, nil, interfaceName, methodName, constant.ConsumeClientStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (con *Consumer) ConsumeServerStream(ctx context.Context, req interface{}, interfaceName, methodName string, opts ...ConsumeOption) (interface{}, error) {
	res, err := con.consume(ctx, []interface{}{req}, interfaceName, methodName, constant.ConsumeServerStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (con *Consumer) ConsumeBidiStream(ctx context.Context, interfaceName, methodName string, opts ...ConsumeOption) (interface{}, error) {
	res, err := con.consume(ctx, nil, interfaceName, methodName, constant.ConsumeBidiStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (con *Consumer) Assemble(interfaceName string, methodNames []string, cli interface{}) error {
	// todo: using ConsumerOptions

	// todo: move logic from ConsumerConfig.Load to Assemble
	// todo: move reference config to Consume
	//con.opts.id = registeredTypeName
	invoker := con.assemble(cli, interfaceName, methodNames)
	con.invokerMap.Store(interfaceName, invoker)
	// todo: add a option to tell Consumer that whether we should inject invocation logic into srv
	//refConfig.Implement(cli)

	//var maxWait int
	//
	//if maxWaitDuration, err := time.ParseDuration(con.cfg.MaxWaitTimeForServiceDiscovery); err != nil {
	//	logger.Warnf("Invalid consumer max wait time for service discovery: %s, fallback to 3s", con.cfg.MaxWaitTimeForServiceDiscovery)
	//	maxWait = 3
	//} else {
	//	maxWait = int(maxWaitDuration.Seconds())
	//}

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

func (con *Consumer) assemble(cli interface{}, interfaceName string, methodNames []string) protocol.Invoker {
	// todo: move to processing logic
	// If adaptive service is enabled,
	// the cluster and load balance should be overridden to "adaptivesvc" and "p2c" respectively.
	if con.opts.AdaptiveService {
		con.opts.Cluster = constant.ClusterKeyAdaptiveService
		con.opts.Loadbalance = constant.LoadBalanceKeyP2C
	}

	// cfgURL is an interface-level invoker url, in the other words, it represents an interface.
	cfgURL := common.NewURLWithOptions(
		common.WithPath(interfaceName),
		common.WithProtocol(con.opts.Protocol),
		// todo: change from config to string
		common.WithMethods(methodNames),
		common.WithParams(getURLMap(interfaceName, con.opts)),
		common.WithParamsValue(constant.BeanNameKey, interfaceName),
		common.WithParamsValue(constant.MetadataTypeKey, con.opts.metaDataType),
	)

	if con.opts.ForceTag {
		cfgURL.AddParam(constant.ForceUseTag, "true")
	}
	// todo: is there need to process this logic?
	//opts.postProcessConfig(cfgURL)

	// if mesh-enabled is set
	//updateOrCreateMeshURL(rc)

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
					serviceURL.Path = "/" + interfaceName
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
		urls = loadRegistries(con.opts.RegistryIDs, con.opts.Registries, common.CONSUMER)
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

func generateInvocation(methodName string, paramsRawVals []interface{}, consumeType string, opts *ConsumeOptions) (protocol.Invocation, error) {
	inv := invocation_impl.NewRPCInvocationWithOptions(
		invocation_impl.WithMethodName(methodName),
		//invocation_impl.WithArguments(inIArr),
		//invocation_impl.WithCallBack(p.callback),
		// todo:// withParameterTypes
		invocation_impl.WithParameterRawValues(paramsRawVals),
	)
	inv.SetAttribute(constant.ConsumeTypeKey, consumeType)

	return inv, nil
}

func getURLMap(interfaceName string, opts *Options) url.Values {
	urlMap := url.Values{}
	// first set user params
	for k, v := range opts.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.InterfaceKey, interfaceName)
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
	//urlMap.Set(constant.ApplicationKey, opts.rootConfig.Application.Name)
	//urlMap.Set(constant.OrganizationKey, opts.rootConfig.Application.Organization)
	//urlMap.Set(constant.NameKey, opts.rootConfig.Application.Name)
	//urlMap.Set(constant.ModuleKey, opts.rootConfig.Application.Module)
	//urlMap.Set(constant.AppVersionKey, opts.rootConfig.Application.Version)
	//urlMap.Set(constant.OwnerKey, opts.rootConfig.Application.Owner)
	//urlMap.Set(constant.EnvironmentKey, opts.rootConfig.Application.Environment)

	// filter
	defaultReferenceFilter := constant.DefaultReferenceFilters
	if opts.Generic != "" {
		defaultReferenceFilter = constant.GenericFilterKey + "," + defaultReferenceFilter
	}
	urlMap.Set(constant.ReferenceFilterKey, mergeValue(opts.Filter, "", defaultReferenceFilter))

	//for _, v := range opts.Methods {
	//	urlMap.Set("methods."+v.Name+"."+constant.LoadbalanceKey, v.LoadBalance)
	//	urlMap.Set("methods."+v.Name+"."+constant.RetriesKey, v.Retries)
	//	urlMap.Set("methods."+v.Name+"."+constant.StickyKey, strconv.FormatBool(v.Sticky))
	//	if len(v.RequestTimeout) != 0 {
	//		urlMap.Set("methods."+v.Name+"."+constant.TimeoutKey, v.RequestTimeout)
	//	}
	//}

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
func loadRegistries(registryIds []string, registries map[string]*RegistryOptions, roleType common.RoleType) []*common.URL {
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

func NewConsumer(opts ...Option) (*Consumer, error) {
	newOpts := &Options{}
	for _, opt := range opts {
		opt(newOpts)
	}
	return &Consumer{
		invokerMap: sync.Map{},
		opts:       newOpts,
	}, nil
}
