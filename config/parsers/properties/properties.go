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

package properties

import (
	"strings"
)

import (
	"github.com/magiconair/properties"
)

type Properties struct{}

func Parser() *Properties {
	return &Properties{}
}

// Unmarshal parses the given properties bytes.
func (p *Properties) Unmarshal(b []byte) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	if load, err := properties.Load(b, properties.UTF8); err != nil {
		return nil, err
	} else {
		// see viper#unmarshalReader
		for _, key := range load.Keys() {
			value, _ := load.Get(key)
			// recursively build nested maps
			path := strings.Split(key, ".")
			lastKey := strings.ToLower(path[len(path)-1])
			deepestMap := deepSearch(out, path[0:len(path)-1])
			// set innermost value
			deepestMap[lastKey] = value
		}
		return out, nil
	}
}

// Marshal marshals the given config map to YAML bytes.
func (p *Properties) Marshal(o map[string]interface{}) ([]byte, error) {
	return nil, nil
}

// deepSearch scans deep maps, following the key indexes listed in the
// sequence "path".
// The last value is expected to be another map, and is returned.
//
// In case intermediate keys do not exist, or map to a non-map value,
// a new map is created and inserted, and the search continues from there:
// the initial map "m" may be modified!
func deepSearch(m map[string]interface{}, path []string) map[string]interface{} {
	for _, k := range path {
		m2, ok := m[k]
		if !ok {
			// intermediate key does not exist
			// => create it and continue from there
			m3 := make(map[string]interface{})
			m[k] = m3
			m = m3
			continue
		}
		m3, ok := m2.(map[string]interface{})
		if !ok {
			// intermediate key is a value
			// => replace with a new map
			m3 = make(map[string]interface{})
			m[k] = m3
		}
		// continue search from here
		m = m3
	}
	return m
}
