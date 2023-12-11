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
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
)

var defaultTime = time.Second * 3

func TestCacheManager(t *testing.T) {
	cm, err := NewCacheManager("test", "test_cache", defaultTime, 10, true)
	if err != nil {
		t.Fatalf("failed to create cache manager: %v", err)
	}

	// Test Set and Get
	cm.Set("key1", "value1")
	cm.Set("key2", "value2")
	value, ok := cm.Get("key1")
	if !ok {
		t.Errorf("failed to get key1: %v", err)
	}
	if value != "value1" {
		t.Errorf("unexpected Value for key1: got %v, want %v", value, "value1")
	}

	// Test Delete
	cm.Delete("key2")
	_, ok = cm.Get("key2")
	if ok {
		t.Errorf("key2 was not removed from cache")
	}

	// Test GetAll
	cm.Set("key3", "value3")
	all := cm.GetAll()
	if len(all) != 2 {
		t.Errorf("unexpected number of items in cache: got %d, want %d", len(all), 2)
	}

	// Test cache file creation and loading
	cm2, err := NewCacheManager("test2", "nonexistent_cache", defaultTime, 10, true)
	if err != nil {
		t.Fatalf("failed to create cache manager: %v", err)
	}
	cm2.Set("key4", "value4")
	time.Sleep(time.Second * 4)
	cm2.StopDump()
	cm3, err := NewCacheManager("test3", "nonexistent_cache", defaultTime, 10, true)
	if err != nil {
		t.Fatalf("failed to create cache manager: %v", err)
	}
	all3 := cm3.GetAll()
	if len(all3) != 1 {
		t.Errorf("unexpected number of items in cache: got %d, want %d", len(all3), 1)
	}
	_, ok = cm3.Get("key4")
	if !ok {
		t.Errorf("failed to get key4: %v", err)
	}
	cm3.destroy()
	cm2.destroy()
	cm.destroy() // clear cache file
}

func TestMetaInfoCacheManager(t *testing.T) {

	serverInfo := make(map[string]*info.ServiceInfo)
	serverInfo["1"] = info.NewServiceInfo("1", "1", "1", "1", "1", make(map[string]string))
	serverInfo["2"] = info.NewServiceInfo("2", "2", "2", "2", "2", make(map[string]string))
	serverInfo["3"] = info.NewServiceInfo("3", "3", "3", "3", "3", make(map[string]string))

	metadataInfo1 := info.NewMetadataInfoWithParams("1", "1", serverInfo)
	metadataInfo2 := info.NewMetadataInfoWithParams("2", "2", serverInfo)
	metadataInfo3 := info.NewMetadataInfoWithParams("3", "3", serverInfo)
	metadataInfo4 := info.NewMetadataInfoWithParams("4", "4", serverInfo)

	cm, err := NewCacheManager("metaTest1", "test_meta_cache", defaultTime, 10, true)
	if err != nil {
		t.Fatalf("failed to create cache manager: %v", err)
	}

	// Test Set and Get
	cm.Set("key1", metadataInfo1)
	cm.Set("key2", metadataInfo2)
	value, ok := cm.Get("key1")
	if !ok {
		t.Errorf("failed to get key1: %v", err)
	}
	if value != metadataInfo1 {
		t.Errorf("unexpected Value for key1: got %v, want %v", value, "value1")
	}

	// Test Delete
	cm.Delete("key2")
	_, ok = cm.Get("key2")
	if ok {
		t.Errorf("key2 was not removed from cache")
	}

	// Test GetAll
	cm.Set("key3", metadataInfo3)
	all := cm.GetAll()
	if len(all) != 2 {
		t.Errorf("unexpected number of items in cache: got %d, want %d", len(all), 2)
	}

	// Test cache file creation and loading
	cm2, err := NewCacheManager("metaTest2", "nonexistent_meta_cache", defaultTime, 10, true)
	if err != nil {
		t.Fatalf("failed to create cache manager: %v", err)
	}
	cm2.Set("key4", metadataInfo4)
	time.Sleep(time.Second * 4)
	cm2.StopDump()
	cm3, err := NewCacheManager("test3", "nonexistent_meta_cache", defaultTime, 10, true)
	if err != nil {
		t.Fatalf("failed to create cache manager: %v", err)
	}
	all3 := cm3.GetAll()
	if len(all3) != 1 {
		t.Errorf("unexpected number of items in cache: got %d, want %d", len(all3), 1)
	}
	_, ok = cm3.Get("key4")
	if !ok {
		t.Errorf("failed to get key4: %v", err)
	}
	cm3.destroy()
	cm2.destroy()
	cm.destroy() // clear cache file
}
