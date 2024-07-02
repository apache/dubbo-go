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

package dubbo

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/rawbytes"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/constant/file"
	"dubbo.apache.org/dubbo-go/v3/config/parsers/properties"
)

var (
	defaultActive   = "default"
	instanceOptions = defaultInstanceOptions()
)

func Load(opts ...LoaderConfOption) error {
	// conf
	conf := NewLoaderConf(opts...)
	if conf.opts == nil {
		koan := GetConfigResolver(conf)
		koan = conf.MergeConfig(koan)
		if err := koan.UnmarshalWithConf(instanceOptions.Prefix(),
			instanceOptions, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
			return err
		}
	} else {
		instanceOptions = conf.opts
	}

	if err := instanceOptions.init(); err != nil {
		return err
	}

	instance := &Instance{insOpts: instanceOptions}
	return instance.start()
}

type loaderConf struct {
	suffix string           // loaderConf file extension default yaml
	path   string           // loaderConf file path default ./conf/dubbogo.yaml
	delim  string           // loaderConf file delim default .
	bytes  []byte           // config bytes
	opts   *InstanceOptions // user provide InstanceOptions built by WithXXX api
	name   string           // config file name
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
	if conf.opts != nil {
		return conf
	}
	if len(conf.bytes) <= 0 {
		if bytes, err := os.ReadFile(conf.path); err != nil {
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
// Deprecated: replaced by WithSuffix
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
		if bytes, err := os.ReadFile(conf.path); err != nil {
			panic(err)
		} else {
			conf.bytes = bytes
		}
		name, suffix := resolverFilePath(path)
		conf.suffix = suffix
		conf.name = name
	})
}

func WithInstanceOptions(opts *InstanceOptions) LoaderConfOption {
	return loaderConfigFunc(func(conf *loaderConf) {
		conf.opts = opts
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

// userHomeDir get gopath
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

// resolverFilePath resolver file path
// eg: give a ./conf/dubbogo.yaml return dubbogo and yaml
func resolverFilePath(path string) (name, suffix string) {
	paths := strings.Split(path, "/")
	fileName := strings.Split(paths[len(paths)-1], ".")
	if len(fileName) < 2 {
		return fileName[0], string(file.YAML)
	}
	return fileName[0], fileName[1]
}

// MergeConfig merge config file
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

// getLegalActive if active is null return default
func getLegalActive(active string) string {
	if len(active) == 0 {
		return defaultActive
	}
	return active
}

// GetConfigResolver get config resolver
func GetConfigResolver(conf *loaderConf) *koanf.Koanf {
	var (
		k   *koanf.Koanf
		err error
	)
	if len(conf.suffix) <= 0 {
		conf.suffix = string(file.YAML)
	}
	if len(conf.delim) <= 0 {
		conf.delim = "."
	}
	bytes := conf.bytes
	if len(bytes) <= 0 {
		panic(errors.New("bytes is nil,please set bytes or file path"))
	}
	k = koanf.New(conf.delim)

	switch conf.suffix {
	case "yaml", "yml":
		err = k.Load(rawbytes.Provider(bytes), yaml.Parser())
	case "json":
		err = k.Load(rawbytes.Provider(bytes), json.Parser())
	case "toml":
		err = k.Load(rawbytes.Provider(bytes), toml.Parser())
	case "properties":
		err = k.Load(rawbytes.Provider(bytes), properties.Parser())
	default:
		err = errors.Errorf("no support %s file suffix", conf.suffix)
	}

	if err != nil {
		panic(err)
	}
	return resolvePlaceholder(k)
}

// resolvePlaceholder replace ${xx} with real value
func resolvePlaceholder(resolver *koanf.Koanf) *koanf.Koanf {
	m := make(map[string]interface{})
	for k, v := range resolver.All() {
		s, ok := v.(string)
		if !ok {
			continue
		}
		newKey, defaultValue := checkPlaceholder(s)
		if newKey == "" {
			continue
		}
		m[k] = resolver.Get(newKey)
		if m[k] == nil {
			m[k] = defaultValue
		}
	}
	err := resolver.Load(confmap.Provider(m, resolver.Delim()), nil)
	if err != nil {
		logger.Errorf("resolvePlaceholder error %s", err)
	}
	return resolver
}

func checkPlaceholder(s string) (newKey, defaultValue string) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, file.PlaceholderPrefix) || !strings.HasSuffix(s, file.PlaceholderSuffix) {
		return
	}
	s = s[len(file.PlaceholderPrefix) : len(s)-len(file.PlaceholderSuffix)]
	indexColon := strings.Index(s, ":")
	if indexColon == -1 {
		newKey = strings.TrimSpace(s)
		return
	}
	newKey = strings.TrimSpace(s[0:indexColon])
	defaultValue = strings.TrimSpace(s[indexColon+1:])

	return
}
