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

package matcher

import (
	"sync"
	"testing"
)

func TestGetMatcherFactoriesReturnsSnapshot(t *testing.T) {
	const (
		registeredName = "snapshot-registered-matcher"
		injectedName   = "snapshot-injected-matcher"
	)
	t.Cleanup(func() {
		matchersMu.Lock()
		delete(matchers, registeredName)
		matchersMu.Unlock()
	})
	SetMatcherFactory(registeredName, NewArgumentMatcherFactory)

	snapshot := GetMatcherFactories()
	delete(snapshot, registeredName)
	snapshot[injectedName] = NewAttachmentMatcherFactory

	current := GetMatcherFactories()
	if _, ok := current[registeredName]; !ok {
		t.Fatalf("deleting from the returned map removed %q from the registry", registeredName)
	}
	if _, ok := current[injectedName]; ok {
		t.Fatalf("adding to the returned map injected %q into the registry", injectedName)
	}
}

func TestMatcherFactoryRegistryConcurrentAccess(t *testing.T) {
	const (
		name       = "concurrent-matcher"
		iterations = 1000
		readers    = 4
	)
	t.Cleanup(func() {
		matchersMu.Lock()
		delete(matchers, name)
		matchersMu.Unlock()
	})
	SetMatcherFactory(name, NewArgumentMatcherFactory)

	var waitGroup sync.WaitGroup
	waitGroup.Add(readers + 1)
	go func() {
		defer waitGroup.Done()
		for index := 0; index < iterations; index++ {
			if index%2 == 0 {
				SetMatcherFactory(name, NewArgumentMatcherFactory)
			} else {
				SetMatcherFactory(name, NewAttachmentMatcherFactory)
			}
		}
	}()
	for range readers {
		go func() {
			defer waitGroup.Done()
			for range iterations {
				_ = GetMatcherFactory(name)
				_ = GetMatcherFactories()
			}
		}()
	}
	waitGroup.Wait()
}
