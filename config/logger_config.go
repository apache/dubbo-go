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
	"fmt"
	"strconv"
)

import (
	getty "github.com/apache/dubbo-getty"

	"github.com/creasty/defaults"

	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

type LoggerConfig struct {
	// logger driver default zap
	Driver string `default:"zap" yaml:"driver"`

	// logger level
	Level string `default:"info" yaml:"level"`

	// logger formatter default text
	Format string `default:"text" yaml:"format"`

	// supports simultaneous file and console eg: console,file default console
	Appender string `default:"console" yaml:"appender"`

	// logger file
	File *File `yaml:"file"`
}

type File struct {
	// log file name default dubbo.log
	Name string `default:"dubbo.log" yaml:"name"`

	// log max size default 100Mb
	MaxSize int `default:"100" yaml:"max-size"`

	// log max backups default 5
	MaxBackups int `default:"5" yaml:"max-backups"`

	// log file max age default 3 day
	MaxAge int `default:"3" yaml:"max-age"`

	Compress *bool `default:"true" yaml:"compress"`
}

// Prefix dubbo.logger
func (l *LoggerConfig) Prefix() string {
	return constant.LoggerConfigPrefix
}

func (l *LoggerConfig) Init() error {
	var (
		log logger.Logger
		err error
	)
	if err = l.check(); err != nil {
		return err
	}

	if log, err = extension.GetLogger(l.Driver, l.toURL()); err != nil {
		return err
	}
	// set log
	logger.SetLogger(log)
	getty.SetLogger(log)
	return nil
}

func (l *LoggerConfig) check() error {
	if err := defaults.Set(l); err != nil {
		return err
	}
	return verify(l)
}

func (l *LoggerConfig) toURL() *common.URL {
	address := fmt.Sprintf("%s://%s", l.Driver, l.Level)
	url, _ := common.NewURL(address,
		common.WithParamsValue(constant.LoggerLevelKey, l.Level),
		common.WithParamsValue(constant.LoggerDriverKey, l.Driver),
		common.WithParamsValue(constant.LoggerFormatKey, l.Format),
		common.WithParamsValue(constant.LoggerAppenderKey, l.Appender),
		common.WithParamsValue(constant.LoggerFileNameKey, l.File.Name),
		common.WithParamsValue(constant.LoggerFileNaxSizeKey, strconv.Itoa(l.File.MaxSize)),
		common.WithParamsValue(constant.LoggerFileMaxBackupsKey, strconv.Itoa(l.File.MaxBackups)),
		common.WithParamsValue(constant.LoggerFileMaxAgeKey, strconv.Itoa(l.File.MaxAge)),
		common.WithParamsValue(constant.LoggerFileCompressKey, strconv.FormatBool(*l.File.Compress)),
	)
	return url
}

// DynamicUpdateProperties dynamically update properties.
func (l *LoggerConfig) DynamicUpdateProperties(new *LoggerConfig) {

}

type LoggerConfigBuilder struct {
	loggerConfig *LoggerConfig
}

func NewLoggerConfigBuilder() *LoggerConfigBuilder {
	return &LoggerConfigBuilder{loggerConfig: &LoggerConfig{File: &File{}}}
}

func (lcb *LoggerConfigBuilder) SetDriver(driver string) *LoggerConfigBuilder {
	lcb.loggerConfig.Driver = driver
	return lcb
}

func (lcb *LoggerConfigBuilder) SetLevel(level string) *LoggerConfigBuilder {
	lcb.loggerConfig.Level = level
	return lcb
}

func (lcb *LoggerConfigBuilder) SetFormat(format string) *LoggerConfigBuilder {
	lcb.loggerConfig.Format = format
	return lcb
}

func (lcb *LoggerConfigBuilder) SetAppender(appender string) *LoggerConfigBuilder {
	lcb.loggerConfig.Appender = appender
	return lcb
}

func (lcb *LoggerConfigBuilder) SetFileName(name string) *LoggerConfigBuilder {
	lcb.loggerConfig.File.Name = name
	return lcb
}

func (lcb *LoggerConfigBuilder) SetFileMaxSize(maxSize int) *LoggerConfigBuilder {
	lcb.loggerConfig.File.MaxSize = maxSize
	return lcb
}

func (lcb *LoggerConfigBuilder) SetFileMaxBackups(maxBackups int) *LoggerConfigBuilder {
	lcb.loggerConfig.File.MaxBackups = maxBackups
	return lcb
}

func (lcb *LoggerConfigBuilder) SetFileMaxAge(maxAge int) *LoggerConfigBuilder {
	lcb.loggerConfig.File.MaxAge = maxAge
	return lcb
}

func (lcb *LoggerConfigBuilder) SetFileCompress(compress bool) *LoggerConfigBuilder {
	lcb.loggerConfig.File.Compress = &compress
	return lcb
}

// Build return config and set default value if nil
func (lcb *LoggerConfigBuilder) Build() *LoggerConfig {
	if err := defaults.Set(lcb.loggerConfig); err != nil {
		return nil
	}
	return lcb.loggerConfig
}
