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
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config/parsers/properties"
)

// GetConfigResolver get config resolver
func GetConfigResolver(conf *loaderConf) *koanf.Koanf {
	var (
		k   *koanf.Koanf
		err error
	)
	k = koanf.New(conf.delim)

	switch conf.genre {
	case "yaml", "yml":
		err = k.Load(file.Provider(conf.path), yaml.Parser())
	case "json":
		err = k.Load(file.Provider(conf.path), json.Parser())
	case "toml":
		err = k.Load(file.Provider(conf.path), toml.Parser())
	case "properties":
		err = k.Load(file.Provider(conf.path), properties.Parser())
	default:
		err = errors.Errorf("no support %s file type", conf.genre)
	}

	if err != nil {
		panic(err)
	}
	return k
}
