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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestNewBaseProtocol(t *testing.T) {
	protocol := NewBaseProtocol()

	assert.NotNil(t, protocol)
	assert.NotNil(t, protocol.exporterMap)
	assert.Empty(t, protocol.invokers)
}

func TestBaseProtocol_SetExporterMap(t *testing.T) {
	protocol := NewBaseProtocol()
	url, _ := common.NewURL("dubbo://localhost:9090")
	invoker := NewBaseInvoker(url)
	exporter := NewBaseExporter("test-key", invoker, protocol.exporterMap)

	protocol.SetExporterMap("test-key", exporter)

	// Verify exporter is stored
	val, ok := protocol.exporterMap.Load("test-key")
	assert.True(t, ok)
	assert.Equal(t, exporter, val)
}

func TestBaseProtocol_ExporterMap(t *testing.T) {
	protocol := NewBaseProtocol()
	exporterMap := protocol.ExporterMap()

	assert.NotNil(t, exporterMap)
	assert.Equal(t, protocol.exporterMap, exporterMap)
}

func TestBaseProtocol_SetInvokers(t *testing.T) {
	protocol := NewBaseProtocol()
	url1, _ := common.NewURL("dubbo://localhost:9090")
	invoker1 := NewBaseInvoker(url1)
	url2, _ := common.NewURL("dubbo://localhost:9091")
	invoker2 := NewBaseInvoker(url2)

	protocol.SetInvokers(invoker1)
	protocol.SetInvokers(invoker2)

	invokers := protocol.Invokers()
	assert.Len(t, invokers, 2)
	assert.Contains(t, invokers, invoker1)
	assert.Contains(t, invokers, invoker2)
}

func TestBaseProtocol_Invokers(t *testing.T) {
	protocol := NewBaseProtocol()
	url, _ := common.NewURL("dubbo://localhost:9090")
	invoker := NewBaseInvoker(url)

	protocol.SetInvokers(invoker)
	invokers := protocol.Invokers()

	assert.Len(t, invokers, 1)
	assert.Equal(t, invoker, invokers[0])
}

func TestBaseProtocol_Export(t *testing.T) {
	protocol := NewBaseProtocol()
	url, _ := common.NewURL("dubbo://localhost:9090")
	invoker := NewBaseInvoker(url)

	exporter := protocol.Export(invoker)

	assert.NotNil(t, exporter)
	assert.Equal(t, invoker, exporter.GetInvoker())
	
	// Note: Export creates a BaseExporter but doesn't automatically store it
	// The exporter needs to be manually stored using SetExporterMap if needed
	baseExporter, ok := exporter.(*BaseExporter)
	assert.True(t, ok)
	assert.Equal(t, "base", baseExporter.key)
	assert.Equal(t, protocol.exporterMap, baseExporter.exporterMap)
}

func TestBaseProtocol_Refer(t *testing.T) {
	protocol := NewBaseProtocol()
	url, _ := common.NewURL("dubbo://localhost:9090")

	invoker := protocol.Refer(url)

	assert.NotNil(t, invoker)
	assert.Equal(t, url, invoker.GetURL())
	
	// Type assert to BaseInvoker to access IsAvailable and IsDestroyed
	baseInvoker, ok := invoker.(*BaseInvoker)
	assert.True(t, ok)
	assert.True(t, baseInvoker.IsAvailable())
	assert.False(t, baseInvoker.IsDestroyed())
}

func TestBaseProtocol_Destroy(t *testing.T) {
	protocol := NewBaseProtocol()

	// Add invokers
	url1, _ := common.NewURL("dubbo://localhost:9090")
	invoker1 := NewBaseInvoker(url1)
	protocol.SetInvokers(invoker1)

	url2, _ := common.NewURL("dubbo://localhost:9091")
	invoker2 := NewBaseInvoker(url2)
	protocol.SetInvokers(invoker2)

	// Add exporters
	exporter1 := NewBaseExporter("key1", invoker1, protocol.exporterMap)
	protocol.SetExporterMap("key1", exporter1)

	exporter2 := NewBaseExporter("key2", invoker2, protocol.exporterMap)
	protocol.SetExporterMap("key2", exporter2)

	// Verify setup
	assert.Len(t, protocol.Invokers(), 2)
	_, ok1 := protocol.exporterMap.Load("key1")
	assert.True(t, ok1)
	_, ok2 := protocol.exporterMap.Load("key2")
	assert.True(t, ok2)

	// Test Destroy
	protocol.Destroy()

	// Verify all invokers are destroyed
	assert.Empty(t, protocol.Invokers())
	assert.True(t, invoker1.IsDestroyed())
	assert.True(t, invoker2.IsDestroyed())

	// Verify all exporters are unexported and removed
	_, ok1 = protocol.exporterMap.Load("key1")
	assert.False(t, ok1)
	_, ok2 = protocol.exporterMap.Load("key2")
	assert.False(t, ok2)
}

func TestBaseProtocol_Destroy_WithNilInvoker(t *testing.T) {
	protocol := NewBaseProtocol()

	// Add a nil invoker
	protocol.invokers = append(protocol.invokers, nil)

	// Add a valid invoker
	url, _ := common.NewURL("dubbo://localhost:9090")
	invoker := NewBaseInvoker(url)
	protocol.SetInvokers(invoker)

	// Should not panic
	assert.NotPanics(t, func() {
		protocol.Destroy()
	})

	assert.Empty(t, protocol.Invokers())
}

func TestBaseProtocol_Destroy_WithNilExporter(t *testing.T) {
	protocol := NewBaseProtocol()

	// Add a nil exporter
	protocol.exporterMap.Store("nil-key", nil)

	// Add a valid exporter
	url, _ := common.NewURL("dubbo://localhost:9090")
	invoker := NewBaseInvoker(url)
	exporter := NewBaseExporter("valid-key", invoker, protocol.exporterMap)
	protocol.SetExporterMap("valid-key", exporter)

	// Should not panic
	assert.NotPanics(t, func() {
		protocol.Destroy()
	})

	// Verify nil entry is deleted
	_, ok := protocol.exporterMap.Load("nil-key")
	assert.False(t, ok)

	// Verify valid exporter is unexported
	_, ok = protocol.exporterMap.Load("valid-key")
	assert.False(t, ok)
}

func TestBaseProtocol_Destroy_EmptyProtocol(t *testing.T) {
	protocol := NewBaseProtocol()

	// Should not panic on empty protocol
	assert.NotPanics(t, func() {
		protocol.Destroy()
	})

	assert.Empty(t, protocol.Invokers())
}

func TestBaseProtocol_MultipleExportRefer(t *testing.T) {
	protocol := NewBaseProtocol()

	// Export multiple services
	url1, _ := common.NewURL("dubbo://localhost:9090/service1")
	invoker1 := NewBaseInvoker(url1)
	exporter1 := protocol.Export(invoker1)

	url2, _ := common.NewURL("dubbo://localhost:9091/service2")
	invoker2 := NewBaseInvoker(url2)
	exporter2 := protocol.Export(invoker2)

	// Refer multiple services
	url3, _ := common.NewURL("dubbo://localhost:9092/service3")
	referInvoker1 := protocol.Refer(url3)

	url4, _ := common.NewURL("dubbo://localhost:9093/service4")
	referInvoker2 := protocol.Refer(url4)

	// Verify all are created correctly
	assert.NotNil(t, exporter1)
	assert.NotNil(t, exporter2)
	assert.NotNil(t, referInvoker1)
	assert.NotNil(t, referInvoker2)

	assert.Equal(t, url1, exporter1.GetInvoker().GetURL())
	assert.Equal(t, url2, exporter2.GetInvoker().GetURL())
	assert.Equal(t, url3, referInvoker1.GetURL())
	assert.Equal(t, url4, referInvoker2.GetURL())
}
