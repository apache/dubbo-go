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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const eventType = constant.MetricsConfigCenter

var ch = make(chan metrics.MetricsEvent, 10)
var info = metrics.NewMetricKey("dubbo_configcenter_total", "Config Changed Total")

func init() {
	metrics.AddCollector("config_center", func(mr metrics.MetricRegistry, url *common.URL) {
		if url.GetParamBool(constant.ConfigCenterEnabledKey, true) {
			c := &configCenterCollector{r: mr}
			c.start()
		}
	})
}

type configCenterCollector struct {
	r metrics.MetricRegistry
}

func (c *configCenterCollector) start() {
	metrics.Subscribe(eventType, ch)
	go func() {
		for e := range ch {
			if event, ok := e.(*ConfigCenterMetricEvent); ok {
				c.handleDataChange(event)
			}
		}
	}()
}

func (c *configCenterCollector) handleDataChange(event *ConfigCenterMetricEvent) {
	id := metrics.NewMetricId(info, metrics.NewConfigCenterLevel(event.key, event.group, event.configCenter, event.getChangeType()))
	c.r.Counter(id).Add(event.size)
}

const (
	Nacos     = "nacos"
	Apollo    = "apollo"
	Zookeeper = "zookeeper"
)

type ConfigCenterMetricEvent struct {
	// Name  MetricName
	key          string
	group        string
	configCenter string
	changeType   remoting.EventType
	size         float64
}

func (e *ConfigCenterMetricEvent) getChangeType() string {
	switch e.changeType {
	case remoting.EventTypeAdd:
		return "added"
	case remoting.EventTypeDel:
		return "deleted"
	case remoting.EventTypeUpdate:
		return "modified"
	default:
		return ""
	}
}

func (*ConfigCenterMetricEvent) Type() string {
	return eventType
}

func NewIncMetricEvent(key, group string, changeType remoting.EventType, c string) *ConfigCenterMetricEvent {
	return &ConfigCenterMetricEvent{key: key, group: group, changeType: changeType, configCenter: c, size: 1}
}
