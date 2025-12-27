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

package base

import (
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestNewBaseExporter(t *testing.T) {
	url, _ := common.NewURL("dubbo://localhost:9090")
	invoker := NewBaseInvoker(url)
	exporterMap := &sync.Map{}
	key := "test-key"

	exporter := NewBaseExporter(key, invoker, exporterMap)

	assert.NotNil(t, exporter)
	assert.Equal(t, key, exporter.key)
	assert.Equal(t, invoker, exporter.invoker)
	assert.Equal(t, exporterMap, exporter.exporterMap)
}

func TestBaseExporter_GetInvoker(t *testing.T) {
	url, _ := common.NewURL("dubbo://localhost:9090")
	invoker := NewBaseInvoker(url)
	exporterMap := &sync.Map{}
	key := "test-key"

	exporter := NewBaseExporter(key, invoker, exporterMap)
	gotInvoker := exporter.GetInvoker()

	assert.NotNil(t, gotInvoker)
	assert.Equal(t, invoker, gotInvoker)
	assert.Equal(t, url, gotInvoker.GetURL())
}

func TestBaseExporter_UnExport(t *testing.T) {
	url, _ := common.NewURL("dubbo://localhost:9090")
	invoker := NewBaseInvoker(url)
	exporterMap := &sync.Map{}
	key := "test-key"

	// Store the exporter in the map
	exporter := NewBaseExporter(key, invoker, exporterMap)
	exporterMap.Store(key, exporter)

	// Verify it's stored
	_, ok := exporterMap.Load(key)
	assert.True(t, ok)

	// Test UnExport
	exporter.UnExport()

	// Verify invoker is destroyed
	assert.True(t, invoker.IsDestroyed())
	assert.False(t, invoker.IsAvailable())

	// Verify exporter is removed from map
	_, ok = exporterMap.Load(key)
	assert.False(t, ok)
}

func TestBaseExporter_UnExport_MultipleEntries(t *testing.T) {
	exporterMap := &sync.Map{}

	// Create multiple exporters
	url1, _ := common.NewURL("dubbo://localhost:9090")
	invoker1 := NewBaseInvoker(url1)
	exporter1 := NewBaseExporter("key1", invoker1, exporterMap)
	exporterMap.Store("key1", exporter1)

	url2, _ := common.NewURL("dubbo://localhost:9091")
	invoker2 := NewBaseInvoker(url2)
	exporter2 := NewBaseExporter("key2", invoker2, exporterMap)
	exporterMap.Store("key2", exporter2)

	// Unexport first exporter
	exporter1.UnExport()

	// Verify first is removed, second remains
	_, ok1 := exporterMap.Load("key1")
	assert.False(t, ok1)
	_, ok2 := exporterMap.Load("key2")
	assert.True(t, ok2)

	// Verify only first invoker is destroyed
	assert.True(t, invoker1.IsDestroyed())
	assert.False(t, invoker2.IsDestroyed())
}
