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

package trace

import (
	"go.opentelemetry.io/otel/attribute"
)

var (
	RPCNameKey             = attribute.Key("name")
	RPCMessageTypeKey      = attribute.Key("message.type")
	RPCMessageIDKey        = attribute.Key("message.id")
	RPCNameMessage         = RPCNameKey.String("message")
	RPCMessageTypeSent     = RPCMessageTypeKey.String("SENT")
	RPCMessageTypeReceived = RPCMessageTypeKey.String("RECEIVED")
)

// Dubbo-specific span attributes.
//
// These describe information that has no OpenTelemetry semantic-convention
// equivalent, so they live under a stable "dubbo.*" namespace. Attributes that
// DO have a semantic-convention equivalent (rpc.system, rpc.service, rpc.method,
// server.address, ...) use the semconv helpers directly instead of these keys.
// Treat these key names as a stable contract: do not rename or remove them once
// released, since users build dashboards and alerts on top of them.
var (
	// DubboSideKey records the invocation side: "consumer" or "provider".
	DubboSideKey = attribute.Key("dubbo.side")
	// DubboProtocolKey records the Dubbo protocol in use, e.g. "dubbo", "tri".
	DubboProtocolKey = attribute.Key("dubbo.protocol")
	// DubboGroupKey records the Dubbo service group.
	DubboGroupKey = attribute.Key("dubbo.group")
	// DubboVersionKey records the Dubbo service version.
	DubboVersionKey = attribute.Key("dubbo.version")
)

const (
	// sideConsumer / sideProvider are the values used for DubboSideKey and the
	// span-name prefix ("dubbo.consumer" / "dubbo.provider").
	sideConsumer = "consumer"
	sideProvider = "provider"
)
