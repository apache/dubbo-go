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

package proxy_factory

import (
	"fmt"
	"reflect"
)

import (
	perrors "github.com/pkg/errors"
)

// CallLocalMethod is used to handle invoke exception in user func.
func callLocalMethod(method reflect.Method, in []reflect.Value) ([]reflect.Value, error) {
	var (
		returnValues []reflect.Value
		retErr       error
	)

	func() {
		defer func() {
			if e := recover(); e != nil {
				if err, ok := e.(error); ok {
					retErr = err
				} else if err, ok := e.(string); ok {
					retErr = perrors.New(err)
				} else {
					retErr = fmt.Errorf("invoke function error, unknow exception: %+v", e)
				}
			}
		}()

		returnValues = method.Func.Call(in)
	}()

	if retErr != nil {
		return nil, retErr
	}

	return returnValues, retErr
}
