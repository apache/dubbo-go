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

package registry

import (
	"github.com/apache/dubbo-go/common"
)

// Extension - Registry
type Registry interface {
	common.Node
	//used for service provider calling , register services to registry
	//And it is also used for service consumer calling , register services cared about ,for dubbo's admin monitoring.
	Register(url common.URL) error

	//When creating new registry extension,pls select one of the following modes.
	//Will remove in dubbogo version v1.1.0
	//mode1 : return Listener with Next function which can return subscribe service event from registry
	//Deprecated!
	//subscribe(common.URL) (Listener, error)

	//Will relace mode1 in dubbogo version v1.1.0
	//mode2 : callback mode, subscribe with notify(notify listener).
	Subscribe(*common.URL, NotifyListener)
}
type NotifyListener interface {
	Notify(*ServiceEvent)
}

//Deprecated!
type Listener interface {
	Next() (*ServiceEvent, error)
	Close()
}
