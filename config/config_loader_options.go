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
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/file"
)

type loaderConf struct {
	// loaderConf file extension default yaml
	suffix string

	// loaderConf file path default ./conf
	path string

	// loaderConf file delim default .
	delim string

	// config bytes
	bytes []byte

	// user provide rootConfig built by config api
	rc *RootConfig
}

func NewLoaderConf(opts ...LoaderConfOption) *loaderConf {
	configFilePath := "../conf/dubbogo.yaml"
	if configFilePathFromEnv := os.Getenv(constant.ConfigFileEnvKey); configFilePathFromEnv != "" {
		configFilePath = configFilePathFromEnv
	}
	suffix := strings.Split(configFilePath, ".")
	conf := &loaderConf{
		suffix: suffix[len(suffix)-1],
		path:   absolutePath(configFilePath),
		delim:  ".",
	}
	for _, opt := range opts {
		opt.apply(conf)
	}
	if conf.rc != nil {
		return conf
	}
	if len(conf.bytes) <= 0 {
		bytes, err := ioutil.ReadFile(conf.path)
		if err != nil {
			panic(err)
		}
		conf.bytes = bytes
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
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
		conf.bytes = bytes
		genre := strings.Split(path, ".")
		conf.suffix = genre[len(genre)-1]
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
