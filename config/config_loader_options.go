package config

import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"log"
)

type LoaderInitOption interface {
	init()
	apply()
}

type optionFunc struct {
	initFunc  func()
	applyFunc func()
}

func (f *optionFunc) init() {
	f.initFunc()
}

func (f *optionFunc) apply() {
	f.applyFunc()
}

func ConsumerInitOption(confConFile string) LoaderInitOption {
	return &optionFunc{
		func() {
			if errCon := ConsumerInit(confConFile); errCon != nil {
				log.Printf("[consumerInit] %#v", errCon)
				consumerConfig = nil
			} else if confBaseFile == "" {
				// Even though baseConfig has been initialized, we override it
				// because we think read from config file is correct config
				baseConfig = &consumerConfig.BaseConfig
			}
		},
		func() {
			loadConsumerConfig()
		},
	}
}

func ProviderInitOption(confProFile string) LoaderInitOption {
	return &optionFunc{
		func() {
			if errPro := ProviderInit(confProFile); errPro != nil {
				log.Printf("[providerInit] %#v", errPro)
				providerConfig = nil
			} else if confBaseFile == "" {
				// Even though baseConfig has been initialized, we override it
				// because we think read from config file is correct config
				baseConfig = &providerConfig.BaseConfig
			}
		},
		func() {
			loadProviderConfig()
		},
	}
}

func RouterInitOption(crf string) LoaderInitOption {
	return &optionFunc{
		func() {
			confRouterFile = crf
		},
		func() {
			initRouter()
		},
	}
}

func BaseInitOption(cbf string) LoaderInitOption {
	return &optionFunc{
		func() {
			if cbf == "" {
				return
			}
			confBaseFile = cbf
			if err := BaseInit(cbf); err != nil {
				log.Printf("[BaseInit] %#v", err)
				baseConfig = nil
			}
		},
		func() {
			// init the global event dispatcher
			extension.SetAndInitGlobalDispatcher(GetBaseConfig().EventDispatcherType)

			// start the metadata report if config set
			if err := startMetadataReport(GetApplicationConfig().MetadataType, GetBaseConfig().MetadataReportConfig); err != nil {
				logger.Errorf("Provider starts metadata report error, and the error is {%#v}", err)
				return
			}
		},
	}
}
