package registry

import (
	perrors "github.com/pkg/errors"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/config"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/remoting"
)

type BaseConfigurationListener struct {
	configurators           []config_center.Configurator
	dynamicConfiguration    config_center.DynamicConfiguration
	defaultConfiguratorFunc func(url *common.URL) config_center.Configurator
}

func (bcl *BaseConfigurationListener) Configurators() []config_center.Configurator {
	return bcl.configurators
}
func (bcl *BaseConfigurationListener) InitWith(key string, listener config_center.ConfigurationListener, f func(url *common.URL) config_center.Configurator) error {
	bcl.dynamicConfiguration = config.GetEnvInstance().GetDynamicConfiguration()
	bcl.defaultConfiguratorFunc = f
	bcl.dynamicConfiguration.AddListener(key, listener)
	if rawConfig, err := bcl.dynamicConfiguration.GetConfig(key, config_center.WithGroup(constant.DUBBO)); err != nil {
		return err
	} else if len(rawConfig) > 0 {
		bcl.genConfiguratorFromRawRule(rawConfig)
	}
	return nil
}

func (bcl *BaseConfigurationListener) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("Notification of overriding rule, change type is: %v , raw config content is:%v", event.ConfigType, event.Value)
	if event.ConfigType == remoting.EventTypeDel {
		bcl.configurators = nil
	} else {
		if err := bcl.genConfiguratorFromRawRule(event.Value.(string)); err != nil {
			logger.Error(perrors.WithStack(err))
		}
	}
}

func (bcl *BaseConfigurationListener) genConfiguratorFromRawRule(rawConfig string) error {
	urls, err := bcl.dynamicConfiguration.Parser().ParseToUrls(rawConfig)
	if err != nil {
		return perrors.WithMessage(err, "Failed to parse raw dynamic config and it will not take effect, the raw config is: "+
			rawConfig)
	}
	bcl.configurators = ToConfigurators(urls, bcl.defaultConfiguratorFunc)
	return nil
}
func (bcl *BaseConfigurationListener) OverrideUrl(url *common.URL) {
	for _, v := range bcl.configurators {
		v.Configure(url)
	}
}

func ToConfigurators(urls []*common.URL, f func(url *common.URL) config_center.Configurator) []config_center.Configurator {
	if len(urls) == 0 {
		return nil
	}
	var configurators []config_center.Configurator
	for _, url := range urls {
		if url.Protocol == constant.EMPTY_PROTOCOL {
			configurators = []config_center.Configurator{}
			break
		}
		//TODO:anyhost_key judage
		configurators = append(configurators, f(url))
	}
	return configurators
}
