package zookeeper

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/registry"
)


func (r *ZkRegistry) RegisterProvider(regConf registry.ServiceConfigIf) error {
	var (
		ok   bool
		err  error
		conf registry.ProviderServiceConfig
	)

	if conf, ok = regConf.(registry.ProviderServiceConfig); !ok {
		return jerrors.Errorf("the tyep of @regConf{%v} is not ProviderServiceConfig", regConf)
	}

	// 检验服务是否已经注册过
	ok = false
	r.cltLock.Lock()
	// 注意此处与consumerZookeeperRegistry的差异，consumer用的是conf.Service，
	// 因为consumer要提供watch功能给selector使用, provider允许注册同一个service的多个group or version
	_, ok = r.services[conf.String()]
	r.cltLock.Unlock()
	if ok {
		return jerrors.Errorf("Service{%s} has been registered", conf.String())
	}

	err = r.register(conf)
	if err != nil {
		return jerrors.Annotatef(err, "register(conf:%+v)", conf)
	}

	r.cltLock.Lock()
	r.services[conf.String()] = conf
	r.cltLock.Unlock()

	log.Debug("(ZkProviderRegistry)Register(conf{%#v})", conf)

	return nil
}
