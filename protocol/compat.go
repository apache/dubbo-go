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
)

// Protocol Deprecated： base.Protocol type alias, just for compatible with old generate pb.go file
type Protocol = base.Protocol

// Invoker Deprecated： base.Invoker type alias, just for compatible with old generate pb.go file
type Invoker = base.Invoker

// Exporter Deprecated： base.Exporter type alias, just for compatible with old generate pb.go file
type Exporter = base.Exporter

// Invocation Deprecated： base.Invocation type alias, just for compatible with old generate pb.go file
type Invocation = base.Invocation

// Result Deprecated： base.Result type alias, just for compatible with old generate pb.go file
type Result = base.Result

// RPCResult Deprecated： base.RPCResult type alias, just for compatible with old generate pb.go file
type RPCResult = base.RPCResult

// RPCStatue Deprecated： base.RPCStatue type alias, just for compatible with old generate pb.go file
type RPCStatue = base.RPCStatus
