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

package pattern_value

import (
	"sync"
	"testing"
)

func TestGetValuePatternsReturnsSnapshot(t *testing.T) {
	const (
		registeredName = "snapshot-registered-pattern"
		injectedName   = "snapshot-injected-pattern"
	)
	t.Cleanup(func() {
		valuePatternsMu.Lock()
		delete(valuePatterns, registeredName)
		valuePatternsMu.Unlock()
	})
	SetValuePattern(registeredName, NewWildcardValuePattern)

	snapshot := GetValuePatterns()
	delete(snapshot, registeredName)
	snapshot[injectedName] = NewScopeValuePattern

	current := GetValuePatterns()
	if _, ok := current[registeredName]; !ok {
		t.Fatalf("deleting from the returned map removed %q from the registry", registeredName)
	}
	if _, ok := current[injectedName]; ok {
		t.Fatalf("adding to the returned map injected %q into the registry", injectedName)
	}
}

func TestValuePatternRegistryConcurrentAccess(t *testing.T) {
	const (
		name       = "concurrent-pattern"
		iterations = 1000
		readers    = 4
	)
	t.Cleanup(func() {
		valuePatternsMu.Lock()
		delete(valuePatterns, name)
		valuePatternsMu.Unlock()
	})
	SetValuePattern(name, NewWildcardValuePattern)

	var waitGroup sync.WaitGroup
	waitGroup.Add(readers + 1)
	go func() {
		defer waitGroup.Done()
		for index := 0; index < iterations; index++ {
			if index%2 == 0 {
				SetValuePattern(name, NewWildcardValuePattern)
			} else {
				SetValuePattern(name, NewScopeValuePattern)
			}
		}
	}()
	for range readers {
		go func() {
			defer waitGroup.Done()
			for range iterations {
				_ = GetValuePattern(name)
				_ = GetValuePatterns()
			}
		}()
	}
	waitGroup.Wait()
}
