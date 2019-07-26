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
	"time"
)

import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/filter/impl"
	"github.com/apache/dubbo-go/protocol"
)

const (
	EXAMPLE_FALLBACK_FILTER = "example_fallback"
)

//This is an example filter that handles result from hystrix filter
//Define your filters and write your service downgrade strategy like this
//TODO: Maybe a base fallback filter can be offered
type ExampleFallbackFilter struct {
}

func GetExampleFallbackFilter() filter.Filter {
	return &ExampleFallbackFilter{}
}

//The name should be the same as in your config
//Put the filter in front of hystrix filter
func init() {
	extension.SetFilter(EXAMPLE_FALLBACK_FILTER, GetExampleFallbackFilter)
}

func (ff *ExampleFallbackFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return invoker.Invoke(invocation)

}
func (ff *ExampleFallbackFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if err := result.Error(); err != nil {
		hystrixError, ok := err.(*impl.HystrixFilterError)
		if ok {
			if hystrixError.FailByHystrix() {
				logger.Debugf("[Example fallback filter]%s get error caused by %s", invocation.MethodName(), hystrixError.Error())
				//Handle the error caused by Hystrix, including circuit breaking, concurrency limit and timeout
				//The detailed error source can be got through hystrixError.Error()
				//In this example we return a mock result under this circumstance
				res := User{
					"MockID",
					"MockName",
					55,
					time.Now(),
					Gender(MAN),
				}
				*(invocation.Reply().(*User)) = res
				result.SetResult(&res)
				result.SetError(nil)
			}
		}
	}
	//If the error is not caused by hystrix, the result doesn't come from hystrix filter or there's no error,
	//we just return it here.
	return result
}
