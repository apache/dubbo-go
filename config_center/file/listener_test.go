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

package file

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type recListener struct {
	ch chan *config_center.ConfigChangeEvent
}

func (r *recListener) Process(e *config_center.ConfigChangeEvent) {
	select {
	case r.ch <- e:
	default:
	}
}

func TestCacheListenerCallbacks(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "cfg.txt")
	if err := os.WriteFile(filePath, []byte("init"), 0o644); err != nil {
		t.Fatalf("write file error %v", err)
	}

	cl := NewCacheListener(dir)

	rec := &recListener{ch: make(chan *config_center.ConfigChangeEvent, 4)}
	cl.AddListener(filePath, rec)

	// update event
	if err := os.WriteFile(filePath, []byte("update"), 0o644); err != nil {
		t.Fatalf("write file error %v", err)
	}
	waitEvent(t, rec.ch, remoting.EventTypeUpdate)

	// remove listener then cleanup
	cl.RemoveListener(filePath, rec)
	if err := cl.Close(); err != nil {
		t.Fatalf("close watcher error %v", err)
	}
	if err := os.Remove(filePath); err != nil {
		t.Fatalf("remove file error %v", err)
	}
	select {
	case <-rec.ch:
		// should not receive after removal
		t.Fatalf("unexpected event after remove")
	case <-time.After(200 * time.Millisecond):
	}
}

func waitEvent(t *testing.T, ch <-chan *config_center.ConfigChangeEvent, expect remoting.EventType) {
	t.Helper()
	select {
	case ev := <-ch:
		if ev.ConfigType != expect {
			t.Fatalf("expected %v, got %v", expect, ev.ConfigType)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for event %v", expect)
	}
}
