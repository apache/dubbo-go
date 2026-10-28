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

// TestDynamicConfigurationRace verifies that concurrent SetDynamicConfiguration
// and GetDynamicConfiguration calls on the global Environment instance are
// race-free. It mirrors the CI flaky race where a registry subscription
// goroutine kept reading the field while a test goroutine wrote it.
func TestDynamicConfigurationRace(t *testing.T) {
	env := GetEnvInstance()
	previous := env.GetDynamicConfiguration()
	t.Cleanup(func() {
		env.SetDynamicConfiguration(previous)
	})

	const iterations = 100000
	start := make(chan struct{})
	ready := make(chan struct{})

	var wg sync.WaitGroup
	wg.Go(func() { // reader goroutine mimics a lingering registry subscription goroutine
		close(ready)
		<-start
		for range iterations {
			_ = env.GetDynamicConfiguration()
		}
	})

	<-ready
	close(start)
	for range iterations { // the test goroutine keeps writing
		env.SetDynamicConfiguration(nil)
	}
	wg.Wait()
}
