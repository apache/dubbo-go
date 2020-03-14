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

package reader_impl

import (
	"github.com/apache/dubbo-go/config/rest"
	"github.com/apache/dubbo-go/config/rest/config_reader"
	"os"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/yaml"
)

var (
	defaultConfigReader *DefaultConfigReader
)

func init() {
	extension.SetRestConfigReader(constant.DEFAULT_KEY, GetDefaultConfigReader)
}

type DefaultConfigReader struct {
}

func NewDefaultConfigReader() *DefaultConfigReader {
	return &DefaultConfigReader{}
}

func (dcr *DefaultConfigReader) ReadConsumerConfig() (*rest.RestConsumerConfig, error) {
	confConFile := os.Getenv(constant.CONF_CONSUMER_FILE_PATH)
	if len(confConFile) == 0 {
		return nil, nil
	}
	restConsumerConfig := &rest.RestConsumerConfig{}
	err := yaml.UnmarshalYMLConfig(confConFile, restConsumerConfig)
	if err != nil {
		return nil, perrors.Errorf("[Rest Config] unmarshal Consumer RestYmlConfig error %v", perrors.WithStack(err))
	}
	return restConsumerConfig, nil
}

func (dcr *DefaultConfigReader) ReadProviderConfig() (*rest.RestProviderConfig, error) {
	confProFile := os.Getenv(constant.CONF_PROVIDER_FILE_PATH)
	if len(confProFile) == 0 {
		return nil, nil
	}
	restProviderConfig := &rest.RestProviderConfig{}
	err := yaml.UnmarshalYMLConfig(confProFile, restProviderConfig)
	if err != nil {
		return nil, perrors.Errorf("[Rest Config] unmarshal Provider RestYmlConfig error %v", perrors.WithStack(err))

	}
	return restProviderConfig, nil
}

func GetDefaultConfigReader() config_reader.RestConfigReader {
	if defaultConfigReader == nil {
		defaultConfigReader = NewDefaultConfigReader()
	}
	return defaultConfigReader
}
