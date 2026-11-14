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

package client

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

type codecActionInvoker struct {
	*base.BaseInvoker
	destroys int
}

func (i *codecActionInvoker) Destroy() { i.destroys++; i.BaseInvoker.Destroy() }

type codecActionOwner struct{ closes int }
type codecActionProtocol struct {
	owners                 []*codecActionOwner
	prepared, released     []*common.URL
	sides                  []string
	referred               []*codecActionInvoker
	failPrepare, failRefer int
	sentinel               error
}

func (p *codecActionProtocol) PrepareCodec(u *common.URL, side string) error {
	p.sides = append(p.sides, side)
	if p.failPrepare > 0 && len(p.sides) == p.failPrepare {
		return p.sentinel
	}
	owner := &codecActionOwner{}
	p.owners = append(p.owners, owner)
	u.SetAttribute("codec.action.owner", owner)
	p.prepared = append(p.prepared, u)
	return nil
}
func (p *codecActionProtocol) ReleaseCodec(u *common.URL, side string) error {
	p.released = append(p.released, u)
	if raw, ok := u.GetAttribute("codec.action.owner"); ok {
		raw.(*codecActionOwner).closes++
	}
	if side != "consumer" {
		panic("wrong side")
	}
	return nil
}
func (p *codecActionProtocol) Refer(u *common.URL) base.Invoker {
	if p.failRefer > 0 && len(p.referred)+1 == p.failRefer {
		return nil
	}
	i := &codecActionInvoker{BaseInvoker: base.NewBaseInvoker(u)}
	p.referred = append(p.referred, i)
	return i
}
func (*codecActionProtocol) Export(base.Invoker) base.Exporter { return nil }
func (*codecActionProtocol) Destroy()                          {}
func TestCodecReferenceRollback(t *testing.T) {
	for _, stage := range []string{"prepare", "refer", "registry"} {
		t.Run(stage, func(t *testing.T) {
			name := "codec-action-" + stage
			p := &codecActionProtocol{sentinel: errors.New("prepare failed")}
			extension.SetProtocol(name, func() base.Protocol { return p })
			t.Cleanup(func() { extension.UnregisterProtocol(name) })
			opts := defaultReferenceOptions()
			opts.Consumer = global.DefaultConsumerConfig()
			opts.Reference.ProtocolClientConfig = global.DefaultClientProtocolConfig()
			opts.Reference.Protocol = name
			opts.Reference.InterfaceName = "codec.Action"
			opts.Reference.Filter = "-default"
			if stage == "registry" {
				had := slices.Contains(extension.GetAllProtocolNames(), "registry")
				var old base.Protocol
				if had {
					old = extension.GetProtocol("registry")
				}
				extension.SetProtocol("registry", func() base.Protocol { return p })
				t.Cleanup(func() {
					if had {
						extension.SetProtocol("registry", func() base.Protocol { return old })
					} else {
						extension.UnregisterProtocol("registry")
					}
				})
				opts.Reference.URL = "registry://127.0.0.1:2181"
				p.failRefer = 1
			} else {
				opts.Reference.URL = fmt.Sprintf("%s://127.0.0.1:1;%s://127.0.0.1:2", name, name)
			}
			switch stage {
			case "prepare":
				p.failPrepare = 2
			case "refer":
				p.failRefer = 2
			}
			err := opts.refer(nil, nil)
			require.Error(t, err)
			if stage == "prepare" {
				require.ErrorIs(t, err, p.sentinel)
			}
			require.Nil(t, opts.invoker)
			require.Empty(t, opts.urls)
			require.Len(t, p.released, len(p.prepared))
			for _, side := range p.sides {
				require.Equal(t, "consumer", side)
			}
			for i, u := range p.prepared {
				require.Equal(t, name, u.Protocol)
				require.Equal(t, "codec.Action", strings.TrimPrefix(u.Path, "/"))
				require.True(t, u.URLEqual(p.released[len(p.released)-1-i]))
			}
			for _, owner := range p.owners {
				require.Equal(t, 1, owner.closes, "prepared codec leaked")
			}
			for _, i := range p.referred {
				require.Equal(t, 1, i.destroys)
			}
			p.failPrepare, p.failRefer = 0, 0
			// One URL avoids a cluster dependency while proving the same options can retry.
			opts.Reference.URL = name + "://127.0.0.1:3"
			opts.Reference.Cluster = "codec-action-cluster"
			extension.SetCluster("codec-action-cluster", func() cluster.Cluster { return timeoutCaptureCluster{} })
			require.NoError(t, opts.refer(nil, nil))
			require.NotNil(t, opts.invoker)
			require.Len(t, opts.urls, 1)
			snapshot := opts.urls[0].Clone()
			opts.invoker.Destroy()
			require.NoError(t, extension.ReleaseCodecURL(snapshot, "consumer"))
		})
	}
}
func TestCodecLegacyReferenceInitialization(t *testing.T) {
	const name = "codec-legacy-action"
	p := &timeoutCaptureProtocol{}
	extension.SetProtocol(name, func() base.Protocol { return p })
	t.Cleanup(func() { extension.UnregisterProtocol(name) })
	opts := defaultReferenceOptions()
	opts.Consumer = global.DefaultConsumerConfig()
	opts.Reference.ProtocolClientConfig = global.DefaultClientProtocolConfig()
	opts.Reference.Protocol = name
	opts.Reference.InterfaceName = "codec.Legacy"
	opts.Reference.URL = name + "://127.0.0.1:1"
	opts.Reference.Cluster = "codec-action-cluster"
	opts.Reference.Filter = "-default"
	extension.SetCluster("codec-action-cluster", func() cluster.Cluster { return timeoutCaptureCluster{} })
	require.NoError(t, opts.refer(nil, nil))
	require.NotNil(t, opts.invoker)
	opts.invoker.Destroy()
}

type codecNilCluster struct{}

func (codecNilCluster) Join(directory.Directory) base.Invoker { return nil }
func TestCodecClusterAssemblyRollback(t *testing.T) {
	for _, stage := range []string{"missing-single", "missing-multiple", "nil-join"} {
		t.Run(stage, func(t *testing.T) {
			name := "codec-cluster-action-" + stage
			p := &codecActionProtocol{}
			extension.SetProtocol(name, func() base.Protocol { return p })
			t.Cleanup(func() { extension.UnregisterProtocol(name) })
			opts := defaultReferenceOptions()
			opts.Consumer = global.DefaultConsumerConfig()
			opts.Reference.InterfaceName = "codec.ClusterAssembly"
			opts.Reference.Protocol = name
			opts.Reference.Filter = "-default"
			opts.Reference.Cluster = "codec-assembly-not-registered"
			opts.Reference.URL = name + "://127.0.0.1:1"
			if stage == "missing-multiple" {
				opts.Reference.URL += ";" + name + "://127.0.0.1:2"
			}
			if stage == "nil-join" {
				opts.Reference.Cluster = "codec-assembly-nil-cluster"
				extension.SetCluster(opts.Reference.Cluster, func() cluster.Cluster { return codecNilCluster{} })
			}
			require.Error(t, opts.refer(nil, nil))
			require.Nil(t, opts.invoker)
			require.Empty(t, opts.urls)
			for _, created := range p.referred {
				require.Equal(t, 1, created.destroys)
			}
			for _, owner := range p.owners {
				require.Equal(t, 1, owner.closes)
			}
			require.Len(t, p.released, len(p.prepared))
		})
	}
}
