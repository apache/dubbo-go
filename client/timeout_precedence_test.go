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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

type timeoutCaptureProtocol struct{ captured *common.URL }

func (p *timeoutCaptureProtocol) Export(base.Invoker) base.Exporter { return nil }
func (p *timeoutCaptureProtocol) Destroy()                          {}
func (p *timeoutCaptureProtocol) Refer(u *common.URL) base.Invoker {
	p.captured = u
	return base.NewBaseInvoker(u)
}

type timeoutCaptureCluster struct{}

func (timeoutCaptureCluster) Join(d directory.Directory) base.Invoker {
	return base.NewBaseInvoker(d.GetURL())
}

func TestReferenceTimeoutPrecedence(t *testing.T) {
	tests := []struct {
		name string
		opts []ReferenceOption
		want string
	}{
		{"reference timeout wins over consumer", []ReferenceOption{WithRequestTimeout(500 * time.Millisecond)}, "500ms"},
		{"consumer timeout is the fallback", nil, "3s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &timeoutCaptureProtocol{}
			extension.SetProtocol("timeout-precedence", func() base.Protocol { return p })
			extension.SetCluster("timeout-precedence", func() cluster.Cluster { return timeoutCaptureCluster{} })

			opts := defaultReferenceOptions()
			opts.Consumer = global.DefaultConsumerConfig()
			opts.Consumer.RequestTimeout = "3s"
			for _, o := range tt.opts {
				o(opts)
			}
			opts.Reference.InterfaceName = "example.TimeoutService"
			opts.Reference.Protocol = "timeout-precedence"
			opts.Reference.Cluster = "timeout-precedence"
			opts.Reference.URL = "timeout-precedence://127.0.0.1:20000"
			opts.Reference.Filter = "-default"
			opts.Refer()

			require.NotNil(t, p.captured)
			require.Equal(t, tt.want, p.captured.GetParam(constant.TimeoutKey, ""))
		})
	}
}
