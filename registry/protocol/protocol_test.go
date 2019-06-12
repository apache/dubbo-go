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

package protocol

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	cluster "github.com/apache/dubbo-go/cluster/cluster_impl"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/protocolwrapper"
	"github.com/apache/dubbo-go/registry"
)

func referNormal(t *testing.T, regProtocol *registryProtocol) {
	extension.SetProtocol("registry", GetProtocol)
	extension.SetRegistry("mock", registry.NewMockRegistry)
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
	extension.SetCluster("mock", cluster.NewMockCluster)

	url, _ := common.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000//", common.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url.SubURL = &suburl

	invoker := regProtocol.Refer(url)
	assert.IsType(t, &protocol.BaseInvoker{}, invoker)
	assert.Equal(t, invoker.GetUrl().String(), url.String())
}

func TestRefer(t *testing.T) {
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)
}

func TestMultiRegRefer(t *testing.T) {
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)
	url2, _ := common.NewURL(context.TODO(), "mock://127.0.0.1:2222")
	suburl2, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000//", common.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url2.SubURL = &suburl2

	regProtocol.Refer(url2)
	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 2)
}

func TestOneRegRefer(t *testing.T) {
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)

	url2, _ := common.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl2, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000//", common.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url2.SubURL = &suburl2

	regProtocol.Refer(url2)
	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 1)
}

func exporterNormal(t *testing.T, regProtocol *registryProtocol) {
	extension.SetProtocol("registry", GetProtocol)
	extension.SetRegistry("mock", registry.NewMockRegistry)
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
	url, _ := common.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000//", common.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url.SubURL = &suburl
	invoker := protocol.NewBaseInvoker(url)
	exporter := regProtocol.Export(invoker)

	assert.IsType(t, &protocol.BaseExporter{}, exporter)
	assert.Equal(t, exporter.GetInvoker().GetUrl().String(), suburl.String())
}

func TestExporter(t *testing.T) {
	regProtocol := newRegistryProtocol()
	exporterNormal(t, regProtocol)
}

func TestMultiRegAndMultiProtoExporter(t *testing.T) {
	regProtocol := newRegistryProtocol()
	exporterNormal(t, regProtocol)

	url2, _ := common.NewURL(context.TODO(), "mock://127.0.0.1:2222")
	suburl2, _ := common.NewURL(context.TODO(), "jsonrpc://127.0.0.1:20000//", common.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url2.SubURL = &suburl2
	invoker2 := protocol.NewBaseInvoker(url2)
	regProtocol.Export(invoker2)

	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 2)

	var count2 int
	regProtocol.bounds.Range(func(key, value interface{}) bool {
		count2++
		return true
	})
	assert.Equal(t, count2, 2)
}

func TestOneRegAndProtoExporter(t *testing.T) {
	regProtocol := newRegistryProtocol()
	exporterNormal(t, regProtocol)

	url2, _ := common.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl2, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000//", common.WithParamsValue(constant.CLUSTER_KEY, "mock"))

	url2.SubURL = &suburl2
	invoker2 := protocol.NewBaseInvoker(url2)
	regProtocol.Export(invoker2)

	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 1)

	var count2 int
	regProtocol.bounds.Range(func(key, value interface{}) bool {
		count2++
		return true
	})
	assert.Equal(t, count2, 1)
}

func TestDestry(t *testing.T) {
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)
	exporterNormal(t, regProtocol)

	regProtocol.Destroy()
	assert.Equal(t, len(regProtocol.invokers), 0)

	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 0)

	var count2 int
	regProtocol.bounds.Range(func(key, value interface{}) bool {
		count2++
		return true
	})
	assert.Equal(t, count2, 0)
}
