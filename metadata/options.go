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

package metadata

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
)

var (
	metadataOptions *Options
	exportOnce      sync.Once
)

type Options struct {
	appName      string
	metadataType string
	port         int
}

func defaultOptions() *Options {
	return &Options{metadataType: constant.DefaultMetadataStorageType}
}

func NewOptions(opts ...Option) *Options {
	metaOptions := defaultOptions()
	for _, opt := range opts {
		opt(metaOptions)
	}
	return metaOptions
}

func (opts *Options) Init() error {
	metadataOptions = opts
	var err error
	exportOnce.Do(func() {
		if opts.metadataType != constant.RemoteMetadataStorageType {
			exporter := &ServiceExporter{service: metadataService, opts: opts}
			defer func() {
				// TODO remove this recover func,this just to avoid some unit test failed,this will not happen in user side mostly
				// config test -> metadata exporter -> dubbo protocol/remoting -> config,cycle import will occur
				// some day we fix the cycle import then can remove this recover
				if err := recover(); err != nil {
					logger.Errorf("metadata export failed,please check if dubbo protocol is imported, error: %v", err)
				}
			}()
			err = exporter.Export()
		}
	})
	return err
}

type Option func(*Options)

func WithAppName(app string) Option {
	return func(options *Options) {
		options.appName = app
	}
}

func WithMetadataType(typ string) Option {
	return func(options *Options) {
		options.metadataType = typ
	}
}

func WithPort(port int) Option {
	return func(options *Options) {
		options.port = port
	}
}

type ReportOptions struct {
	registryId string
	*global.MetadataReportConfig
}

func (opts *ReportOptions) Init() error {
	fac := extension.GetMetadataReportFactory(opts.Protocol)
	if fac == nil {
		logger.Errorf("no metadata report factory of protocol %s found!", opts.Protocol)
		return nil
	}
	url, err := toUrl(opts)
	if err != nil {
		logger.Errorf("metadata report create error %v", err)
		return err
	}
	instances[opts.registryId] = &DelegateMetadataReport{instance: fac.CreateMetadataReport(url)}
	return nil
}

func defaultReportOptions() *ReportOptions {
	return &ReportOptions{MetadataReportConfig: global.DefaultMetadataReportConfig()}
}

func NewReportOptions(opts ...ReportOption) *ReportOptions {
	reportOptions := defaultReportOptions()
	for _, opt := range opts {
		opt(reportOptions)
	}
	return reportOptions
}

type ReportOption func(*ReportOptions)

func WithZookeeper() ReportOption {
	return func(opts *ReportOptions) {
		opts.Protocol = constant.ZookeeperKey
	}
}

func WithNacos() ReportOption {
	return func(opts *ReportOptions) {
		opts.Protocol = constant.NacosKey
	}
}

func WithEtcdV3() ReportOption {
	return func(opts *ReportOptions) {
		opts.Protocol = constant.EtcdV3Key
	}
}

func WithProtocol(meta string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Protocol = meta
	}
}

func WithAddress(address string) ReportOption {
	return func(opts *ReportOptions) {
		if i := strings.Index(address, "://"); i > 0 {
			opts.Protocol = address[0:i]
		}
		opts.Address = address
	}
}

func WithUsername(username string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Username = username
	}
}

func WithPassword(password string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Password = password
	}
}

func WithTimeout(timeout time.Duration) ReportOption {
	return func(opts *ReportOptions) {
		opts.Timeout = strconv.Itoa(int(timeout.Milliseconds()))
	}
}

func WithGroup(group string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Group = group
	}
}

func WithNamespace(namespace string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Namespace = namespace
	}
}

func WithParams(params map[string]string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Params = params
	}
}

func WithRegistryId(id string) ReportOption {
	return func(opts *ReportOptions) {
		opts.registryId = id
	}
}
