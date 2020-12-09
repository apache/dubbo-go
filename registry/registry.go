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

/*
 * -----------------------------------NOTICE---------------------------------------------
 * If there is no special case, you'd better inherit BaseRegistry and implement the
 * FacadeBasedRegistry interface instead of directly implementing the Registry interface.
 * --------------------------------------------------------------------------------------
 */
// Registry Extension - Registry
type Registry interface {
	common.Node

	// Register is used for service provider calling, register services
	// to registry. And it is also used for service consumer calling, register
	// services cared about, for dubbo's admin monitoring.
	Register(url *common.URL) error

	// UnRegister is required to support the contract:
	// 1. If it is the persistent stored data of dynamic=false, the
	//    registration data can not be found, then the IllegalStateException
	//    is thrown, otherwise it is ignored.
	// 2. Unregister according to the full url match.
	// url Registration information, is not allowed to be empty, e.g:
	// dubbo://10.20.153.10/org.apache.dubbo.foo.BarService?version=1.0.0&application=kylin
	UnRegister(url *common.URL) error

	// Subscribe is required to support the contract:
	// When creating new registry extension, pls select one of the
	// following modes.
	// Will remove in dubbogo version v1.1.0
	// mode1: return Listener with Next function which can return
	//        subscribe service event from registry
	// Deprecated!
	// subscribe(event.URL) (Listener, error)
	// Will replace mode1 in dubbogo version v1.1.0
	// mode2: callback mode, subscribe with notify(notify listener).
	Subscribe(*common.URL, NotifyListener) error

	// UnSubscribe is required to support the contract:
	// 1. If don't subscribe, ignore it directly.
	// 2. Unsubscribe by full URL match.
	// url Subscription condition, not allowed to be empty, e.g.
	// consumer://10.20.153.10/org.apache.dubbo.foo.BarService?version=1.0.0&application=kylin
	// listener A listener of the change event, not allowed to be empty
	UnSubscribe(*common.URL, NotifyListener) error
}

// nolint
type NotifyListener interface {
	// Notify supports notifications on the service interface and the dimension of the data type. When a list of
	// events are passed in, it's considered as a complete list, on the other side, if one single event is
	// passed in, then it's a incremental event. Pls. note when a list (instead of single event) comes,
	// the impl of NotifyListener may abandon the accumulated result from previous notifications.
	Notify(...*ServiceEvent)
}

// Listener Deprecated!
type Listener interface {
	// Next returns next service event once received
	Next() (*ServiceEvent, error)
	// Close closes this listener
	Close()
}
