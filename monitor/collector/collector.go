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

package collector

import (
	"github.com/apache/dubbo-go/metrics/impl"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

const (
	groupName = "dubbo"
)

type Exporter struct {
	//URI    string
	metrics map[string]*prometheus.Desc
	mutex   sync.RWMutex
}

func newGlobalMetric(namespace string, metricName string, docString string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(namespace+"_"+metricName, docString, labels, nil)
}

func NewExporter(namespace string) (*Exporter, error) {
	return &Exporter{
		metrics: map[string]*prometheus.Desc{
			"dubbogo_compass_metric": newGlobalMetric(namespace, "compass_metric", "The description of compass_metric", []string{"method_name"}),
		},
	}, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.metrics {
		ch <- m
	}
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	metricsManager := new(impl.DefaultMetricManager)
	metricsRegistry := metricsManager.GetMetricRegistry(groupName)

	for _, v := range metricsRegistry.GetMetrics() {
		compassImpl := metricsRegistry.GetCompass(v.MetricName)
		//compassImpl := metricsManager.GetCompass("dubbo", v.MetricName)
		snapshot := compassImpl.GetSnapshot()
		values, _ := snapshot.GetValues()
		sum := len(values)
		line50, _ := snapshot.GetMedian()
		line75, _ := snapshot.Get75thPercentile()
		line95, _ := snapshot.Get95thPercentile()
		line98, _ := snapshot.Get98thPercentile()
		line99, _ := snapshot.Get99thPercentile()
		ch <- prometheus.MustNewConstSummary(e.metrics["dubbogo_compass_metric"], uint64(compassImpl.GetCount()),
			float64(sum), map[float64]float64{float64(0.50): line50})
		ch <- prometheus.MustNewConstSummary(e.metrics["dubbogo_compass_metric"], uint64(compassImpl.GetCount()),
			float64(sum), map[float64]float64{float64(0.75): line75})
		ch <- prometheus.MustNewConstSummary(e.metrics["dubbogo_compass_metric"], uint64(compassImpl.GetCount()),
			float64(sum), map[float64]float64{float64(0.95): line95})
		ch <- prometheus.MustNewConstSummary(e.metrics["dubbogo_compass_metric"], uint64(compassImpl.GetCount()),
			float64(sum), map[float64]float64{float64(0.98): line98})
		ch <- prometheus.MustNewConstSummary(e.metrics["dubbogo_compass_metric"], uint64(compassImpl.GetCount()),
			float64(sum), map[float64]float64{float64(0.99): line99})
	}

}
