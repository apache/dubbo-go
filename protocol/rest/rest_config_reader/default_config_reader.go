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

package rest_config_reader

import (
	"io/ioutil"
	"os"
	"path"
)

import (
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
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

func (dcr *DefaultConfigReader) ReadConsumerConfig() *rest_interface.RestConsumerConfig {
	confConFile := os.Getenv(constant.CONF_CONSUMER_FILE_PATH)
	if confConFile == "" {
		logger.Warnf("rest consumer configure(consumer) file name is nil")
		return nil
	}
	if path.Ext(confConFile) != ".yml" {
		logger.Warnf("rest consumer configure file name{%v} suffix must be .yml", confConFile)
		return nil
	}
	confFileStream, err := ioutil.ReadFile(confConFile)
	if err != nil {
		logger.Warnf("ioutil.ReadFile(file:%s) = error:%v", confConFile, perrors.WithStack(err))
		return nil
	}
	restConsumerConfig := &rest_interface.RestConsumerConfig{}
	err = yaml.Unmarshal(confFileStream, restConsumerConfig)
	if err != nil {
		logger.Warnf("yaml.Unmarshal() = error:%v", perrors.WithStack(err))
		return nil
	}
	return restConsumerConfig
}

func (dcr *DefaultConfigReader) ReadProviderConfig() *rest_interface.RestProviderConfig {
	confProFile := os.Getenv(constant.CONF_PROVIDER_FILE_PATH)
	if len(confProFile) == 0 {
		logger.Warnf("rest provider configure(provider) file name is nil")
		return nil
	}

	if path.Ext(confProFile) != ".yml" {
		logger.Warnf("rest provider configure file name{%v} suffix must be .yml", confProFile)
		return nil
	}
	confFileStream, err := ioutil.ReadFile(confProFile)
	if err != nil {
		logger.Warnf("ioutil.ReadFile(file:%s) = error:%v", confProFile, perrors.WithStack(err))
		return nil
	}
	restProviderConfig := &rest_interface.RestProviderConfig{}
	err = yaml.Unmarshal(confFileStream, restProviderConfig)
	if err != nil {
		logger.Warnf("yaml.Unmarshal() = error:%v", perrors.WithStack(err))
		return nil
	}

	return restProviderConfig
}

func GetDefaultConfigReader() rest_interface.RestConfigReader {
	if defaultConfigReader == nil {
		defaultConfigReader = NewDefaultConfigReader()
	}
	return defaultConfigReader
}
