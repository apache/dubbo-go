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
package main

import (
	"github.com/afex/hystrix-go/hystrix"
)
import (
	"github.com/apache/dubbo-go/filter/impl"
	"github.com/apache/dubbo-go/protocol"
)

const EXAMPLE_FALLBACK_NAME = "exampleFallback"

type ExampleHystrixFallback struct {
}

//Example for hystrix fallback function
//The 4 parameters:
//The error that caused the fallback;
//The invoker that the filter receives;
//The invocation that should be invoked;
//The copy of circuit breaker for this invocation, for getting its status
func (f *ExampleHystrixFallback) FallbackFunc(err error, invoker protocol.Invoker, invocation protocol.Invocation, cb hystrix.CircuitBreaker) protocol.Result {
	result := &protocol.RPCResult{}
	if cb.IsOpen() {
		result.SetError(nil)
		result.SetResult("Mock OK")
	} else {
		result.SetError(err)
	}
	return result
}

//Add the fallback function to the map
//The name MUST be the same as in your config file
func init() {
	impl.SetHystrixFallback(EXAMPLE_FALLBACK_NAME, &ExampleHystrixFallback{})
}
