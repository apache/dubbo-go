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

package serialization

import (
	"errors"
	"github.com/stretchr/testify/require"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type closeProbe struct{ calls *atomic.Int64 }

func (closeProbe) Name() string                { return "probe" }
func (closeProbe) Marshal(any) ([]byte, error) { return nil, nil }
func (closeProbe) Unmarshal([]byte, any) error { return nil }
func (c closeProbe) Close() error              { c.calls.Add(1); return nil }
func TestResourcesDrainAndCloseOnce(t *testing.T) {
	var calls atomic.Int64
	resources := &Resources{}
	resources.Add(closeProbe{&calls})
	if !resources.Begin() {
		t.Fatal("initial acquisition failed")
	}
	closed := make(chan struct{})
	go func() { resources.Close(); close(closed) }()
	deadline := time.Now().Add(time.Second)
	for {
		resources.mu.Lock()
		closing := resources.closing
		resources.mu.Unlock()
		if closing {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Close did not begin")
		}
		time.Sleep(time.Millisecond)
	}
	if resources.Begin() {
		resources.End()
		t.Fatal("acquisition after shutdown")
	}
	if calls.Load() != 0 {
		t.Fatal("codec closed during in-flight use")
	}
	resources.End()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not drain")
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); resources.Close() }()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("Close count %d", calls.Load())
	}
}

type orderedCloseProbe struct {
	closeProbe
	id    int
	order *[]int
	err   error
}

func (c orderedCloseProbe) Close() error {
	*c.order = append(*c.order, c.id)
	c.calls.Add(1)
	return c.err
}
func TestCodecResourcesCloseErrors(t *testing.T) {
	first, second := errors.New("first close"), errors.New("second close")
	var calls atomic.Int64
	var order []int
	r := &Resources{}
	for i, err := range []error{first, nil, second} {
		r.Add(orderedCloseProbe{closeProbe{&calls}, i, &order, err})
	}
	err := r.Close()
	require.ErrorIs(t, err, first)
	require.ErrorIs(t, err, second)
	require.Equal(t, []int{2, 1, 0}, order)
	require.Same(t, err, r.Close())
	require.EqualValues(t, 3, calls.Load())
}
