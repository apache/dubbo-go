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
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlersSuccess(t *testing.T) {
	resetProbeState()

	RegisterLiveness("ok", func(context.Context) error { return nil })
	RegisterReadiness("ok", func(context.Context) error { return nil })
	RegisterStartup("ok", func(context.Context) error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rec := httptest.NewRecorder()
	livenessHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for liveness, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec = httptest.NewRecorder()
	readinessHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for readiness, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/startup", nil)
	rec = httptest.NewRecorder()
	startupHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for startup, got %d", rec.Code)
	}
}

func TestHandlersFailure(t *testing.T) {
	resetProbeState()

	RegisterLiveness("fail", func(context.Context) error { return errors.New("bad") })
	RegisterReadiness("fail", func(context.Context) error { return errors.New("bad") })
	RegisterStartup("fail", func(context.Context) error { return errors.New("bad") })

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rec := httptest.NewRecorder()
	livenessHandler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for liveness, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec = httptest.NewRecorder()
	readinessHandler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for readiness, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/startup", nil)
	rec = httptest.NewRecorder()
	startupHandler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for startup, got %d", rec.Code)
	}
}
