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

package metrics

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRecorderIndependentOfChannelSubscribers(t *testing.T) {
	const typ = "test.synchronous.recorder"
	var records atomic.Int64
	SubscribeRecorder(typ, func(MetricsEvent) { records.Add(1) })
	defer SubscribeRecorder(typ, nil)
	ch := make(chan MetricsEvent, 1)
	Subscribe(typ, ch)
	event := NewMockEventWithType(typ)
	Publish(event)
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 128 {
				Publish(event)
			}
		})
	}
	wg.Wait()
	if got := records.Load(); got != 4097 {
		t.Fatalf("recorded %d", got)
	}
	if len(ch) != 1 {
		t.Fatal("observer should remain bounded")
	}
	Unsubscribe(typ)
	Publish(event)
	if records.Load() != 4098 {
		t.Fatal("unsubscribing observer removed recorder")
	}
	SubscribeRecorder(typ, nil)
	Publish(event)
	if records.Load() != 4098 {
		t.Fatal("removed recorder still called")
	}
}

func TestRecorderMayChangeSubscriptions(t *testing.T) {
	const typ = "test.recorder.reentrant"
	SubscribeRecorder(typ, func(MetricsEvent) { Subscribe("test.recorder.observer", make(chan MetricsEvent, 1)) })
	defer SubscribeRecorder(typ, nil)
	defer Unsubscribe("test.recorder.observer")
	Publish(NewMockEventWithType(typ))
}
