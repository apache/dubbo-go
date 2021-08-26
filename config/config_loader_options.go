/*
* Licensed to the Apache Software Foundation (ASF) under one or more
* contributor license agreements.  See the NOTICE file distributed with
* this work for additional information regarding copyright ownership.
* The ASF licenses this file to You under the Apache License, Version 2.0
* (the "License"); you may not use this file except in compliance with
* the License.  You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

type loaderConf struct {
	// loaderConf file type default yaml
	genre string
	// loaderConf file path default ./conf
	path string
	// loaderConf file delim default .
	delim string
}

func NewLoaderConf(opts ...LoaderConfOption) *loaderConf {
	configFilePath := "../conf/dubbogo.yaml"
	if configFilePathFromEnv := os.Getenv(constant.CONFIG_FILE_ENV_KEY); configFilePathFromEnv != "" {
		configFilePath = configFilePathFromEnv
	}

	conf := &loaderConf{
		genre: "yaml",
		path:  configFilePath,
		delim: ".",
	}

	for _, opt := range opts {
		opt.apply(conf)
	}
	return conf
}

type LoaderConfOption interface {
	apply(vc *loaderConf)
}

type loaderConfigFunc func(*loaderConf)

func (fn loaderConfigFunc) apply(vc *loaderConf) {
	fn(vc)
}

// WithGenre set loaderConf Genre
func WithGenre(genre string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		g := strings.ToLower(genre)
		if err := checkGenre(g); err != nil {
			panic(err)
		}
		conf.genre = g
	})
}

// WithPath set loaderConf path
func WithPath(path string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.path = absolutePath(path)
	})
}

func WithDelim(delim string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.delim = delim
	})
}

// absolutePath get absolut path
func absolutePath(inPath string) string {

	if inPath == "$HOME" || strings.HasPrefix(inPath, "$HOME"+string(os.PathSeparator)) {
		inPath = userHomeDir() + inPath[5:]
	}

	if filepath.IsAbs(inPath) {
		return filepath.Clean(inPath)
	}

	p, err := filepath.Abs(inPath)
	if err == nil {
		return filepath.Clean(p)
	}

	return ""
}

//userHomeDir get gopath
func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

// checkGenre check Genre
func checkGenre(genre string) error {
	genres := []string{"json", "toml", "yaml", "yml"}
	sort.Strings(genres)
	idx := sort.SearchStrings(genres, genre)
	if genres[idx] != genre {
		return errors.New(fmt.Sprintf("no support %s", genre))
	}
	return nil
}

//
//import (
//	"dubbo.apache.org/dubbo-go/v3/loaderConf/base"
//	"dubbo.apache.org/dubbo-go/v3/loaderConf/consumer"
//	"dubbo.apache.org/dubbo-go/v3/loaderConf/provider"
//	"log"
//)
//
//import (
//	"dubbo.apache.org/dubbo-go/v3/common/extension"
//)
//
//type LoaderInitOption interface {
//	init()
//	apply()
//}
//
//type loaderConfigFunc struct {
//	initFunc  func()
//	applyFunc func()
//}
//
//func (f *loaderConfigFunc) init() {
//	f.initFunc()
//}
//
//func (f *loaderConfigFunc) apply() {
//	f.applyFunc()
//}
//
//func ConsumerInitOption(confConFile string) LoaderInitOption {
//	return consumerInitOption(confConFile, false)
//}
//
//func ConsumerMustInitOption(confConFile string) LoaderInitOption {
//	return consumerInitOption(confConFile, true)
//}
//
//func consumerInitOption(confConFile string, must bool) LoaderInitOption {
//	return &loaderConfigFunc{
//		func() {
//			if consumerConfig != nil && !must {
//				return
//			}
//			if errCon := consumer.ConsumerInit(confConFile); errCon != nil {
//				log.Printf("[consumerInit] %#v", errCon)
//				consumerConfig = nil
//			} else if confBaseFile == "" {
//				// Check if there are some important key fields missing,
//				// if so, we set a default value for it
//				setDefaultValue(consumerConfig)
//				// Even though baseConfig has been initialized, we override it
//				// because we think read from loaderConf file is correct loaderConf
//				baseConfig = &consumerConfig.BaseConfig
//			}
//		},
//		func() {
//			loadConsumerConfig()
//		},
//	}
//}
//
//func ProviderInitOption(confProFile string) LoaderInitOption {
//	return providerInitOption(confProFile, false)
//}
//
//func ProviderMustInitOption(confProFile string) LoaderInitOption {
//	return providerInitOption(confProFile, true)
//}
//
//func providerInitOption(confProFile string, must bool) LoaderInitOption {
//	return &loaderConfigFunc{
//		func() {
//			if providerConfig != nil && !must {
//				return
//			}
//			if errPro := provider.ProviderInit(confProFile); errPro != nil {
//				log.Printf("[providerInit] %#v", errPro)
//				providerConfig = nil
//			} else if confBaseFile == "" {
//				// Check if there are some important key fields missing,
//				// if so, we set a default value for it
//				setDefaultValue(providerConfig)
//				// Even though baseConfig has been initialized, we override it
//				// because we think read from loaderConf file is correct loaderConf
//				baseConfig = &providerConfig.BaseConfig
//			}
//		},
//		func() {
//			loadProviderConfig()
//		},
//	}
//}
//
//func RouterInitOption(crf string) LoaderInitOption {
//	return &loaderConfigFunc{
//		func() {
//			confRouterFile = crf
//		},
//		func() {
//			initRouter()
//		},
//	}
//}
//
//func BaseInitOption(cbf string) LoaderInitOption {
//	return &loaderConfigFunc{
//		func() {
//			if cbf == "" {
//				return
//			}
//			confBaseFile = cbf
//			if err := base.BaseInit(cbf); err != nil {
//				log.Printf("[BaseInit] %#v", err)
//				baseConfig = nil
//			}
//		},
//		func() {
//			// init the global event dispatcher
//			extension.SetAndInitGlobalDispatcher(GetBaseConfig().EventDispatcherType)
//		},
//	}
//}
