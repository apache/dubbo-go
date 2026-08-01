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
	"maps"
	"path"
	"strings"
)

type OpenAPIConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled" default:"false"`

	Path string `yaml:"path" json:"path,omitempty"`

	InfoTitle       string `yaml:"info-title" json:"infoTitle,omitempty"`
	InfoDescription string `yaml:"info-description" json:"infoDescription,omitempty"`
	InfoVersion     string `yaml:"info-version" json:"infoVersion,omitempty"`

	DefaultConsumesMediaTypes []string `yaml:"default-consumes-media-types" json:"defaultConsumesMediaTypes,omitempty"`
	DefaultProducesMediaTypes []string `yaml:"default-produces-media-types" json:"defaultProducesMediaTypes,omitempty"`
	DefaultHttpStatusCodes    []string `yaml:"default-http-status-codes" json:"defaultHttpStatusCodes,omitempty"`

	Settings map[string]string `yaml:"settings" json:"settings,omitempty"`
}

func DefaultOpenAPIConfig() *OpenAPIConfig {
	return &OpenAPIConfig{
		Enabled:                   false,
		Path:                      "/dubbo/openapi",
		InfoTitle:                 "Dubbo-go OpenAPI",
		InfoDescription:           "Dubbo-go OpenAPI",
		InfoVersion:               "1.0.0",
		DefaultConsumesMediaTypes: []string{"application/json"},
		DefaultProducesMediaTypes: []string{"application/json"},
		DefaultHttpStatusCodes:    []string{"200", "400", "500"},
		Settings:                  make(map[string]string),
	}
}

func (c *OpenAPIConfig) Clone() *OpenAPIConfig {
	if c == nil {
		return nil
	}

	settings := make(map[string]string)
	maps.Copy(settings, c.Settings)

	return &OpenAPIConfig{
		Enabled:                   c.Enabled,
		Path:                      c.Path,
		InfoTitle:                 c.InfoTitle,
		InfoDescription:           c.InfoDescription,
		InfoVersion:               c.InfoVersion,
		DefaultConsumesMediaTypes: append([]string{}, c.DefaultConsumesMediaTypes...),
		DefaultProducesMediaTypes: append([]string{}, c.DefaultProducesMediaTypes...),
		DefaultHttpStatusCodes:    append([]string{}, c.DefaultHttpStatusCodes...),
		Settings:                  settings,
	}
}

func (c *OpenAPIConfig) GetSetting(key string) string {
	if c == nil || c.Settings == nil {
		return ""
	}
	return c.Settings[key]
}

// Init fills missing defaults and normalizes the config fields.
func (c *OpenAPIConfig) Init() {
	if c == nil {
		return
	}
	defaultConfig := DefaultOpenAPIConfig()
	if c.Path == "" {
		c.Path = defaultConfig.Path
	}
	if c.InfoTitle == "" {
		c.InfoTitle = defaultConfig.InfoTitle
	}
	if c.InfoDescription == "" {
		c.InfoDescription = defaultConfig.InfoDescription
	}
	if c.InfoVersion == "" {
		c.InfoVersion = defaultConfig.InfoVersion
	}
	if c.DefaultConsumesMediaTypes == nil {
		c.DefaultConsumesMediaTypes = append([]string{}, defaultConfig.DefaultConsumesMediaTypes...)
	}
	if c.DefaultProducesMediaTypes == nil {
		c.DefaultProducesMediaTypes = append([]string{}, defaultConfig.DefaultProducesMediaTypes...)
	}
	if c.DefaultHttpStatusCodes == nil {
		c.DefaultHttpStatusCodes = append([]string{}, defaultConfig.DefaultHttpStatusCodes...)
	}
	if c.Settings == nil {
		c.Settings = make(map[string]string, len(defaultConfig.Settings))
	}
	for k, v := range defaultConfig.Settings {
		if _, ok := c.Settings[k]; !ok {
			c.Settings[k] = v
		}
	}

	// Normalize Path:
	//  1. path.Clean collapses duplicate slashes and resolves ".." / "." segments
	//  2. Trim trailing slash (preserve "/" as root)
	if c.Path != "" && c.Path != "/" {
		c.Path = path.Clean(c.Path)
		c.Path = strings.TrimRight(c.Path, "/")
		if c.Path == "" {
			c.Path = "/"
		}
	}
}
