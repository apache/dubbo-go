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

package adasvc

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/cluster/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"fmt"
	"github.com/pkg/errors"
	"math/rand"
	"time"
)

var ErrUnsupportedMetricsType = errors.New("unsupported metrics type")

type clusterInvoker struct {
	base.ClusterInvoker
}

func NewClusterInvoker(directory directory.Directory) protocol.Invoker {
	return &clusterInvoker{
		ClusterInvoker: base.NewClusterInvoker(directory),
	}
}

func (ivk *clusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	ivks := ivk.Directory.List(invocation)
	if len(ivks) < 2 {
		return ivks[0].Invoke(ctx, invocation)
	}
	// picks two nodes randomly
	var i, j int
	if len(ivks) == 2 {
		i, j = 0, 1
	} else {
		rand.Seed(time.Now().Unix())
		i = rand.Intn(len(ivks))
		j = i
		for i == j {
			j = rand.Intn(len(ivks))
		}
	}
	// compare which node is better
	m := metrics.LocalMetrics
	// TODO(justxuewei): please consider get the real method name from $invoke,
	// 	see also [#1511](https://github.com/apache/dubbo-go/issues/1511)
	methodName := invocation.MethodName()
	// viInterface, vjInterface means vegas latency of node i and node j
	// If one of the metrics is empty, invoke the invocation to that node directly.
	viInterface, err := m.GetMethodMetrics(ivks[i].GetURL(), methodName, "vegas")
	if err != nil {
		if errors.Is(err, metrics.ErrMetricsNotFound) {
			return ivks[i].Invoke(ctx, invocation)
		}
		return &protocol.RPCResult{
			Err: fmt.Errorf("get method metrics err: %w", err),
		}
	}

	vjInterface, err := m.GetMethodMetrics(ivks[j].GetURL(), methodName, "vegas")
	if err != nil {
		if errors.Is(err, metrics.ErrMetricsNotFound) {
			return ivks[j].Invoke(ctx, invocation)
		}
		return &protocol.RPCResult{
			Err: fmt.Errorf("get method metrics err: %w", err),
		}
	}

	// Convert interface to int, if the type is unexpected, return an error immediately
	vi, ok := viInterface.(int)
	if !ok {
		return &protocol.RPCResult{
			Err: ErrUnsupportedMetricsType,
		}
	}

	vj, ok := vjInterface.(int)
	if !ok {
		return &protocol.RPCResult{
			Err: ErrUnsupportedMetricsType,
		}
	}

	// For the latency time, the smaller, the better.
	if vi < vj {
		return ivks[i].Invoke(ctx, invocation)
	}

	return ivks[j].Invoke(ctx, invocation)
}
