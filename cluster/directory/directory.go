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

package directory

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// Directory implementations include RegistryDirectory, ServiceDiscoveryRegistryDirectory, StaticDirectory
type Directory interface {
	common.Node

	// List candidate invoker list for the current Directory.
	// NOTICE: The invoker list returned to the caller may be backed by the same data hold by the current Directory
	// implementation for the sake of performance consideration. This requires the caller of List() shouldn't modify
	// the return result directly.
	List(invocation protocol.Invocation) []protocol.Invoker

	// Subscribe listen to registry instances
	Subscribe(url *common.URL) error
}
