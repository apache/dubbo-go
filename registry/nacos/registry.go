package nacos

import (
	"bytes"
	"net"
	"strconv"
	"strings"
	"time"
)
import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/registry"
)

var (
	localIP = ""
)

func init() {
	localIP, _ = utils.GetLocalIP()
	extension.SetRegistry(constant.NACOS_KEY, newNacosRegistry)
}

type nacosRegistry struct {
	*common.URL
	namingClient naming_client.INamingClient
}

func getNacosConfig(url *common.URL) (map[string]interface{}, error) {
	if url == nil {
		return nil, perrors.New("url is empty!")
	}
	if len(url.Location) == 0 {
		return nil, perrors.New("url.location is empty!")
	}
	configMap := make(map[string]interface{}, 2)

	addresses := strings.Split(url.Location, ",")
	serverConfigs := make([]nacosConstant.ServerConfig, 0, len(addresses))
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
	params := make(map[string]string, len(url.Params)+3)
	for k := range url.Params {
		params[k] = url.Params.Get(k)
	}
	params[constant.NACOS_CATEGORY_KEY] = category
	params[constant.NACOS_PROTOCOL_KEY] = url.Protocol
	params[constant.NACOS_PATH_KEY] = url.Path
	if len(url.Ip) == 0 {
		url.Ip = localIP
	}
	if len(url.Port) == 0 || url.Port == "0" {
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
		return perrors.New("registry [" + serviceName + "] to  nacos failed")
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
