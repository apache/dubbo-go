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

package extension

// Scope identifies the lifecycle level at which an extension is initialized.
// The values are bit flags when declared by an extension, so one extension can
// support multiple lifecycle levels. A single initialization always receives
// one concrete scope value.
type Scope uint8

const (
	// InstanceScope is the lifecycle of a dubbo.Instance.
	InstanceScope Scope = 1 << iota
	// ClientScope is the lifecycle of a client/consumer.
	ClientScope
	// ServerScope is the lifecycle of a server/provider.
	ServerScope
)

func (s Scope) valid() bool {
	return s == InstanceScope || s == ClientScope || s == ServerScope
}

// Supports reports whether declared contains one concrete supported scope.
// It is used for capability declarations; runtime initialization still passes
// only one concrete scope to Config.Init.
func (declared Scope) Supports(scope Scope) bool {
	return declared != 0 && scope.valid() && declared&scope == scope
}
