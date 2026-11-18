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
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type publicationCodec struct{ exportGenerationCodec }

func (*publicationCodec) Name() string { return "publication-json" }

type publicationInvoker struct {
	*base.BaseInvoker
	label         string
	entered, gate chan struct{}
}

func (c *publicationInvoker) Invoke(context.Context, base.Invocation) result.Result {
	if c.entered != nil {
		close(c.entered)
		<-c.gate
	}
	r := &result.RPCResult{}
	if c.IsDestroyed() {
		r.SetError(errors.New("invoker-destroyed"))
	} else {
		r.SetResult(c.label)
	}
	return r
}
func publicationURL(t *testing.T) *common.URL {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	url, err := common.NewURL("tri://"+address+"/codec.Publication", common.WithParamsValue("interface", "codec.Publication"), common.WithParamsValue("serialization", "publication-json"))
	require.NoError(t, err)
	url.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{Methods: []common.MethodInfo{{Name: "Echo", Type: constant.CallUnary, ReqInitFunc: func() any { return []any{new(string)} }}}})
	return url
}
func publicationClient(url *common.URL) *tri.Client {
	opts, _ := tri.ConfiguredSerializationOptions(&publicationCodec{exportGenerationCodec{closes: &atomic.Int64{}}}, nil, nil, true, false)
	return tri.NewClient(http.DefaultClient, "http://"+url.Location+"/codec.Publication", opts...)
}
func publicationCall(cli *tri.Client) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var reply string
	err := cli.CallUnary(ctx, tri.NewRequest([]any{"hello"}), "Echo", tri.NewResponse(&reply))
	return reply, err
}
func TestConcurrentExportsPublishMatchingGenerations(t *testing.T) {
	var entered, gate chan struct{}
	var made, closed atomic.Int64
	require.NoError(t, extension.RegisterSerialization(serialization.Registration{Protocol: "triple", Name: "publication-json", Mode: serialization.WrappedObjectMode, WireName: "proto", New: func(o serialization.Options) (serialization.Codec, error) {
		if strings.HasPrefix(o.Parameters["epoch"], "slow") {
			close(entered)
			<-gate
		}
		made.Add(1)
		codec := &publicationCodec{exportGenerationCodec{closes: &closed}}
		if o.Parameters["epoch"] == "slow-factory-failure" {
			return codec, errors.New("factory failed")
		}
		return codec, nil
	}}))
	for _, epoch := range []string{"slow-success", "slow-factory-failure", "slow-transport-failure"} {
		t.Run(epoch, func(t *testing.T) {
			tp := NewTripleProtocol()
			defer tp.Destroy()
			entered, gate = make(chan struct{}), make(chan struct{})
			var release sync.Once
			defer release.Do(func() { close(gate) })
			url := publicationURL(t)
			slow, fast := url.Clone(), url.Clone()
			slow.SetParam("epoch", epoch)
			if epoch == "slow-transport-failure" {
				slow.SetAttribute(constant.TripleConfigKey, &global.TripleConfig{Http3: &global.Http3Config{Enable: true}})
			}
			slowInvoker := &publicationInvoker{BaseInvoker: base.NewBaseInvoker(slow), label: "slow"}
			fastInvoker := &publicationInvoker{BaseInvoker: base.NewBaseInvoker(fast), label: "fast"}
			type outcome struct {
				exporter   base.Exporter
				panicValue any
			}
			done := make(chan outcome, 1)
			go func() {
				out := outcome{}
				defer func() { out.panicValue = recover(); done <- out }()
				out.exporter = tp.Export(slowInvoker)
			}()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("slow factory did not start")
			}
			// A blocked user factory must neither publish an exporter nor hold the
			// protocol lock needed by another export.
			_, pending := tp.ExporterMap().Load(url.ServiceKey())
			require.False(t, pending)
			fastDone := make(chan base.Exporter, 1)
			go func() { fastDone <- tp.Export(fastInvoker) }()
			var fastExporter base.Exporter
			select {
			case fastExporter = <-fastDone:
			case <-time.After(5 * time.Second):
				t.Fatal("fast export blocked behind slow factory")
			}
			defer fastExporter.UnExport()
			waitExportListener(t, url.Location)
			cli := publicationClient(url)
			reply, err := publicationCall(cli)
			require.NoError(t, err)
			require.Equal(t, "fast", reply)
			release.Do(func() { close(gate) })
			var out outcome
			select {
			case out = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("slow export did not finish")
			}
			if epoch == "slow-success" {
				require.Nil(t, out.panicValue)
				defer out.exporter.UnExport()
				current, ok := tp.ExporterMap().Load(url.ServiceKey())
				require.True(t, ok)
				require.Same(t, out.exporter, current)
				fastExporter.UnExport()
				reply, err = publicationCall(cli)
				require.NoError(t, err)
				require.Equal(t, "slow", reply)
				out.exporter.UnExport()
				require.True(t, slowInvoker.IsDestroyed())
			} else {
				require.NotNil(t, out.panicValue)
				require.Nil(t, out.exporter)
				require.True(t, slowInvoker.IsDestroyed())
				current, ok := tp.ExporterMap().Load(url.ServiceKey())
				require.True(t, ok)
				require.Same(t, fastExporter, current)
				reply, err = publicationCall(cli)
				require.NoError(t, err)
				require.Equal(t, "fast", reply)
				fastExporter.UnExport()
			}
			_, tracked := tp.ExporterMap().Load(url.ServiceKey())
			require.False(t, tracked)
			_, err = publicationCall(cli)
			require.Error(t, err, "retiring the tracked generation must remove its handler")
			require.True(t, fastInvoker.IsDestroyed())
		})
	}
	require.Equal(t, made.Load(), closed.Load(), "all constructed generations must be released")
}

func TestRetirementDoesNotHoldPublicationLockWhileDraining(t *testing.T) {
	// Built-in Hessian2 has no closer; retirement must still drain the call
	// without blocking publication of a replacement.
	tp := NewTripleProtocol()
	defer tp.Destroy()
	url := publicationURL(t)
	url.SetParam("serialization", "hessian2")
	entered, gate := make(chan struct{}), make(chan struct{})
	var release sync.Once
	defer release.Do(func() { close(gate) })
	oldInvoker := &publicationInvoker{BaseInvoker: base.NewBaseInvoker(url), label: "old", entered: entered, gate: gate}
	old := tp.Export(oldInvoker)
	waitExportListener(t, url.Location)
	codec := tri.BuiltinSerializationCodecs()["hessian2"]
	opts, _ := tri.ConfiguredSerializationOptions(codec, nil, nil, true, false)
	cli := tri.NewClient(http.DefaultClient, "http://"+url.Location+"/codec.Publication", opts...)
	result := make(chan error, 1)
	go func() { _, err := publicationCall(cli); result <- err }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not enter")
	}
	retired := make(chan struct{})
	go func() { old.UnExport(); close(retired) }()
	raw, _ := url.GetAttribute("triple.serialization.resources.provider")
	owner := raw.(*serialization.Resources)
	require.Eventually(t, func() bool {
		if owner.Begin() {
			owner.End()
			return false
		}
		return true
	}, 3*time.Second, time.Millisecond)
	nextURL := url.Clone()
	nextInvoker := &publicationInvoker{BaseInvoker: base.NewBaseInvoker(nextURL), label: "next"}
	replacement := make(chan base.Exporter, 1)
	go func() { replacement <- tp.Export(nextInvoker) }()
	var next base.Exporter
	select {
	case next = <-replacement:
	case <-time.After(5 * time.Second):
		t.Fatal("replacement export blocked behind retiring request")
	}
	defer next.UnExport()
	reply, err := publicationCall(cli)
	require.NoError(t, err)
	require.Equal(t, "next", reply)
	release.Do(func() { close(gate) })
	require.NoError(t, <-result)
	select {
	case <-retired:
	case <-time.After(5 * time.Second):
		t.Fatal("retirement did not finish")
	}
	require.True(t, oldInvoker.IsDestroyed())
	require.False(t, nextInvoker.IsDestroyed())
	current, ok := tp.ExporterMap().Load(nextURL.ServiceKey())
	require.True(t, ok)
	require.Same(t, next, current)
	reply, err = publicationCall(cli)
	require.NoError(t, err)
	require.Equal(t, "next", reply)
}
