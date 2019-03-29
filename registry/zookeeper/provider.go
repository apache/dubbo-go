package zookeeper

import (
	log "github.com/AlexStocks/log4go"
	"github.com/dubbo/dubbo-go/service"
	jerrors "github.com/juju/errors"
)

type ProviderServiceConfig struct {
	service.ServiceConfig
}


func (r *ZkRegistry) NewProviderServiceConfig(config service.ServiceConfig)service.ServiceConfigIf{
	return ProviderServiceConfig{
		config,
	}
}

func (r *ZkRegistry) ProviderRegister(c service.ServiceConfigIf) error {
	var (
		ok   bool
		err  error
		conf ProviderServiceConfig
	)

	if conf, ok = c.(ProviderServiceConfig); !ok {
		return jerrors.Errorf("@c{%v} type is not ServiceConfig", c)
	}

	// 检验服务是否已经注册过
	ok = false
	r.Lock()
	// 注意此处与consumerZookeeperRegistry的差异，consumer用的是conf.Service，
	// 因为consumer要提供watch功能给selector使用, provider允许注册同一个service的多个group or version
	_, ok = r.services[conf.String()]
	r.Unlock()
	if ok {
		return jerrors.Errorf("Service{%s} has been registered", conf.String())
	}

	err = r.register(conf)
	if err != nil {
		return jerrors.Annotatef(err, "register(conf:%+v)", conf)
	}

	r.Lock()
	r.services[conf.String()] = conf
	log.Debug("(ZkProviderRegistry)Register(conf{%#v})", conf)
	r.Unlock()

	return nil
}