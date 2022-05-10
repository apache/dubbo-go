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

	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

var (
	RPCNameKey             = attribute.Key("name")
	RPCMessageTypeKey      = attribute.Key("message.type")
	RPCMessageIDKey        = attribute.Key("message.id")
	RPCSystemDubbo         = semconv.RPCSystemKey.String("apache_dubbo")
	RPCNameMessage         = RPCNameKey.String("message")
	RPCMessageTypeSent     = RPCMessageTypeKey.String("SENT")
	RPCMessageTypeReceived = RPCMessageTypeKey.String("RECEIVED")
)
