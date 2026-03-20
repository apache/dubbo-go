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

import (
	"container/list"
	"context"
	"testing"
)

func TestGracefulShutdownCallbacksReturnsSnapshot(t *testing.T) {
	t.Cleanup(func() {
		UnregisterGracefulShutdownCallback("snapshot-test")
	})

	RegisterGracefulShutdownCallback("snapshot-test", func(context.Context) error {
		return nil
	})

	callbacks := GracefulShutdownCallbacks()
	delete(callbacks, "snapshot-test")

	if _, ok := LookupGracefulShutdownCallback("snapshot-test"); !ok {
		t.Fatal("expected stored callback to remain after mutating returned snapshot")
	}
}

func TestGetAllCustomShutdownCallbacksReturnsSnapshot(t *testing.T) {
	customShutdownCallbacksMu.Lock()
	original := customShutdownCallbacks
	customShutdownCallbacks = list.New()
	customShutdownCallbacksMu.Unlock()

	t.Cleanup(func() {
		customShutdownCallbacksMu.Lock()
		customShutdownCallbacks = original
		customShutdownCallbacksMu.Unlock()
	})

	AddCustomShutdownCallback(func() {})
	callbacks := GetAllCustomShutdownCallbacks()
	callbacks.PushBack(func() {})

	if got := GetAllCustomShutdownCallbacks().Len(); got != 1 {
		t.Fatalf("expected custom callback snapshot mutation not to affect stored callbacks, got %d", got)
	}
}
