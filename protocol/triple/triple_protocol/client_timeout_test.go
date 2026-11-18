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

package triple_protocol

import (
	"context"
	"testing"
	"time"
)

func TestApplyDefaultTimeout(t *testing.T) {
	t.Parallel()

	assertDeadlineNear := func(t *testing.T, ctx context.Context, want time.Duration) {
		t.Helper()
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected a deadline near %s", want)
		}
		remain := time.Until(deadline)
		// Allow scheduling slack, but keep the bound tight enough to
		// distinguish 5s call-level values from a 1s config fallback.
		if remain < want-200*time.Millisecond || remain > want {
			t.Fatalf("deadline remaining %s, want about %s", remain, want)
		}
	}

	t.Run("numeric milliseconds without unit", func(t *testing.T) {
		t.Parallel()
		ctx := context.WithValue(context.Background(), TimeoutKey{}, "5000")
		got, applied, cancel := applyDefaultTimeout(ctx, time.Second)
		if applied {
			t.Cleanup(cancel)
		}
		if !applied {
			t.Fatal("numeric millisecond timeout must apply a deadline")
		}
		assertDeadlineNear(t, got, 5*time.Second)
	})

	t.Run("legal Go duration", func(t *testing.T) {
		t.Parallel()
		ctx := context.WithValue(context.Background(), TimeoutKey{}, "2s")
		got, applied, cancel := applyDefaultTimeout(ctx, time.Second)
		if applied {
			t.Cleanup(cancel)
		}
		if !applied {
			t.Fatal("Go duration timeout must apply a deadline")
		}
		assertDeadlineNear(t, got, 2*time.Second)
	})

	t.Run("existing deadline is not extended", func(t *testing.T) {
		t.Parallel()
		base, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		t.Cleanup(cancel)
		original, ok := base.Deadline()
		if !ok {
			t.Fatal("precondition: base context must have a deadline")
		}
		ctx := context.WithValue(base, TimeoutKey{}, "10s")
		got, applied, innerCancel := applyDefaultTimeout(ctx, time.Second)
		if applied {
			t.Cleanup(innerCancel)
		}
		if applied {
			t.Fatal("an existing deadline must not be replaced or extended")
		}
		gotDeadline, ok := got.Deadline()
		if !ok {
			t.Fatal("existing deadline must be preserved")
		}
		if !gotDeadline.Equal(original) {
			t.Fatalf("deadline changed from %s to %s", original, gotDeadline)
		}
	})

	t.Run("config timeout is used when attachment is absent", func(t *testing.T) {
		t.Parallel()
		got, applied, cancel := applyDefaultTimeout(context.Background(), 3*time.Second)
		if applied {
			t.Cleanup(cancel)
		}
		if !applied {
			t.Fatal("config timeout must remain the no-attachment fallback")
		}
		assertDeadlineNear(t, got, 3*time.Second)
	})

	t.Run("no deadline is invented when attachment and config are empty", func(t *testing.T) {
		t.Parallel()
		got, applied, cancel := applyDefaultTimeout(context.Background(), 0)
		if applied {
			t.Cleanup(cancel)
		}
		if applied {
			t.Fatal("must not introduce a default countdown")
		}
		if _, ok := got.Deadline(); ok {
			t.Fatal("must not invent a deadline")
		}
	})

	t.Run("illegal attachment does not invent a deadline; config fallback applies", func(t *testing.T) {
		t.Parallel()
		ctx := context.WithValue(context.Background(), TimeoutKey{}, "not-a-duration")
		got, applied, cancel := applyDefaultTimeout(ctx, time.Second)
		if applied {
			t.Cleanup(cancel)
		}
		if !applied {
			t.Fatal("config timeout must still apply when the attachment is illegal")
		}
		assertDeadlineNear(t, got, time.Second)
	})

	t.Run("unitless non-positive does not take the millisecond fallback", func(t *testing.T) {
		t.Parallel()
		ctx := context.WithValue(context.Background(), TimeoutKey{}, "-1")
		got, applied, cancel := applyDefaultTimeout(ctx, time.Second)
		if applied {
			t.Cleanup(cancel)
		}
		if !applied {
			t.Fatal("config timeout must still apply when unitless attachment is non-positive")
		}
		assertDeadlineNear(t, got, time.Second)
	})
}
