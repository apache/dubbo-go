package servicediscovery

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	metadataInstance "dubbo.apache.org/dubbo-go/v3/metadata/report/instance"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"strconv"
	"time"
)

type serviceDiscoveryMeta struct {
	metadataInfo *info.MetadataInfo
	instance     registry.ServiceInstance
}

func NewBaseServiceDiscovery(app string) serviceDiscoveryMeta {
	return serviceDiscoveryMeta{metadataInfo: info.NewMetadataInfWithApp(app)}
}

func (sd *serviceDiscoveryMeta) GetLocalMetadata() *info.MetadataInfo {
	return sd.metadataInfo
}

func (sd *serviceDiscoveryMeta) GetRemoteMetadata(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	// TODO cache
	return GetRemoteMetadata(revision, instance)
}

func (sd *serviceDiscoveryMeta) Register() error {
	revisionUpdated := sd.calOrUpdateInstanceRevision(sd.instance)
	if revisionUpdated {
		err := metadataInstance.GetMetadataReport().PublishAppMetadata(sd.metadataInfo.App, sd.metadataInfo.Revision, sd.metadataInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sd *serviceDiscoveryMeta) Unregister() error {
	return nil
}

func (sd *serviceDiscoveryMeta) calOrUpdateInstanceRevision(instance registry.ServiceInstance) bool {
	return true
}

//// registerServiceInstance register service instance
//func registerServiceInstance() {
//	p := extension.GetProtocol(constant.RegistryProtocol)
//	var rp registry.RegistryFactory
//	var ok bool
//	if rp, ok = p.(registry.RegistryFactory); !ok {
//		panic("dubbo registry protocol{" + reflect.TypeOf(p).String() + "} is invalid")
//	}
//	rs := rp.GetRegistries()
//	for _, r := range rs {
//		var sdr registry.ServiceDiscoveryHolder
//		if sdr, ok = r.(registry.ServiceDiscoveryHolder); !ok {
//			continue
//		}
//		// publish app level data to registry
//		err := sdr.GetServiceDiscovery().Register()
//		if err != nil {
//			panic(err)
//		}
//	}
//}

// // nolint
//func createInstance(url *common.URL) (registry.ServiceInstance, error) {
//	appConfig := GetApplicationConfig()
//	port, err := strconv.ParseInt(url.Port, 10, 32)
//	if err != nil {
//		return nil, perrors.WithMessage(err, "invalid port: "+url.Port)
//	}
//
//	host := url.Ip
//	if len(host) == 0 {
//		host = common.GetLocalIp()
//	}
//
//	// usually we will add more metadata
//	metadata := make(map[string]string, 8)
//	metadata[constant.MetadataStorageTypePropertyName] = appConfig.MetadataType
//
//	instance := &registry.DefaultServiceInstance{
//		ServiceName: appConfig.Name,
//		Host:        host,
//		Port:        int(port),
//		ID:          host + constant.KeySeparator + url.Port,
//		Enable:      true,
//		Healthy:     true,
//		Metadata:    metadata,
//		Tag:         appConfig.Tag,
//	}
//
//	for _, cus := range extension.GetCustomizers() {
//		cus.Customize(instance)
//	}
//
//	return instance, nil
//}

// GetMetadata get the medata info of service from report
func GetRemoteMetadata(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	meta, err := getMetadataFromCache(revision, instance)
	if err != nil || meta == nil {
		meta, err = getMetadataFromMetadataReport(revision, instance)
		if err != nil || meta == nil {
			meta, err = getMetadataFromRpc(revision, instance)
		}
		// TODO: need to update cache
	}
	return meta, err
}

func getMetadataFromCache(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	// TODO
	return nil, nil
}

func getMetadataFromMetadataReport(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	report := metadataInstance.GetMetadataReport()
	return report.GetAppMetadata(instance.GetServiceName(), revision)
}

func getMetadataFromRpc(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	url := common.NewURLWithOptions(
		common.WithProtocol(constant.Dubbo),
		common.WithIp(instance.GetHost()),
		common.WithPort(strconv.Itoa(instance.GetPort())),
	)
	url.SetParam(constant.SideKey, constant.Consumer)
	url.SetParam(constant.VersionKey, "1.0.0")
	url.SetParam(constant.InterfaceKey, constant.MetadataServiceName)
	url.SetParam(constant.GroupKey, instance.GetServiceName())
	service, err := createRpcClient(url)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5000))
	defer cancel()
	return service.GetMetadataInfo(ctx, revision)
}

type metadataService struct {
	GetExportedURLs       func(context context.Context, serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) `dubbo:"getExportedURLs"`
	GetMetadataInfo       func(context context.Context, revision string) (*info.MetadataInfo, error)                                                   `dubbo:"getMetadataInfo"`
	GetMetadataServiceURL func(context context.Context) (*common.URL, error)
	GetSubscribedURLs     func(context context.Context) ([]*common.URL, error)
	Version               func(context context.Context) (string, error)
}

func createRpcClient(url *common.URL) (*metadataService, error) {
	rpcService := &metadataService{}
	invoker := extension.GetProtocol(constant.Dubbo).Refer(url)
	proxy := extension.GetProxyFactory("").GetProxy(invoker, url)
	proxy.Implement(rpcService)
	return rpcService, nil
}
