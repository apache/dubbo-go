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
	"log"
)

// MetaData meta data from registration center
type MetaData interface {
	ShowChildren() (map[string][]string, error)
}

// Factory metaData factory function
type Factory func(name string, zkAddrs []string) MetaData

var factMap = make(map[string]Factory)

// Register a factory into factMap
func Register(name string, fact Factory) {
	if _, ok := factMap[name]; ok {
		log.Printf("duplicate name: %s", name)
	}
	factMap[name] = fact
}

// GetFactory get a factory function from factMap
func GetFactory(name string) (fact Factory, ok bool) {
	fact, ok = factMap[name]
	return
}
