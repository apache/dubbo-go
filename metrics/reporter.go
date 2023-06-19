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

package metrics

import (
	"context"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const DefMaxAge = 600000000000

type ReporterConfig struct {
	Enable             bool
	Namespace          string
	Mode               ReportMode
	Port               string
	Path               string
	PushGatewayAddress string
	SummaryMaxAge      int64
}

type ReportMode string

const (
	ReportModePull = "pull"
	ReportModePush = "push"
)

func NewReporterConfig() *ReporterConfig {
	return &ReporterConfig{
		Enable:             true,
		Namespace:          "dubbo",
		Port:               "9090",
		Path:               "/metrics",
		Mode:               ReportModePull,
		PushGatewayAddress: "",
		SummaryMaxAge:      DefMaxAge,
	}
}

// Reporter is the interface which will be used to report the invocation's duration
//
// Report method reports the duration of an invocation.
type Reporter interface {
	ReportAfterInvocation(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation,
		cost time.Duration, res protocol.Result)
	ReportBeforeInvocation(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation)
}
