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

package probe

import (
	"testing"
)

func TestStateDefaultsAndToggle(t *testing.T) {
	resetProbeState()

	if internalReady() != true {
		t.Fatalf("expected internalReady true when internal state disabled")
	}
	if internalStartup() != true {
		t.Fatalf("expected internalStartup true when internal state disabled")
	}

	EnableInternalState(true)
	if internalReady() {
		t.Fatalf("expected internalReady false when enabled and not ready")
	}
	if internalStartup() {
		t.Fatalf("expected internalStartup false when enabled and not started")
	}

	SetReady(true)
	SetStartupComplete(true)
	if !internalReady() {
		t.Fatalf("expected internalReady true after SetReady(true)")
	}
	if !internalStartup() {
		t.Fatalf("expected internalStartup true after SetStartupComplete(true)")
	}
}
