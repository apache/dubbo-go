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
	DisableCaller     bool                   `default:"false" json:"disableCaller" yaml:"disableCaller" property:"property"`
	DisableStacktrace bool                   `default:"false" json:"disableStacktrace" yaml:"disableStacktrace" property:"disableStacktrace"`
	Encoding          string                 `default:"console" json:"encoding" yaml:"encoding" property:"encoding"`
	EncoderConfig     EncoderConfig          `default:"" json:"encoderConfig" yaml:"encoderConfig" property:"encoderConfig"`
	OutputPaths       []string               `default:"[\"stderr\"]" json:"outputPaths" yaml:"outputPaths" property:"outputPaths"`
	ErrorOutputPaths  []string               `default:"[\"stderr\"]" json:"errorOutputPaths" yaml:"errorOutputPaths" property:"errorOutputPaths"`
	InitialFields     map[string]interface{} `default:"" json:"initialFields" yaml:"initialFields" property:"initialFields"`
}

type EncoderConfig struct {
	MessageKey     string `default:"message" json:"messageKey" yaml:"messageKey" property:"messageKey"`
	LevelKey       string `default:"level" json:"levelKey" yaml:"levelKey" property:"levelKey"`
	TimeKey        string `default:"time" json:"timeKey" yaml:"timeKey" property:"timeKey"`
	NameKey        string `default:"logger" json:"nameKey" yaml:"nameKey" property:"nameKey"`
	CallerKey      string `default:"caller" json:"callerKey" yaml:"callerKey" property:"callerKey"`
	StacktraceKey  string `default:"stacktrace" json:"stacktraceKey" yaml:"stacktraceKey" property:"stacktraceKey"`
	EncodeLevel    string `default:"capitalColor" json:"levelEncoder" yaml:"levelEncoder" property:"levelEncoder"`
	EncodeTime     string `default:"iso8601" json:"timeEncoder" yaml:"timeEncoder" property:"timeEncoder"`
	EncodeDuration string `default:"seconds" json:"durationEncoder" yaml:"durationEncoder" property:"durationEncoder"`
	EncodeCaller   string `default:"short" json:"callerEncoder" yaml:"callerEncoder" property:"callerEncoder"`
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
