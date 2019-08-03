package nacos

import (
	"bytes"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/registry"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	perrors "github.com/pkg/errors"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	localIP = ""
)

func init() {
	localIP, _ = utils.GetLocalIP()
	extension.SetRegistry("nacos", newNacosRegistry)
}

type nacosRegistry struct {
	*common.URL
	namingClient naming_client.INamingClient
}

func getNacosConfig(url *common.URL) (map[string]interface{}, error) {
	if url == nil {
		return nil, perrors.New("url is empty!")
	}
	if url.Location == "" {
		return nil, perrors.New("url.location is empty!")
	}
	configMap := make(map[string]interface{})

	var serverConfigs []nacosConstant.ServerConfig
	addresses := strings.Split(url.Location, ",")
	for _, addr := range addresses {
		ip, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, perrors.WithMessagef(err, "split [%s] ", addr)
		}
		port, _ := strconv.Atoi(portStr)
		serverConfigs = append(serverConfigs, nacosConstant.ServerConfig{
			IpAddr: ip,
			Port:   uint64(port),
		})
	}
	configMap["serverConfigs"] = serverConfigs

	var clientConfig nacosConstant.ClientConfig
	timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
	if err != nil {
		return nil, err
	}
	clientConfig.TimeoutMs = uint64(timeout.Seconds() * 1000)
	clientConfig.ListenInterval = 2 * clientConfig.TimeoutMs
	clientConfig.CacheDir = url.GetParam(constant.NACOS_CACHE_DIR_KEY, "")
	clientConfig.LogDir = url.GetParam(constant.NACOS_LOG_DIR_KEY, "")
	clientConfig.Endpoint = url.GetParam(constant.NACOS_ENDPOINT, "")
	clientConfig.NotLoadCacheAtStart = true
	configMap["clientConfig"] = clientConfig

	return configMap, nil
}

func newNacosRegistry(url *common.URL) (registry.Registry, error) {
	nacosConfig, err := getNacosConfig(url)
	if err != nil {
		return nil, err
	}
	client, err := clients.CreateNamingClient(nacosConfig)
	if err != nil {
		return nil, err
	}
	registry := nacosRegistry{
		URL:          url,
		namingClient: client,
	}
	return &registry, nil
}

func getCategory(url common.URL) string {
	role, _ := strconv.Atoi(url.GetParam(constant.ROLE_KEY, strconv.Itoa(constant.NACOS_DEFAULT_ROLETYPE)))
	category := common.DubboNodes[role]
	return category
}

func getServiceName(url common.URL) string {
	var buffer bytes.Buffer

	buffer.Write([]byte(getCategory(url)))
	appendParam(&buffer, url, constant.INTERFACE_KEY)
	appendParam(&buffer, url, constant.VERSION_KEY)
	appendParam(&buffer, url, constant.GROUP_KEY)
	return buffer.String()
}

func appendParam(target *bytes.Buffer, url common.URL, key string) {
	value := url.GetParam(key, "")
	if strings.TrimSpace(value) != "" {
		target.Write([]byte(constant.NACOS_SERVICE_NAME_SEPARATOR))
		target.Write([]byte(value))
	}
}

func createRegisterParam(url common.URL, serviceName string) vo.RegisterInstanceParam {
	category := getCategory(url)
	params := map[string]string{}
	for k, _ := range url.Params {
		params[k] = url.Params.Get(k)
	}
	params[constant.NACOS_CATEGORY_KEY] = category
	params[constant.NACOS_PROTOCOL_KEY] = url.Protocol
	params[constant.NACOS_PATH_KEY] = url.Path
	if url.Ip == "" {
		url.Ip = localIP
	}
	if url.Port == "" || url.Port == "0" {
		url.Port = "80"
	}
	port, _ := strconv.Atoi(url.Port)
	instance := vo.RegisterInstanceParam{
		Ip:          url.Ip,
		Port:        uint64(port),
		Metadata:    params,
		Weight:      1,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		ServiceName: serviceName,
	}
	return instance
}

func (nr *nacosRegistry) Register(url common.URL) error {
	serviceName := getServiceName(url)
	param := createRegisterParam(url, serviceName)
	isRegistry, err := nr.namingClient.RegisterInstance(param)
	if err != nil {
		return err
	}
	if !isRegistry {
		return perrors.New("registry to  nacos failed")
	}
	return nil
}

func (nr *nacosRegistry) Subscribe(conf common.URL) (registry.Listener, error) {
	return NewNacosListener(conf, nr.namingClient)
}

func (nr *nacosRegistry) GetUrl() common.URL {
	return *nr.URL
}

func (nr *nacosRegistry) IsAvailable() bool {
	return true
}

func (nr *nacosRegistry) Destroy() {
	return
}
