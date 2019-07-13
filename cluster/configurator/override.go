package configurator

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/dubbogo/gost/container"
	"strings"
)

func init() {
	extension.SetDefaultConfigurator(newConfigurator)
}
func newConfigurator(url *common.URL) cluster.Configurator {
	return &overrideConfigurator{configuratorUrl: url}
}

type overrideConfigurator struct {
	configuratorUrl *common.URL
}

func (c *overrideConfigurator) GetUrl() *common.URL {
	return c.configuratorUrl
}

func (c *overrideConfigurator) Configure(url *common.URL) {
	//remove configuratorUrl some param that can not be configured
	if c.configuratorUrl.GetParam(constant.ENABLED_KEY, "true") == "false" || len(c.configuratorUrl.Location) == 0 {
		return
	}

	//branch for version 2.7.x
	apiVersion := c.configuratorUrl.GetParam(constant.CONFIG_VERSION_KEY, "")
	if len(apiVersion) != 0 {
		currentSide := url.GetParam(constant.SIDE_KEY, "")
		configuratorSide := c.configuratorUrl.GetParam(constant.SIDE_KEY, "")
		if currentSide == configuratorSide && common.DubboRole[common.CONSUMER] == currentSide && c.configuratorUrl.Port == "0" {
			localIP, _ := utils.GetLocalIP()
			c.configureIfMatch(localIP, url)
		} else if currentSide == configuratorSide && common.DubboRole[common.PROVIDER] == currentSide && c.configuratorUrl.Port == url.Port {
			c.configureIfMatch(url.Ip, url)
		}
	} else {
		//branch for version 2.6.x and less
		c.configureDeprecated(url)
	}
}

//translate from java, compatible rules in java
func (c *overrideConfigurator) configureIfMatch(host string, url *common.URL) {
	if constant.ANYHOST_VALUE == c.configuratorUrl.Ip || host == c.configuratorUrl.Ip {
		providers := c.configuratorUrl.GetParam(constant.OVERRIDE_PROVIDERS_KEY, "")
		if len(providers) == 0 || strings.Index(providers, url.Location) >= 0 || strings.Index(providers, constant.ANYHOST_VALUE) >= 0 {
			configApp := c.configuratorUrl.GetParam(constant.APPLICATION_KEY, c.configuratorUrl.Username)
			currentApp := url.GetParam(constant.APPLICATION_KEY, url.Username)
			if len(configApp) == 0 || constant.ANY_VALUE == configApp || configApp == currentApp {
				conditionKeys := container.NewSet()
				conditionKeys.Add(constant.CATEGORY_KEY)
				conditionKeys.Add(constant.CHECK_KEY)
				conditionKeys.Add(constant.ENABLED_KEY)
				conditionKeys.Add(constant.GROUP_KEY)
				conditionKeys.Add(constant.VERSION_KEY)
				conditionKeys.Add(constant.APPLICATION_KEY)
				conditionKeys.Add(constant.SIDE_KEY)
				conditionKeys.Add(constant.CONFIG_VERSION_KEY)
				conditionKeys.Add(constant.COMPATIBLE_CONFIG_KEY)
				for k, _ := range c.configuratorUrl.Params {
					value := c.configuratorUrl.Params.Get(k)
					if strings.HasPrefix(k, "~") || k == constant.APPLICATION_KEY || k == constant.SIDE_KEY {
						conditionKeys.Add(k)
						if len(value) != 0 && value != constant.ANY_VALUE && value != c.configuratorUrl.Params.Get(strings.TrimPrefix(k, "~")) {
							return
						}
					}
				}
				c.configuratorUrl.RemoveParams(conditionKeys)
				url.SetParams(c.configuratorUrl.Params)
			}
		}
	}
}

func (c *overrideConfigurator) configureDeprecated(url *common.URL) {
	// If override url has port, means it is a provider address. We want to control a specific provider with this override url, it may take effect on the specific provider instance or on consumers holding this provider instance.
	if c.configuratorUrl.Port != "0" {
		if url.Port == c.configuratorUrl.Port {
			c.configureIfMatch(url.Ip, url)
		}
	} else {
		// override url don't have a port, means the ip override url specify is a consumer address or 0.0.0.0
		// 1.If it is a consumer ip address, the intention is to control a specific consumer instance, it must takes effect at the consumer side, any provider received this override url should ignore;
		// 2.If the ip is 0.0.0.0, this override url can be used on consumer, and also can be used on provider
		if url.GetParam(constant.SIDE_KEY, "") == common.DubboRole[common.CONSUMER] {
			localIP, _ := utils.GetLocalIP()
			c.configureIfMatch(localIP, url)
		} else {
			c.configureIfMatch(constant.ANYHOST_VALUE, url)
		}
	}
}
