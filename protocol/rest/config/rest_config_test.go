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

package config

import (
	"sync"
	"testing"
)

type restServiceConfigMapAccessors struct {
	set    func(map[string]*RestServiceConfig)
	get    func(string) *RestServiceConfig
	getAll func() map[string]*RestServiceConfig
}

func restServiceConfigMapAccessorTests() map[string]restServiceConfigMapAccessors {
	return map[string]restServiceConfigMapAccessors{
		"consumer": {
			set:    SetRestConsumerServiceConfigMap,
			get:    GetRestConsumerServiceConfig,
			getAll: GetRestConsumerServiceConfigMap,
		},
		"provider": {
			set:    SetRestProviderServiceConfigMap,
			get:    GetRestProviderServiceConfig,
			getAll: GetRestProviderServiceConfigMap,
		},
	}
}

func TestRestServiceConfigMapIsolation(t *testing.T) {
	for name, accessors := range restServiceConfigMapAccessorTests() {
		t.Run(name, func(t *testing.T) {
			original := accessors.getAll()
			t.Cleanup(func() {
				accessors.set(original)
			})

			first := &RestServiceConfig{InterfaceName: "first"}
			input := map[string]*RestServiceConfig{"first": first}
			accessors.set(input)

			delete(input, "first")
			input["second"] = &RestServiceConfig{InterfaceName: "second"}
			if got := accessors.get("first"); got != first {
				t.Fatalf("setter retained caller map: got %p, want %p", got, first)
			}
			if got := accessors.get("second"); got != nil {
				t.Fatalf("setter exposed later caller mutation: got %#v", got)
			}

			snapshot := accessors.getAll()
			if got := snapshot["first"]; got != first {
				t.Fatalf("snapshot changed value identity: got %p, want %p", got, first)
			}
			delete(snapshot, "first")
			snapshot["second"] = &RestServiceConfig{InterfaceName: "second"}
			if got := accessors.get("first"); got != first {
				t.Fatalf("snapshot mutation changed registry: got %p, want %p", got, first)
			}
			if got := accessors.get("second"); got != nil {
				t.Fatalf("snapshot mutation added registry value: got %#v", got)
			}

			replacement := &RestServiceConfig{InterfaceName: "replacement"}
			accessors.set(map[string]*RestServiceConfig{"second": replacement})
			if got := accessors.get("first"); got != nil {
				t.Fatalf("replacement retained stale value: got %#v", got)
			}
			if got := accessors.get("second"); got != replacement {
				t.Fatalf("replacement stored wrong value: got %p, want %p", got, replacement)
			}

			accessors.set(nil)
			if got := accessors.getAll(); got != nil {
				t.Fatalf("nil map semantics changed: got %#v", got)
			}
		})
	}
}

func TestRestServiceConfigMapConcurrentAccess(t *testing.T) {
	for name, accessors := range restServiceConfigMapAccessorTests() {
		t.Run(name, func(t *testing.T) {
			original := accessors.getAll()
			t.Cleanup(func() {
				accessors.set(original)
			})
			accessors.set(map[string]*RestServiceConfig{
				"service": {InterfaceName: "initial"},
			})

			const (
				goroutines = 16
				iterations = 200
			)
			start := make(chan struct{})
			var wg sync.WaitGroup
			for i := 0; i < goroutines; i++ {
				wg.Add(2)
				go func() {
					defer wg.Done()
					<-start
					for j := 0; j < iterations; j++ {
						input := map[string]*RestServiceConfig{
							"service": {InterfaceName: "published"},
						}
						accessors.set(input)
						delete(input, "service")
					}
				}()
				go func() {
					defer wg.Done()
					<-start
					for j := 0; j < iterations; j++ {
						if got := accessors.get("service"); got == nil {
							t.Error("single getter observed caller-side map deletion")
							return
						}
						snapshot := accessors.getAll()
						if snapshot["service"] == nil {
							t.Error("snapshot getter observed caller-side map deletion")
							return
						}
						delete(snapshot, "service")
					}
				}()
			}
			close(start)
			wg.Wait()
		})
	}
}
