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

package prometheus

import (
	"sync"
)

import (
	"github.com/prometheus/client_golang/prometheus"
)

type syncMaps struct {
	userGauge      sync.Map
	userSummary    sync.Map
	userCounter    sync.Map
	userCounterVec sync.Map
	userGaugeVec   sync.Map
	userSummaryVec sync.Map
}

// setGauge set gauge to target value with given label, if label is not empty, set gauge vec
// if target gauge/gaugevec not exist, just create new gauge and set the value
func (reporter *PrometheusReporter) setGauge(gaugeName string, toSetValue float64, labelMap prometheus.Labels) {
	if len(labelMap) == 0 {
		// gauge
		if val, exist := reporter.userGauge.Load(gaugeName); !exist {
			gauge := newGauge(gaugeName, reporter.namespace)
			err := prometheus.DefaultRegisterer.Register(gauge)
			if err == nil {
				reporter.userGauge.Store(gaugeName, gauge)
				gauge.Set(toSetValue)
			} else if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				// A gauge for that metric has been registered before.
				// Use the old gauge from now on.
				are.ExistingCollector.(prometheus.Gauge).Set(toSetValue)
			}

		} else {
			val.(prometheus.Gauge).Set(toSetValue)
		}
		return
	}

	// gauge vec
	if val, exist := reporter.userGaugeVec.Load(gaugeName); !exist {
		keyList := make([]string, 0)
		for k := range labelMap {
			keyList = append(keyList, k)
		}
		gaugeVec := newGaugeVec(gaugeName, reporter.namespace, keyList)
		err := prometheus.DefaultRegisterer.Register(gaugeVec)
		if err == nil {
			reporter.userGaugeVec.Store(gaugeName, gaugeVec)
			gaugeVec.With(labelMap).Set(toSetValue)
		} else if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// A gauge for that metric has been registered before.
			// Use the old gauge from now on.
			are.ExistingCollector.(*prometheus.GaugeVec).With(labelMap).Set(toSetValue)
		}
	} else {
		val.(*prometheus.GaugeVec).With(labelMap).Set(toSetValue)
	}
}

// incCounter inc counter to inc if label is not empty, set counter vec
// if target counter/counterVec not exist, just create new counter and inc the value
func (reporter *PrometheusReporter) incCounter(counterName string, labelMap prometheus.Labels) {
	if len(labelMap) == 0 {
		// counter
		if val, exist := reporter.userCounter.Load(counterName); !exist {
			counter := newCounter(counterName, reporter.namespace)
			err := prometheus.DefaultRegisterer.Register(counter)
			if err == nil {
				reporter.userCounter.Store(counterName, counter)
				counter.Inc()
			} else if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				// A counter for that metric has been registered before.
				// Use the old counter from now on.
				are.ExistingCollector.(prometheus.Counter).Inc()
			}
		} else {
			val.(prometheus.Counter).Inc()
		}
		return
	}

	// counter vec inc
	if val, exist := reporter.userCounterVec.Load(counterName); !exist {
		keyList := make([]string, 0)
		for k := range labelMap {
			keyList = append(keyList, k)
		}
		counterVec := newCounterVec(counterName, reporter.namespace, keyList)
		err := prometheus.DefaultRegisterer.Register(counterVec)
		if err == nil {
			reporter.userCounterVec.Store(counterName, counterVec)
			counterVec.With(labelMap).Inc()
		} else if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// A counter for that metric has been registered before.
			// Use the old counter from now on.
			are.ExistingCollector.(*prometheus.CounterVec).With(labelMap).Inc()
		}
	} else {
		val.(*prometheus.CounterVec).With(labelMap).Inc()
	}
}

// incSummary inc summary to target value with given label, if label is not empty, set summary vec
// if target summary/summaryVec not exist, just create new summary and set the value
func (reporter *PrometheusReporter) incSummary(summaryName string, toSetValue float64, labelMap prometheus.Labels) {
	if len(labelMap) == 0 {
		// summary
		if val, exist := reporter.userSummary.Load(summaryName); !exist {
			summary := newSummary(summaryName, reporter.namespace)
			err := prometheus.DefaultRegisterer.Register(summary)
			if err == nil {
				reporter.userSummary.Store(summaryName, summary)
				summary.Observe(toSetValue)
			} else if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				// A summary for that metric has been registered before.
				// Use the old summary from now on.
				are.ExistingCollector.(prometheus.Summary).Observe(toSetValue)
			}
		} else {
			val.(prometheus.Summary).Observe(toSetValue)
		}
		return
	}

	// summary vec
	if val, exist := reporter.userSummaryVec.Load(summaryName); !exist {
		keyList := make([]string, 0)
		for k := range labelMap {
			keyList = append(keyList, k)
		}
		summaryVec := newSummaryVec(summaryName, reporter.namespace, keyList, reporter.reporterConfig.SummaryMaxAge)
		err := prometheus.DefaultRegisterer.Register(summaryVec)
		if err == nil {
			reporter.userSummaryVec.Store(summaryName, summaryVec)
			summaryVec.With(labelMap).Observe(toSetValue)
		} else if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// A summary for that metric has been registered before.
			// Use the old summary from now on.
			are.ExistingCollector.(*prometheus.SummaryVec).With(labelMap).Observe(toSetValue)
		}
	} else {
		val.(*prometheus.SummaryVec).With(labelMap).Observe(toSetValue)
	}
}

func SetGaugeWithLabel(gaugeName string, val float64, label prometheus.Labels) {
	if reporterInstance.reporterConfig.Enable {
		reporterInstance.setGauge(gaugeName, val, label)
	}
}

func SetGauge(gaugeName string, val float64) {
	if reporterInstance.reporterConfig.Enable {
		reporterInstance.setGauge(gaugeName, val, make(prometheus.Labels))
	}
}

func IncCounterWithLabel(counterName string, label prometheus.Labels) {
	if reporterInstance.reporterConfig.Enable {
		reporterInstance.incCounter(counterName, label)
	}
}

func IncCounter(summaryName string) {
	if reporterInstance.reporterConfig.Enable {
		reporterInstance.incCounter(summaryName, make(prometheus.Labels))
	}
}

func IncSummaryWithLabel(counterName string, val float64, label prometheus.Labels) {
	if reporterInstance.reporterConfig.Enable {
		reporterInstance.incSummary(counterName, val, label)
	}
}

func IncSummary(summaryName string, val float64) {
	if reporterInstance.reporterConfig.Enable {
		reporterInstance.incSummary(summaryName, val, make(prometheus.Labels))
	}
}
