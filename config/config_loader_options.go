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
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/knadh/koanf"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/constant/file"
)

type loaderConf struct {
	suffix string      // loaderConf file extension default yaml
	path   string      // loaderConf file path default ./conf/dubbogo.yaml
	delim  string      // loaderConf file delim default .
	bytes  []byte      // config bytes
	rc     *RootConfig // user provide rootConfig built by config api
	name   string      // config file name
}

func NewLoaderConf(opts ...LoaderConfOption) *loaderConf {
	configFilePath := "../conf/dubbogo.yaml"
	if configFilePathFromEnv := os.Getenv(constant.ConfigFileEnvKey); configFilePathFromEnv != "" {
		configFilePath = configFilePathFromEnv
	}
	name, suffix := resolverFilePath(configFilePath)
	conf := &loaderConf{
		suffix: suffix,
		path:   absolutePath(configFilePath),
		delim:  ".",
		name:   name,
	}
	for _, opt := range opts {
		opt.apply(conf)
	}
	if conf.rc != nil {
		return conf
	}
	if len(conf.bytes) <= 0 {
		if bytes, err := ioutil.ReadFile(conf.path); err != nil {
			panic(err)
		} else {
			conf.bytes = bytes
		}
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

// WithGenre set load config file suffix
//Deprecated: replaced by WithSuffix
func WithGenre(suffix string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		g := strings.ToLower(suffix)
		if err := checkFileSuffix(g); err != nil {
			panic(err)
		}
		conf.suffix = g
	})
}

// WithSuffix set load config file suffix
func WithSuffix(suffix file.Suffix) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.suffix = string(suffix)
	})
}

// WithPath set load config path
func WithPath(path string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.path = absolutePath(path)
		if bytes, err := ioutil.ReadFile(path); err != nil {
			panic(err)
		} else {
			conf.bytes = bytes
		}
		name, suffix := resolverFilePath(path)
		conf.suffix = suffix
		conf.name = name
	})
}

func WithRootConfig(rc *RootConfig) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.rc = rc
	})
}

func WithDelim(delim string) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.delim = delim
	})
}

// WithBytes set load config  bytes
func WithBytes(bytes []byte) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.bytes = bytes
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

// checkFileSuffix check file suffix
func checkFileSuffix(suffix string) error {
	for _, g := range []string{"json", "toml", "yaml", "yml", "properties"} {
		if g == suffix {
			return nil
		}
	}
	return errors.Errorf("no support file suffix: %s", suffix)
}

//resolverFilePath resolver file path
// eg: give a ./conf/dubbogo.yaml return dubbogo and yaml
func resolverFilePath(path string) (name, suffix string) {
	paths := strings.Split(path, "/")
	fileName := strings.Split(paths[len(paths)-1], ".")
	if len(fileName) < 2 {
		return fileName[0], string(file.YAML)
	}
	return fileName[0], fileName[1]
}

//MergeConfig merge config file
func (conf *loaderConf) MergeConfig(koan *koanf.Koanf) *koanf.Koanf {
	var (
		activeKoan *koanf.Koanf
		activeConf *loaderConf
	)
	active := koan.String("dubbo.profiles.active")
	active = getLegalActive(active)
	logger.Infof("The following profiles are active: %s", active)
	if defaultActive != active {
		path := conf.getActiveFilePath(active)
		if !pathExists(path) {
			logger.Debugf("Config file:%s not exist skip config merge", path)
			return koan
		}
		activeConf = NewLoaderConf(WithPath(path))
		activeKoan = GetConfigResolver(activeConf)
		if err := koan.Merge(activeKoan); err != nil {
			logger.Debugf("Config merge err %s", err)
		}
	}
	return koan
}

func (conf *loaderConf) getActiveFilePath(active string) string {
	suffix := constant.DotSeparator + conf.suffix
	return strings.ReplaceAll(conf.path, suffix, "") + "-" + active + suffix
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	} else {
		return !os.IsNotExist(err)
	}
}
