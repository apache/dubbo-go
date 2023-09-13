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

package global

import (
	"github.com/creasty/defaults"
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

func DefaultLoggerConfig() *LoggerConfig {
	// this logic is same as /config/logger_config.go/LoggerConfigBuilder.Build
	cfg := &LoggerConfig{
		File: &File{},
	}
	defaults.MustSet(cfg)

	return cfg
}

type LoggerOption func(*LoggerConfig)

func WithLogger_Driver(driver string) LoggerOption {
	return func(cfg *LoggerConfig) {
		cfg.Driver = driver
	}
}

func WithLogger_Level(level string) LoggerOption {
	return func(cfg *LoggerConfig) {
		cfg.Level = level
	}
}

func WithLogger_Format(format string) LoggerOption {
	return func(cfg *LoggerConfig) {
		cfg.Format = format
	}
}

func WithLogger_Appender(appender string) LoggerOption {
	return func(cfg *LoggerConfig) {
		cfg.Appender = appender
	}
}

func WithLogger_File_Name(name string) LoggerOption {
	return func(cfg *LoggerConfig) {
		if cfg.File == nil {
			cfg.File = new(File)
		}
		cfg.File.Name = name
	}
}

func WithLogger_File_MaxSize(size int) LoggerOption {
	return func(cfg *LoggerConfig) {
		if cfg.File == nil {
			cfg.File = new(File)
		}
		cfg.File.MaxSize = size
	}
}

func WithLogger_File_MaxBackups(backups int) LoggerOption {
	return func(cfg *LoggerConfig) {
		if cfg.File == nil {
			cfg.File = new(File)
		}
		cfg.File.MaxBackups = backups
	}
}

func WithLogger_File_MaxAge(age int) LoggerOption {
	return func(cfg *LoggerConfig) {
		if cfg.File == nil {
			cfg.File = new(File)
		}
		cfg.File.MaxAge = age
	}
}

func WithLogger_File_Compress(flag bool) LoggerOption {
	return func(cfg *LoggerConfig) {
		if cfg.File == nil {
			cfg.File = new(File)
		}
		cfg.File.Compress = &flag
	}
}
