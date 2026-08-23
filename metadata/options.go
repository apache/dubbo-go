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
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
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

// Options holds the configuration for the metadata service.
type Options struct {
	// appName is the application name.
	appName string
	// metadataType is the metadata storage type (local or remote), default value is local.
	metadataType string
	// port is the metadata service listen port, 0 means a random port is used.
	port int
	// protocol is the protocol used to export the MetadataService, default value is dubbo.
	protocol string
}

// defaultOptions returns a default Options instance.
func defaultOptions() *Options {
	return &Options{metadataType: constant.DefaultMetadataStorageType, protocol: constant.DefaultProtocol}
}

// NewOptions returns an Options instance from given options.
func NewOptions(opts ...Option) *Options {
	metaOptions := defaultOptions()
	for _, opt := range opts {
		opt(metaOptions)
	}
	return metaOptions
}

// Init registers opts as the global metadata options and, for local storage,
// exports the metadata service only once.
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
					logger.Errorf("[Metadata] metadata export failed, please check if dubbo protocol is imported, err=%v", err)
				}
			}()
			err = exporter.Export()
		}
	})
	return err
}

// Option configures an Options instance.
type Option func(*Options)

// WithAppName sets the application owning the metadata service.
func WithAppName(app string) Option {
	return func(options *Options) {
		options.appName = app
	}
}

// WithMetadataType sets the metadata storage type,
// allowed values are "local" and "remote",
// any other value behaves as "local".
func WithMetadataType(typ string) Option {
	return func(options *Options) {
		options.metadataType = typ
	}
}

// WithPort sets the metadata service listen port.
func WithPort(port int) Option {
	return func(options *Options) {
		options.port = port
	}
}

// WithMetadataProtocol sets the protocol used to export the MetadataService,
// allowed values are "dubbo" and "tri",
// any other value behaves as "tri".
func WithMetadataProtocol(protocol string) Option {
	return func(options *Options) {
		options.protocol = protocol
	}
}

// ReportOptions holds the configuration for a metadata report center connection.
type ReportOptions struct {
	// registryId is used as a key to look up the report instance.
	registryId string
	// MetadataReportConfig embeds the connection configuration.
	*global.MetadataReportConfig
}

// InitRegistryMetadataReport initializes a metadata report for each registry
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
			logger.Errorf("[Metadata] parse registry timeout config, err=%v", rc.Timeout)
		} else {
			WithTimeout(timeout)(opts)
		}
	}
	return opts
}

// Init builds a report URL from opts and registers the metadata report under opts.registryId.
func (opts *ReportOptions) Init() error {
	url, err := opts.toUrl()
	if err != nil {
		logger.Errorf("[Metadata] metadata report create error, err=%v", err)
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
		return nil, errors.New("Invalid MetadataReport Config.")
	}
	res.SetParam("metadata", res.Protocol)
	for key, val := range opts.Params {
		res.SetParam(key, val)
	}
	return res, nil
}

// defaultReportOptions returns a default ReportOptions instance.
func defaultReportOptions() *ReportOptions {
	return &ReportOptions{MetadataReportConfig: global.DefaultMetadataReportConfig()}
}

// NewReportOptions returns a ReportOptions instance from given options.
func NewReportOptions(opts ...ReportOption) *ReportOptions {
	reportOptions := defaultReportOptions()
	for _, opt := range opts {
		opt(reportOptions)
	}
	return reportOptions
}

// ReportOption configures a ReportOptions instance.
type ReportOption func(*ReportOptions)

// WithZookeeper sets the metadata report protocol to zookeeper.
func WithZookeeper() ReportOption {
	return func(opts *ReportOptions) {
		opts.Protocol = constant.ZookeeperKey
	}
}

// WithNacos sets the metadata report protocol to nacos.
func WithNacos() ReportOption {
	return func(opts *ReportOptions) {
		opts.Protocol = constant.NacosKey
	}
}

// WithEtcdV3 sets the metadata report protocol to etcd v3.
func WithEtcdV3() ReportOption {
	return func(opts *ReportOptions) {
		opts.Protocol = constant.EtcdV3Key
	}
}

// WithProtocol sets the metadata report protocol to a custom value.
// For the built-in backends, use WithZookeeper, WithNacos, or WithEtcdV3.
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

// WithUsername sets the metadata report username. Consumed only by nacos.
func WithUsername(username string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Username = username
	}
}

// WithPassword sets the metadata report password. Consumed only by nacos.
func WithPassword(password string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Password = password
	}
}

// WithTimeout sets the metadata report timeout.
func WithTimeout(timeout time.Duration) ReportOption {
	return func(opts *ReportOptions) {
		opts.Timeout = strconv.Itoa(int(timeout.Milliseconds()))
	}
}

// WithGroup sets the isolation group.
func WithGroup(group string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Group = group
	}
}

// WithNamespace sets the metadata report namespace. Consumed only by nacos.
func WithNamespace(namespace string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Namespace = namespace
	}
}

// WithParams sets extra params passed through to the backend client library.
func WithParams(params map[string]string) ReportOption {
	return func(opts *ReportOptions) {
		opts.Params = params
	}
}

// WithRegistryId sets the registry id this report originates from.
func WithRegistryId(id string) ReportOption {
	return func(opts *ReportOptions) {
		opts.registryId = id
	}
}
