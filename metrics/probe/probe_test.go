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
	"context"
	"errors"
	"testing"
)

func resetProbeState() {
	livenessMu.Lock()
	livenessChecks = map[string]CheckFunc{}
	livenessMu.Unlock()

	readinessMu.Lock()
	readinessChecks = map[string]CheckFunc{}
	readinessMu.Unlock()

	startupMu.Lock()
	startupChecks = map[string]CheckFunc{}
	startupMu.Unlock()

	internalStateEnabled.Store(false)
	readyFlag.Store(false)
	startupFlag.Store(false)
}

func TestRegisterAndCheckLiveness(t *testing.T) {
	resetProbeState()

	RegisterLiveness("", func(context.Context) error { return nil })
	RegisterLiveness("nil", nil)

	livenessMu.RLock()
	if got := len(livenessChecks); got != 0 {
		livenessMu.RUnlock()
		t.Fatalf("expected 0 liveness checks, got %d", got)
	}
	livenessMu.RUnlock()

	RegisterLiveness("ok", func(context.Context) error { return nil })
	if err := CheckLiveness(context.Background()); err != nil {
		t.Fatalf("expected liveness ok, got %v", err)
	}

	RegisterLiveness("fail", func(context.Context) error { return errors.New("boom") })
	if err := CheckLiveness(context.Background()); err == nil {
		t.Fatalf("expected liveness error, got nil")
	}
}

func TestCheckReadinessAndStartupDefault(t *testing.T) {
	resetProbeState()

	if err := CheckReadiness(context.Background()); err != nil {
		t.Fatalf("expected readiness ok with no checks, got %v", err)
	}
	if err := CheckStartup(context.Background()); err != nil {
		t.Fatalf("expected startup ok with no checks, got %v", err)
	}
}

func TestInternalStateFlags(t *testing.T) {
	resetProbeState()

	EnableInternalState(true)
	if internalReady() {
		t.Fatalf("expected internalReady false by default")
	}
	if internalStartup() {
		t.Fatalf("expected internalStartup false by default")
	}

	SetReady(true)
	SetStartupComplete(true)
	if !internalReady() {
		t.Fatalf("expected internalReady true after SetReady(true)")
	}
	if !internalStartup() {
		t.Fatalf("expected internalStartup true after SetStartupComplete(true)")
	}

	EnableInternalState(false)
	if !internalReady() || !internalStartup() {
		t.Fatalf("expected internal state checks bypassed when disabled")
	}
}
