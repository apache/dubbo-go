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
	"fmt"
	"sync"
)

// CheckFunc returns nil when healthy.
type CheckFunc func(context.Context) error

var (
	livenessMu  sync.RWMutex
	readinessMu sync.RWMutex
	startupMu   sync.RWMutex

	livenessChecks  = make(map[string]CheckFunc)
	readinessChecks = make(map[string]CheckFunc)
	startupChecks   = make(map[string]CheckFunc)
)

// RegisterLiveness registers a liveness check.
func RegisterLiveness(name string, fn CheckFunc) {
	if name == "" || fn == nil {
		return
	}
	livenessMu.Lock()
	defer livenessMu.Unlock()
	livenessChecks[name] = fn
}

// RegisterReadiness registers a readiness check.
func RegisterReadiness(name string, fn CheckFunc) {
	if name == "" || fn == nil {
		return
	}
	readinessMu.Lock()
	defer readinessMu.Unlock()
	readinessChecks[name] = fn
}

// RegisterStartup registers a startup check.
func RegisterStartup(name string, fn CheckFunc) {
	if name == "" || fn == nil {
		return
	}
	startupMu.Lock()
	defer startupMu.Unlock()
	startupChecks[name] = fn
}

func runChecks(ctx context.Context, mu *sync.RWMutex, checks map[string]CheckFunc) error {
	mu.RLock()
	defer mu.RUnlock()
	for name, fn := range checks {
		if err := fn(ctx); err != nil {
			return fmt.Errorf("probe %s: %w", name, err)
		}
	}
	return nil
}

// CheckLiveness evaluates all liveness checks.
func CheckLiveness(ctx context.Context) error {
	return runChecks(ctx, &livenessMu, livenessChecks)
}

// CheckReadiness evaluates all readiness checks.
func CheckReadiness(ctx context.Context) error {
	return runChecks(ctx, &readinessMu, readinessChecks)
}

// CheckStartup evaluates all startup checks.
func CheckStartup(ctx context.Context) error {
	return runChecks(ctx, &startupMu, startupChecks)
}
