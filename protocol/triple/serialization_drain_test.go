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
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
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
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type noCloseDrainCodec struct {
	phase         string
	entered, gate chan struct{}
}

func (*noCloseDrainCodec) Name() string { return "no-close-drain" }
func (c *noCloseDrainCodec) block(phase string) {
	if c.phase == phase {
		close(c.entered)
		<-c.gate
	}
}
func (c *noCloseDrainCodec) Marshal(v any) ([]byte, error) { c.block("encode"); return json.Marshal(v) }
func (c *noCloseDrainCodec) Unmarshal(b []byte, v any) error {
	c.block("decode")
	return json.Unmarshal(b, v)
}

type noCloseDrainInvoker struct {
	exportGenerationInvoker
	codec *noCloseDrainCodec
}

func (c *noCloseDrainInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	c.codec.block("invoke")
	return c.exportGenerationInvoker.Invoke(ctx, inv)
}

func TestServiceRetirementDrainsCodecsWithoutClosers(t *testing.T) {
	var active *noCloseDrainCodec
	require.NoError(t, extension.RegisterSerialization(serialization.Registration{Protocol: "triple", Name: "no-close-drain", Mode: serialization.WrappedObjectMode, WireName: "proto", New: func(serialization.Options) (serialization.Codec, error) { return active, nil }}))
	for _, wire := range []string{"grpc", "triple"} {
		for _, phase := range []string{"decode", "invoke", "encode"} {
			t.Run(wire+"/"+phase, func(t *testing.T) {
				tp := NewTripleProtocol()
				defer tp.Destroy()
				entered, gate := make(chan struct{}), make(chan struct{})
				var release sync.Once
				defer release.Do(func() { close(gate) })
				active = &noCloseDrainCodec{phase: phase, entered: entered, gate: gate}
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				address := listener.Addr().String()
				require.NoError(t, listener.Close())
				url, err := common.NewURL("tri://"+address+"/codec.NoCloseDrain", common.WithParamsValue("interface", "codec.NoCloseDrain"), common.WithParamsValue("serialization", "no-close-drain"))
				require.NoError(t, err)
				url.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{Methods: []common.MethodInfo{{Name: "Echo", Type: constant.CallUnary, ReqInitFunc: func() any { return []any{new(string)} }}}})
				inv := &noCloseDrainInvoker{exportGenerationInvoker: exportGenerationInvoker{base.NewBaseInvoker(url)}, codec: active}
				exporter := tp.Export(inv)
				waitExportListener(t, address)
				opts, _ := tri.ConfiguredSerializationOptions(&noCloseDrainCodec{}, nil, nil, true, false)
				if wire == "triple" {
					opts = append(opts, tri.WithTriple())
				}
				cli := tri.NewClient(http.DefaultClient, "http://"+address+"/codec.NoCloseDrain", opts...)
				call := func() error {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					var reply string
					err := cli.CallUnary(ctx, tri.NewRequest([]any{"request"}), "Echo", tri.NewResponse(&reply))
					if err == nil && reply != "hello" {
						return fmt.Errorf("unexpected reply %q", reply)
					}
					return err
				}
				outcome := make(chan error, 1)
				go func() { outcome <- call() }()
				select {
				case <-entered:
				case err := <-outcome:
					t.Fatalf("RPC failed before blocked phase: %v", err)
				case <-time.After(5 * time.Second):
					t.Fatal("RPC did not reach blocked phase")
				}
				done := make(chan struct{})
				go func() { exporter.UnExport(); close(done) }()
				raw, _ := url.GetAttribute("triple.serialization.resources.provider")
				owner := raw.(*serialization.Resources)
				require.False(t, owner.HasClosers())
				require.Eventually(t, func() bool {
					if owner.Begin() {
						owner.End()
						return false
					}
					return true
				}, 3*time.Second, time.Millisecond)
				require.False(t, inv.IsDestroyed(), "invoker must remain alive while the handler is active")
				select {
				case <-done:
					t.Fatal("Unexport returned before active handler completed")
				default:
				}
				require.Error(t, call(), "retired service must reject new calls")
				release.Do(func() { close(gate) })
				require.NoError(t, <-outcome)
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Fatal("Unexport did not finish")
				}
				require.True(t, inv.IsDestroyed())
				exporter.UnExport()
			})
		}
	}
}
