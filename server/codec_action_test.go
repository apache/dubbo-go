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

package server

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

type CodecActionService struct{}

func (*CodecActionService) Reference() string                            { return "codec.Action" }
func (*CodecActionService) Echo(context.Context, string) (string, error) { return "ok", nil }

type codecActionExporter struct {
	inv    base.Invoker
	closed int
}

func (e *codecActionExporter) GetInvoker() base.Invoker { return e.inv }
func (e *codecActionExporter) UnExport()                { e.closed++; e.inv.Destroy() }

type codecServiceOwner struct{ closes int }
type codecExportProtocol struct {
	owners                            []*codecServiceOwner
	prepared, released                []*common.URL
	exporters                         []*codecActionExporter
	sides                             []string
	failPrepare, failExport, disabled bool
	sentinel                          error
}

func (p *codecExportProtocol) PrepareCodec(u *common.URL, side string) error {
	p.sides = append(p.sides, side)
	if p.failPrepare {
		return p.sentinel
	}
	owner := &codecServiceOwner{}
	p.owners = append(p.owners, owner)
	u.SetAttribute("codec.action.owner", owner)
	p.prepared = append(p.prepared, u)
	if p.disabled {
		u.SetParam("export", "false")
	}
	return nil
}
func (p *codecExportProtocol) ReleaseCodec(u *common.URL, side string) error {
	if side != "provider" {
		panic("wrong side")
	}
	p.released = append(p.released, u)
	if raw, ok := u.GetAttribute("codec.action.owner"); ok {
		raw.(*codecServiceOwner).closes++
	}
	return nil
}
func (p *codecExportProtocol) Export(i base.Invoker) base.Exporter {
	if p.failExport {
		return nil
	}
	e := &codecActionExporter{inv: i}
	p.exporters = append(p.exporters, e)
	return e
}
func (*codecExportProtocol) Refer(*common.URL) base.Invoker { return nil }
func (*codecExportProtocol) Destroy()                       {}
func TestCodecServiceRollback(t *testing.T) {
	for _, stage := range []string{"prepare", "export", "disabled"} {
		t.Run(stage, func(t *testing.T) {
			a, b := &codecExportProtocol{}, &codecExportProtocol{sentinel: errors.New("provider prepare failed")}
			names := []string{"codec-export-a-" + stage, "codec-export-b-" + stage}
			for i, p := range []*codecExportProtocol{a, b} {
				name := names[i]
				extension.SetProtocol(name, func() base.Protocol { return p })
				t.Cleanup(func() { extension.UnregisterProtocol(name) })
			}
			// A fresh wrapper resolves each invoker's protocol; no network ports are used.
			srv := &Server{cfg: defaultServerOptions()}
			opts := defaultServiceOptions()
			opts.srvOpts = srv.cfg
			opts.rpcService = &CodecActionService{}
			opts.Provider = global.DefaultProviderConfig()
			opts.Protocols = map[string]*global.ProtocolConfig{}
			opts.Service.Interface = "codec.Action." + stage
			opts.Service.NotRegister = true
			opts.Service.Filter = "-default"
			opts.Service.ProtocolIDs = names
			running := "codec.AlreadyRunning." + stage
			for _, name := range names {
				_, err := common.ServiceMap.Register(running, name, "", "", &CodecActionService{})
				require.NoError(t, err)
				t.Cleanup(func() {
					_ = common.ServiceMap.UnRegister(running, name, common.ServiceKey(running, "", ""))
					_ = common.ServiceMap.UnRegister(opts.Service.Interface, name, common.ServiceKey(opts.Service.Interface, "", ""))
				})
			}
			for _, name := range names {
				opts.Protocols[name] = &global.ProtocolConfig{Name: name, Ip: "127.0.0.1", Port: "1"}
			}
			existing := &codecActionExporter{inv: base.NewBaseInvoker(common.NewURLWithOptions())}
			opts.exporters = []base.Exporter{existing}
			if stage == "prepare" {
				b.failPrepare = true
			} else if stage == "export" {
				b.failExport = true
			} else {
				b.disabled = true
			}
			err := opts.Export()
			if stage == "disabled" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.False(t, opts.IsExport())
			require.Equal(t, []base.Exporter{existing}, opts.exporters)
			require.Zero(t, existing.closed)
			require.Len(t, a.exporters, 1)
			require.Equal(t, 1, a.exporters[0].closed)
			for _, p := range []*codecExportProtocol{a, b} {
				require.Len(t, p.released, len(p.prepared))
				for _, owner := range p.owners {
					require.Equal(t, 1, owner.closes, "prepared codec leaked")
				}
				for _, side := range p.sides {
					require.Equal(t, "provider", side)
				}
			}
			for _, name := range names {
				require.Nil(t, common.ServiceMap.GetService(name, opts.Service.Interface, "", ""))
				require.NotNil(t, common.ServiceMap.GetService(name, running, "", ""))
			}
			b.failPrepare, b.failExport, b.disabled = false, false, false
			require.NoError(t, opts.Export())
			require.True(t, opts.IsExport())
			require.Len(t, opts.exporters, 3)
			snapshots := []*common.URL{a.prepared[len(a.prepared)-1].Clone(), b.prepared[len(b.prepared)-1].Clone()}
			opts.Unexport()
			for i, snapshot := range snapshots {
				p := a
				if i == 1 {
					p = b
				}
				_ = p.ReleaseCodec(snapshot, "provider")
			}
			require.Equal(t, 1, existing.closed)
			for _, name := range names {
				_ = common.ServiceMap.UnRegister(opts.Service.Interface, name, common.ServiceKey(opts.Service.Interface, "", ""))
			}
		})
	}
}
