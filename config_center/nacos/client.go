package nacos

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	nacosconst "github.com/nacos-group/nacos-sdk-go/common/constant"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

const (
	ConnDelay    = 3
	MaxFailTimes = 15
)

type NacosClient struct {
	name       string
	NacosAddrs []string
	sync.Mutex // for Client
	Client     *config_client.IConfigClient
	exit       chan struct{}
	Timeout    time.Duration
	once       sync.Once
	onceClose  func()
}

type Option func(*Options)

type Options struct {
	nacosName string
	client    *NacosClient
}

func WithNacosName(name string) Option {
	return func(opt *Options) {
		opt.nacosName = name
	}
}

func ValidateNacosClient(container nacosClientFacade, opts ...Option) error {
	var (
		err error
	)
	opions := &Options{}
	for _, opt := range opts {
		opt(opions)
	}

	err = nil

	lock := container.NacosClientLock()
	url := container.GetUrl()

	lock.Lock()
	defer lock.Unlock()

	if container.NacosClient() == nil {
		//in dubbo ,every registry only connect one node ,so this is []string{r.Address}
		timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
		if err != nil {
			logger.Errorf("timeout config %v is invalid ,err is %v",
				url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), err.Error())
			return perrors.WithMessagef(err, "newNacosClient(address:%+v)", url.Location)
		}
		nacosAddresses := strings.Split(url.Location, ",")
		newClient, err := newNacosClient(opions.nacosName, nacosAddresses, timeout)
		if err != nil {
			logger.Warnf("newNacosClient(name{%s}, nacos address{%v}, timeout{%d}) = error{%v}",
				opions.nacosName, url.Location, timeout.String(), err)
			return perrors.WithMessagef(err, "newNacosClient(address:%+v)", url.Location)
		}
		container.SetNacosClient(newClient)
	}

	if container.NacosClient().Client == nil {
		svrConfList := []nacosconst.ServerConfig{}
		for _, nacosAddr := range container.NacosClient().NacosAddrs {
			split := strings.Split(nacosAddr, ":")
			port, err := strconv.ParseUint(split[1], 10, 64)
			if err != nil {
				logger.Warnf("nacos addr port parse error ,error message is %v", err)
				continue
			}
			svrconf := nacosconst.ServerConfig{
				IpAddr: split[0],
				Port:   port,
			}
			svrConfList = append(svrConfList, svrconf)
		}

		client, err := clients.CreateConfigClient(map[string]interface{}{
			"serverConfigs": svrConfList,
			"clientConfig": nacosconst.ClientConfig{
				TimeoutMs:           uint64(int32(container.NacosClient().Timeout / time.Millisecond)),
				ListenInterval:      10000,
				NotLoadCacheAtStart: true,
				LogDir:              "logs/nacos/log",
			},
		})
		container.NacosClient().Client = &client
		if err != nil {
			logger.Errorf("nacos create config client error:%v", err)
		}
	}

	return perrors.WithMessagef(err, "newNacosClient(address:%+v)", url.PrimitiveURL)
}

func newNacosClient(name string, nacosAddrs []string, timeout time.Duration) (*NacosClient, error) {
	var (
		err error
		n   *NacosClient
	)

	n = &NacosClient{
		name:       name,
		NacosAddrs: nacosAddrs,
		Timeout:    timeout,
		exit:       make(chan struct{}),
		onceClose: func() {
			close(n.exit)
		},
	}

	svrConfList := []nacosconst.ServerConfig{}
	for _, nacosAddr := range n.NacosAddrs {
		split := strings.Split(nacosAddr, ":")
		port, err := strconv.ParseUint(split[1], 10, 64)
		if err != nil {
			continue
		}
		svrconf := nacosconst.ServerConfig{
			IpAddr: split[0],
			Port:   port,
		}
		svrConfList = append(svrConfList, svrconf)
	}
	client, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": svrConfList,
		"clientConfig": nacosconst.ClientConfig{
			TimeoutMs:           uint64(timeout / time.Millisecond),
			ListenInterval:      20000,
			NotLoadCacheAtStart: true,
			LogDir:              "logs/nacos/log",
		},
	})
	n.Client = &client
	if err != nil {
		return nil, perrors.WithMessagef(err, "nacos clients.CreateConfigClient(nacosAddrs:%+v)", nacosAddrs)
	}

	return n, nil
}

func (n *NacosClient) Done() <-chan struct{} {
	return n.exit
}

func (n *NacosClient) stop() bool {
	select {
	case <-n.exit:
		return true
	default:
		n.once.Do(n.onceClose)
	}

	return false
}

func (n *NacosClient) NacosClientValid() bool {
	select {
	case <-n.exit:
		return false
	default:
	}

	valid := true
	n.Lock()
	if n.Client == nil {
		valid = false
	}
	n.Unlock()

	return valid
}

func (n *NacosClient) Close() {
	if n == nil {
		return
	}

	n.stop()
	n.Lock()
	n.Client = nil
	n.Unlock()
	logger.Warnf("nacosClient{name:%s, nacos addr:%s} exit now.", n.name, n.NacosAddrs)
}
