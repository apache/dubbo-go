package config

import (
	"log"
)

type LoaderInitOption interface {
	apply()
}

type optionFunc func()

func (f optionFunc) apply() {
	f()
}

func ConsumerInitOption(confConFile string) LoaderInitOption {
	return optionFunc(func() {
		if errCon := ConsumerInit(confConFile); errCon != nil {
			log.Printf("[consumerInit] %#v", errCon)
			consumerConfig = nil
		} else {
			// Even though baseConfig has been initialized, we override it
			// because we think read from config file is correct config
			baseConfig = &consumerConfig.BaseConfig
		}
		loadConsumerConfig()
	})
}

func ProviderInitOption(confProFile string) LoaderInitOption {
	return optionFunc(func() {
		if errPro := ProviderInit(confProFile); errPro != nil {
			log.Printf("[providerInit] %#v", errPro)
			providerConfig = nil
		} else {
			// Even though baseConfig has been initialized, we override it
			// because we think read from config file is correct config
			baseConfig = &providerConfig.BaseConfig
		}
		loadProviderConfig()
	})
}

func RouterInitOption(crf string) LoaderInitOption {
	return optionFunc(func() {
		confRouterFile = crf
		initRouter()
	})
}
