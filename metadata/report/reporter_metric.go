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

package report

import (
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/metrics/metadata"
)

type PubMetricEventReport struct {
	MetadataReport
}

func NewPubMetricEventReport(r MetadataReport) MetadataReport {
	return &PubMetricEventReport{MetadataReport: r}
}

func (r *PubMetricEventReport) StoreProviderMetadata(i *identifier.MetadataIdentifier, s string) error {
	event := metadata.NewMetadataMetricTimeEvent(metadata.StoreProvider)
	err := r.MetadataReport.StoreProviderMetadata(i, s)
	event.Succ = err == nil
	event.End = time.Now()
	event.Attachment[constant.InterfaceKey] = i.ServiceInterface
	metrics.Publish(event)
	return err
}

func (r *PubMetricEventReport) GetAppMetadata(i *identifier.SubscriberMetadataIdentifier) (*common.MetadataInfo, error) {
	event := metadata.NewMetadataMetricTimeEvent(metadata.MetadataSub)
	info, err := r.MetadataReport.GetAppMetadata(i)
	event.Succ = err == nil
	event.End = time.Now()
	metrics.Publish(event)
	return info, err
}

func (r *PubMetricEventReport) PublishAppMetadata(i *identifier.SubscriberMetadataIdentifier, info *common.MetadataInfo) error {
	event := metadata.NewMetadataMetricTimeEvent(metadata.MetadataPush)
	err := r.MetadataReport.PublishAppMetadata(i, info)
	event.Succ = err == nil
	event.End = time.Now()
	metrics.Publish(event)
	return err
}
