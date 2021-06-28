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
	"dubbo.apache.org/dubbo-go/v3/filter"
)

var generics = make(map[string]filter.GenericProcessor)

// SetGenericProcessor registers a GenericProcessor with the given name.
func SetGenericProcessor(name string, processor filter.GenericProcessor) {
	generics[name] = processor
}

// GetGenericProcessor finds a GenericProcessor by name.
func GetGenericProcessor(name string) filter.GenericProcessor {
	return generics[name]
}

// GetGenericProcessors returns all registered instances of GenericProcessor.
func GetGenericProcessors() []filter.GenericProcessor {
	ret := make([]filter.GenericProcessor, 0, len(generics))
	for _, v := range generics {
		ret = append(ret, v)
	}
	return ret
}
