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

// Package filter provides Filter definition and implementations for RPC call interception.
package filter

import (
	"context"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// Filter is the interface which wraps Invoke and OnResponse method and defines the functions of a filter.
// Invoke method is the core function of a filter, it determines the process of the filter.
// OnResponse method updates the results from Invoke and then returns the modified results.
type Filter interface {
	Invoke(context.Context, base.Invoker, base.Invocation) result.Result
	OnResponse(context.Context, result.Result, base.Invoker, base.Invocation) result.Result
}
