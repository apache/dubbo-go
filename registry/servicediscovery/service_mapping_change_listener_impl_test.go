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

package servicediscovery

import (
	"sync"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// TestServiceMappingChangedListener_StopRace verifies that Stop() (which now
// takes lstn.mux) and OnEvent() (which reads `stop` under lstn.mux) do not race
// on the `stop` field. Run with -race. Regression for the issue where Stop wrote
// `stop` outside the mutex that OnEvent reads it under.
func TestServiceMappingChangedListener_StopRace(t *testing.T) {
	regURL, _ := common.NewURL("service-discovery://localhost:12345")
	svcURL, _ := common.NewURL("dubbo://127.0.0.1:20000/IFoo")
	lstn := NewMappingListener(regURL, svcURL, gxset.NewSet("app1"), nil)

	// OnEvent(nil) acquires mux, reads `stop`, then early-returns (a nil event is
	// not a *ServiceMappingChangeEvent), exercising exactly the Stop-vs-OnEvent
	// race on `stop`.
	var wg sync.WaitGroup
	const rounds = 100
	for range rounds {
		wg.Add(2)
		go func() { defer wg.Done(); lstn.Stop() }()
		go func() { defer wg.Done(); _ = lstn.OnEvent(nil) }()
	}
	wg.Wait()

	// After the concurrent Stops complete the listener must be stopped.
	lstn.mux.Lock()
	defer lstn.mux.Unlock()
	assert.Equal(t, ServiceMappingListenerStop, lstn.stop)
}
