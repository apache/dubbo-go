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

package extension

import (
	"dubbo.apache.org/dubbo-go/v3/config/interfaces"
)

var (
	configReaders = make(map[string]func() interfaces.ConfigReader)
	defaults      = make(map[string]string)
)

// SetConfigReaders sets a creator of config reader with @name
func SetConfigReaders(name string, v func() interfaces.ConfigReader) {
	configReaders[name] = v
}

// GetConfigReaders gets a config reader with @name
func GetConfigReaders(name string) interfaces.ConfigReader {
	if configReaders[name] == nil {
		panic("config reader for " + name + " is not existing, make sure you have imported the package.")
	}
	return configReaders[name]()
}

// SetDefaultConfigReader sets @name for @module in default config reader
func SetDefaultConfigReader(module, name string) {
	defaults[module] = name
}

// GetDefaultConfigReader gets default config reader
func GetDefaultConfigReader() map[string]string {
	return defaults
}
