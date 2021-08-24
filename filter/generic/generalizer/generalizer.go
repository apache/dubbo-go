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

package generalizer

import (
	"reflect"
)

type Generalizer interface {

	// Generalize generalizes the object to a general struct.
	// For example:
	// map, the type of the `obj` allows a basic type, e.g. string, and a complicated type which is a POJO, see also
	// `hessian.POJO` at [apache/dubbo-go-hessian2](github.com/apache/dubbo-go-hessian2).
	Generalize(obj interface{}) (interface{}, error)

	// Realize realizes a general struct, described in `obj`, to an object for Golang.
	Realize(obj interface{}, typ reflect.Type) (interface{}, error)

	// GetType returns the type of the `obj`
	GetType(obj interface{}) (string, error)
}
