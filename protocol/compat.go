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

package protocol

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// Protocol is an alias retained for compatibility with old generated code.
//
// Deprecated: use base.Protocol instead.
type Protocol = base.Protocol

// Invoker is an alias retained for compatibility with old generated code.
//
// Deprecated: use base.Invoker instead.
type Invoker = base.Invoker

// Exporter is an alias retained for compatibility with old generated code.
//
// Deprecated: use base.Exporter instead.
type Exporter = base.Exporter

// Invocation is an alias retained for compatibility with old generated code.
//
// Deprecated: use base.Invocation instead.
type Invocation = base.Invocation

// Result is an alias retained for compatibility with old generated code.
//
// Deprecated: use result.Result instead.
type Result = result.Result

// RPCResult is an alias retained for compatibility with old generated code.
//
// Deprecated: use result.RPCResult instead.
type RPCResult = result.RPCResult

// RPCStatue is an alias retained for compatibility with old generated code.
//
// Deprecated: use base.RPCStatus instead.
type RPCStatue = base.RPCStatus
