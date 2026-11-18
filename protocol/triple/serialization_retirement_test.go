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

package triple

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/emptypb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

func TestCodecServiceRetirementDropsServerOwnership(t *testing.T) {
	server := NewServer(nil)
	server.triServer = tri.NewServer("", nil)
	for i := range 4 {
		var closes atomic.Int64
		owner := &serialization.Resources{}
		owner.Add(&providerLeaseCodec{closes: &closes})
		name := fmt.Sprintf("codec.Retirement%d", i)
		url := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithPath(name), common.WithParamsValue("interface", name))
		url.SetAttribute("triple.serialization.resources.provider", owner)
		info := &common.ServiceInfo{Methods: []common.MethodInfo{{Name: "Call", Type: constant.CallUnary, ReqInitFunc: func() any { return &emptypb.Empty{} }}}}
		server.registerServiceHandlers(base.NewBaseInvoker(url), info, []tri.HandlerOption{tri.WithSerializationResources(owner)})
		require.Len(t, server.codecResources, 1)
		require.NoError(t, server.retireCodec(owner))
		require.NoError(t, server.retireCodec(owner))
		require.Empty(t, server.codecResources)
		require.Empty(t, server.codecServices)
		require.Empty(t, server.GetServiceInfo())
		require.EqualValues(t, 1, closes.Load())
	}
}
func TestOldTripleExporterCannotRemoveReplacement(t *testing.T) {
	url := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithPath("codec.Exporter"))
	var exporters sync.Map
	oldInvoker, newInvoker := base.NewBaseInvoker(url.Clone()), base.NewBaseInvoker(url.Clone())
	old := NewTripleExporter("key", oldInvoker, &exporters)
	next := NewTripleExporter("key", newInvoker, &exporters)
	var retired atomic.Int64
	old.retire = func() error { retired.Add(1); return nil }
	exporters.Store("key", next)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(old.UnExport)
	}
	wg.Wait()
	current, ok := exporters.Load("key")
	require.True(t, ok)
	require.Same(t, next, current)
	require.True(t, oldInvoker.IsDestroyed())
	require.False(t, newInvoker.IsDestroyed())
	require.EqualValues(t, 1, retired.Load())
	newInvoker.Destroy()
}

func TestLegacyExportReleasesLazilyConstructedCodec(t *testing.T) {
	const name = "legacy-export-retirement"
	probe := &codecEpochProbe{name: name}
	require.NoError(t, extension.RegisterSerialization(serialization.Registration{Protocol: "triple", Name: name, WireName: "proto", Mode: serialization.WrappedObjectMode, New: func(serialization.Options) (serialization.Codec, error) { return probe, nil }}))
	tp := NewTripleProtocol()
	server := NewServer(nil)
	server.triServer = tri.NewServer("", nil)
	t.Cleanup(tp.Destroy)
	url := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithLocation("127.0.0.1:0"), common.WithPath("codec.LegacyExport"), common.WithParamsValue("interface", "codec.LegacyExport"), common.WithParamsValue("serialization", name))
	url.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{Methods: []common.MethodInfo{{Name: "Call", Type: constant.CallUnary, ReqInitFunc: func() any { return &emptypb.Empty{} }}}})
	tp.serverMap[url.Location] = server
	_, prepared := url.GetAttribute("triple.serialization.resources.provider")
	require.False(t, prepared)
	exported := tp.Export(base.NewBaseInvoker(url))
	require.Len(t, server.codecResources, 1)
	require.Zero(t, probe.closes.Load())
	exported.UnExport()
	require.EqualValues(t, 1, probe.closes.Load())
	require.Empty(t, server.codecResources)
}
