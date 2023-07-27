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

package registry

import (
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

// RegistryMetricsEvent contains info about register metrics
type RegistryMetricsEvent struct {
	Name       MetricName
	Succ       bool
	Start      time.Time
	End        time.Time
	Attachment map[string]string
}

func (r RegistryMetricsEvent) Type() string {
	return constant.MetricsRegistry
}

func (r *RegistryMetricsEvent) CostMs() float64 {
	return float64(r.End.Sub(r.Start)) / float64(time.Millisecond)
}

// NewRegisterEvent for register metrics
func NewRegisterEvent(succ bool, start time.Time) metrics.MetricsEvent {
	return &RegistryMetricsEvent{
		Name:  Reg,
		Succ:  succ,
		Start: start,
		End:   time.Now(),
	}
}

// NewSubscribeEvent for subscribe metrics
func NewSubscribeEvent(succ bool) metrics.MetricsEvent {
	return &RegistryMetricsEvent{
		Name: Sub,
		Succ: succ,
	}
}

// NewNotifyEvent for notify metrics
func NewNotifyEvent(start time.Time) metrics.MetricsEvent {
	return &RegistryMetricsEvent{
		Name:  Notify,
		Start: start,
		End:   time.Now(),
	}
}

// NewDirectoryEvent for directory metrics
func NewDirectoryEvent(dirTyp string) metrics.MetricsEvent {
	return &RegistryMetricsEvent{
		Name:       Directory,
		Attachment: map[string]string{"DirTyp": dirTyp},
	}
}

// NewServerRegisterEvent for server register metrics
func NewServerRegisterEvent(succ bool, start time.Time) metrics.MetricsEvent {
	return &RegistryMetricsEvent{
		Name:  ServerReg,
		Succ:  succ,
		Start: start,
		End:   time.Now(),
	}
}

// NewServerSubscribeEvent for server subscribe metrics
func NewServerSubscribeEvent(succ bool) metrics.MetricsEvent {
	return &RegistryMetricsEvent{
		Name: ServerSub,
		Succ: succ,
	}
}
