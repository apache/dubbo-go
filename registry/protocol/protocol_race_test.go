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
	"sync"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type noopConfigurator struct {
	url *common.URL
}

func (c *noopConfigurator) GetUrl() *common.URL {
	return c.url
}

func (*noopConfigurator) Configure(*common.URL) {}

func TestOverrideSubscribeListenerConfiguratorConcurrentAccess(t *testing.T) {
	configuratorURL, err := common.NewURL("override://127.0.0.1:20000/org.apache.dubbo-go.mockService")
	if err != nil {
		t.Fatalf("new configurator url failed: %v", err)
	}
	targetURL, err := common.NewURL("dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService")
	if err != nil {
		t.Fatalf("new target url failed: %v", err)
	}

	listener := &overrideSubscribeListener{}
	configurator := &noopConfigurator{url: configuratorURL}

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			listener.setConfigurator(configurator)
		}()
		go func() {
			defer wg.Done()
			if current := listener.getConfigurator(); current != nil {
				current.Configure(targetURL)
			}
		}()
	}
	wg.Wait()
}
