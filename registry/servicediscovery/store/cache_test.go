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
	"testing"
)

func TestLruCache(t *testing.T) {
	maxCacheSize := 3

	lc := NewLruCache(maxCacheSize)

	// Test Set and Get methods
	lc.Set("key1", "value1")
	lc.Set("key2", "value2")
	lc.Set("key3", "value3")

	val, err := lc.Get("key1")
	if err != nil || val != "value1" {
		t.Errorf("Expected 'value1' for Key 'key1', but got '%v' with error '%v'", val, err)
	}

	val, err = lc.Get("key4")
	if err == nil || val != nil {
		t.Errorf("Expected error 'Key not found' for Key 'key4', but got '%v' with no error", val)
	}

	// Test LRU eviction
	lc.Set("key4", "value4")
	lc.Set("key5", "value5")
	val, err = lc.Get("key2")
	if err == nil || val != nil {
		t.Errorf("Expected error 'Key not found' for evicted Key 'key2', but got '%v' with no error", val)
	}

	// Test Remove method
	lc.Remove("key4")
	val, err = lc.Get("key4")
	if err == nil || val != nil {
		t.Errorf("Expected error 'Key not found' for removed Key 'key4', but got '%v' with no error", val)
	}

	// Test GetAll method
	all := lc.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected to get 2 entries from GetAll, but got %d", len(all))
	}
	if all["key1"] != "value1" || all["key5"] != "value5" {
		t.Errorf("Unexpected entries returned by GetAll: %v", all)
	}

	// Test Clear method
	lc.Clear()
	val, err = lc.Get("key1")
	if err == nil || val != nil {
		t.Errorf("Expected error 'Key not found' for cleared Key 'key1', but got '%v' with no error", val)
	}

	all = lc.GetAll()
	if len(all) != 0 {
		t.Errorf("Expected to get empty map from GetAll after Clear, but got %v", all)
	}
}
