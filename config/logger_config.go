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
	"net/url"
)

import (
	"github.com/creasty/defaults"
	"go.uber.org/zap"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/common/yaml"
)

type LoggerConfig struct {
	Level             string                 `default:"debug" json:"level" yaml:"level" property:"level"`
	Development       bool                   `default:"false" json:"development" yaml:"development" property:"development"`
	DisableCaller     bool                   `default:"false" json:"disable_caller" yaml:"disable_caller" property:"disable_caller"`
	DisableStacktrace bool                   `default:"false" json:"disable_stacktrace" yaml:"disable_stacktrace" property:"disable_stacktrace"`
	Encoding          string                 `default:"console" json:"encoding" yaml:"encoding" property:"encoding"`
	EncoderConfig     EncoderConfig          `default:"" json:"encoder_config" yaml:"encoder_config" property:"encoder_config"`
	OutputPaths       []string               `default:"[\"stderr\"]" json:"output_paths" yaml:"output_paths" property:"output_paths"`
	ErrorOutputPaths  []string               `default:"[\"stderr\"]" json:"error_output_paths" yaml:"error_output_paths" property:"error_output_paths"`
	InitialFields     map[string]interface{} `default:"" json:"initial_fields" yaml:"initial_fields" property:"initial_fields"`
}

type EncoderConfig struct {
	MessageKey     string            `default:"message" json:"message_key" yaml:"message_key" property:"message_key"`
	LevelKey       string            `default:"level" json:"level_key" yaml:"level_key" property:"level_key"`
	TimeKey        string            `default:"time" json:"time_key" yaml:"time_key" property:"time_key"`
	NameKey        string            `default:"logger" json:"name_key" yaml:"name_key" property:"name_key"`
	CallerKey      string            `default:"caller" json:"caller_key" yaml:"caller_key" property:"caller_key"`
	StacktraceKey  string            `default:"stacktrace" json:"stacktrace_key" yaml:"stacktrace_key" property:"stacktrace_key"`
	EncodeLevel    string            `default:"capitalColor" json:"level_encoder" yaml:"level_encoder" property:"level_encoder"`
	EncodeTime     string            `default:"iso8601" json:"time_encoder" yaml:"time_encoder" property:"time_encoder"`
	EncodeDuration string            `default:"seconds" json:"duration_encoder" yaml:"duration_encoder" property:"duration_encoder"`
	EncodeCaller   string            `default:"short" json:"caller_encoder" yaml:"caller_encoder" property:"caller_encoder"`
	Params         map[string]string `yaml:"params" json:"params,omitempty"`
}

func initLoggerConfig(rc *RootConfig) error {
	logConfig := rc.Logger
	if logConfig == nil {
		logConfig = new(LoggerConfig)
	}
	err := logConfig.check()
	if err != nil {
		return err
	}
	rc.Logger = logConfig
	byte, err := yaml.MarshalYML(logConfig)
	if err != nil {
		return err
	}
	zapConfig := &zap.Config{}
	err = yaml.UnmarshalYML(byte, zapConfig)
	if err != nil {
		return err
	}
	logger.InitLogger(zapConfig)
	return nil
}

func (l *LoggerConfig) check() error {
	if err := defaults.Set(l); err != nil {
		return err
	}
	return verify(l)
}

func (l *LoggerConfig) getUrlMap() url.Values {
	urlMap := url.Values{}

	for key, val := range l.EncoderConfig.Params {
		urlMap.Set(key, val)
	}
	return urlMap
}
