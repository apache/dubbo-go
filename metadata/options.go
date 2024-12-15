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

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
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
	protocol     string
}

func defaultOptions() *Options {
	return &Options{metadataType: constant.DefaultMetadataStorageType, protocol: constant.DefaultProtocol}
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
			exporter := &serviceExporter{service: metadataService, opts: opts}
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

func WithMetadataProtocol(protocol string) Option {
	return func(options *Options) {
		options.protocol = protocol
	}
}

type ReportOptions struct {
	registryId string
	*global.MetadataReportConfig
}

func InitRegistryMetadataReport(registries map[string]*global.RegistryConfig) error {
	if len(registries) > 0 {
		for id, reg := range registries {
			ok, err := reg.UseAsMetadataReport()
			if err != nil {
				return err
			}
			if ok {
				opts := fromRegistry(id, reg)
				if err := opts.Init(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func fromRegistry(id string, rc *global.RegistryConfig) *ReportOptions {
	opts := NewReportOptions(
		WithRegistryId(id),
		WithProtocol(rc.Protocol),
		WithAddress(rc.Address),
		WithUsername(rc.Username),
		WithPassword(rc.Password),
		WithGroup(rc.Group),
		WithNamespace(rc.Namespace),
		WithParams(rc.Params),
	)
	if rc.Timeout != "" {
		timeout, err := time.ParseDuration(rc.Timeout)
		if err != nil {
			logger.Errorf("parse registry timeout config error %v", rc.Timeout)
		} else {
			WithTimeout(timeout)(opts)
		}
	}
	return opts
}

func (opts *ReportOptions) Init() error {
	url, err := opts.toUrl()
	if err != nil {
		logger.Errorf("metadata report create error %v", err)
		return err
	}
	return addMetadataReport(opts.registryId, url)
}

func (opts *ReportOptions) toUrl() (*common.URL, error) {
	res, err := common.NewURL(opts.Address,
		common.WithUsername(opts.Username),
		common.WithPassword(opts.Password),
		common.WithLocation(opts.Address),
		common.WithProtocol(opts.Protocol),
		common.WithParamsValue(constant.TimeoutKey, opts.Timeout),
		common.WithParamsValue(constant.MetadataReportGroupKey, opts.Group),
		common.WithParamsValue(constant.MetadataReportNamespaceKey, opts.Namespace),
		common.WithParamsValue(constant.ClientNameKey, strings.Join([]string{constant.MetadataReportPrefix, opts.Protocol, opts.Address}, "-")),
	)
	if err != nil || len(res.Protocol) == 0 {
		return nil, perrors.New("Invalid MetadataReport Config.")
	}
	res.SetParam("metadata", res.Protocol)
	for key, val := range opts.Params {
		res.SetParam(key, val)
	}
	return res, nil
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

// WithAddress address metadata report will to use, if a URL schema is set,this will also set the protocol,
// such as WithAddress("zookeeper://127.0.0.1") will set address to "127.0.0.1" and protocol to "zookeeper"
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
