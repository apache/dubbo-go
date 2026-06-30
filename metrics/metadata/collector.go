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

package metadata

import (
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

const eventType = constant.MetricsMetadata

var ch = make(chan metrics.MetricsEvent, 10)

func init() {
	metrics.AddCollector("metadata", func(mr metrics.MetricRegistry, url *common.URL) {
		if url.GetParamBool(constant.MetadataEnabledKey, true) {
			l := &MetadataMetricCollector{metrics.BaseCollector{R: mr}}
			l.start()
		}
	})
}

type MetadataMetricCollector struct {
	metrics.BaseCollector
}

func (c *MetadataMetricCollector) start() {
	metrics.Subscribe(eventType, ch)
	go func() {
		for e := range ch {
			if event, ok := e.(*MetadataMetricEvent); ok {
				switch event.Name {
				case StoreProvider:
					c.handleStoreProvider(event)
				case MetadataPush:
					c.handleMetadataPush(event)
				case MetadataSub:
					c.handleMetadataSub(event)
				case SubscribeServiceRt:
					c.handleSubscribeService(event)
				case MetadataMappingRegister:
					c.handleMetadataMappingRegister(event)
				case MetadataMappingGet:
					c.handleMetadataMappingGet(event)
				case MetadataMappingRemove:
					c.handleMetadataMappingRemove(event)
				default:
				}
			}
		}
	}()
}

func (c *MetadataMetricCollector) handleMetadataPush(event *MetadataMetricEvent) {
	level := metrics.GetApplicationLevel()
	c.StateCount(metadataPushNum, metadataPushSucceed, metadataPushFailed, level, event.Succ)
	c.R.Rt(metrics.NewMetricId(pushRt, level), &metrics.RtOpts{}).Observe(event.CostMs())
}

func (c *MetadataMetricCollector) handleMetadataSub(event *MetadataMetricEvent) {
	level := metrics.GetApplicationLevel()
	c.StateCount(metadataSubNum, metadataSubSucceed, metadataSubFailed, level, event.Succ)
	c.R.Rt(metrics.NewMetricId(subscribeRt, level), &metrics.RtOpts{}).Observe(event.CostMs())
}

func (c *MetadataMetricCollector) handleStoreProvider(event *MetadataMetricEvent) {
	level := metrics.NewServiceMetric(event.Attachment[constant.InterfaceKey])
	c.StateCount(metadataStoreProviderNum, metadataStoreProviderSucceed, metadataStoreProviderFailed, level, event.Succ)
	c.R.Rt(metrics.NewMetricId(storeProviderInterfaceRt, level), &metrics.RtOpts{}).Observe(event.CostMs())
}

func (c *MetadataMetricCollector) handleSubscribeService(event *MetadataMetricEvent) {
	level := metrics.NewServiceMetric(event.Attachment[constant.InterfaceKey])
	c.R.Rt(metrics.NewMetricId(subscribeServiceRt, level), &metrics.RtOpts{}).Observe(event.CostMs())
}

func (c *MetadataMetricCollector) handleMetadataMappingRegister(event *MetadataMetricEvent) {
	level := newMetadataMappingMetricLevel(event.Attachment)
	c.StateCount(metadataMappingRegisterNum, metadataMappingRegisterSucceed, metadataMappingRegisterFailed, level, event.Succ)
	c.R.Rt(metrics.NewMetricId(metadataMappingRegisterRt, level), &metrics.RtOpts{}).Observe(event.CostMs())
}

func (c *MetadataMetricCollector) handleMetadataMappingGet(event *MetadataMetricEvent) {
	level := newMetadataMappingMetricLevel(event.Attachment)
	c.StateCount(metadataMappingGetNum, metadataMappingGetSucceed, metadataMappingGetFailed, level, event.Succ)
	c.R.Rt(metrics.NewMetricId(metadataMappingGetRt, level), &metrics.RtOpts{}).Observe(event.CostMs())
}

func (c *MetadataMetricCollector) handleMetadataMappingRemove(event *MetadataMetricEvent) {
	level := newMetadataMappingMetricLevel(event.Attachment)
	c.StateCount(metadataMappingRemoveNum, metadataMappingRemoveSucceed, metadataMappingRemoveFailed, level, event.Succ)
	c.R.Rt(metrics.NewMetricId(metadataMappingRemoveRt, level), &metrics.RtOpts{}).Observe(event.CostMs())
}

type MetadataMetricEvent struct {
	Name       MetricName
	Succ       bool
	Start      time.Time
	End        time.Time
	Attachment map[string]string
}

func (*MetadataMetricEvent) Type() string {
	return eventType
}

func (e *MetadataMetricEvent) CostMs() float64 {
	return float64(e.End.Sub(e.Start)) / float64(time.Millisecond)
}

func NewMetadataMetricTimeEvent(n MetricName) *MetadataMetricEvent {
	return &MetadataMetricEvent{Name: n, Start: time.Now(), Attachment: make(map[string]string)}
}

type metadataMappingMetricLevel struct {
	*metrics.ApplicationMetricLevel
	attachment map[string]string
}

func newMetadataMappingMetricLevel(attachment map[string]string) metadataMappingMetricLevel {
	return metadataMappingMetricLevel{
		ApplicationMetricLevel: metrics.GetApplicationLevel(),
		attachment:             attachment,
	}
}

func (m metadataMappingMetricLevel) Tags() map[string]string {
	tags := m.ApplicationMetricLevel.Tags()
	tags[constant.TagInterface] = m.attachment[constant.InterfaceKey]
	tags[constant.TagGroup] = m.attachment[constant.GroupKey]
	tags[constant.ApplicationKey] = m.attachment[constant.ApplicationKey]
	tags[MetadataMappingOperationKey] = m.attachment[MetadataMappingOperationKey]
	if listenerRequested, ok := m.attachment[MetadataMappingListenerRequestedKey]; ok {
		tags[MetadataMappingListenerRequestedKey] = listenerRequested
	}
	return tags
}
