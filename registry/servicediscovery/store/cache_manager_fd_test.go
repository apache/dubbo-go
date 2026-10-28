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

package store

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

import (
	lru "github.com/hashicorp/golang-lru"
)

func TestLoadCacheClosesFileOnDecodeError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux /proc fd accounting")
	}

	cacheFile := filepath.Join(t.TempDir(), "cache")
	if err := os.WriteFile(cacheFile, []byte("corrupt gob"), 0o600); err != nil {
		t.Fatal(err)
	}
	cache, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	cm := &CacheManager{cacheFile: cacheFile, cache: cache}

	before := countOpenFiles(t)
	for range 128 {
		if err := cm.loadCache(); err == nil {
			t.Fatal("loadCache should return the decoder error")
		}
	}
	after := countOpenFiles(t)
	if after > before+2 {
		t.Fatalf("open file descriptors grew from %d to %d", before, after)
	}
}

func countOpenFiles(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}
