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
	"strings"
)

import (
	log "github.com/dubbogo/gost/log/logger"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/rawbytes"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant/file"
	"dubbo.apache.org/dubbo-go/v3/config/parsers/properties"
)

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
		log.Errorf("resolvePlaceholder error %s", err)
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
