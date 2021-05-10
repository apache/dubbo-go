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

package filter

import (
	"context"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// Filter interface defines the functions of a filter
// Extension - Filter
type Filter interface {
	// Invoke is the core function of a filter, it determines the process of the filter
	Invoke(context.Context, protocol.Invoker, protocol.Invocation) protocol.Result
	// OnResponse updates the results from Invoke and then returns the modified results.
	OnResponse(context.Context, protocol.Result, protocol.Invoker, protocol.Invocation) protocol.Result
}
