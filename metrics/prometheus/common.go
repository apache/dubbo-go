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
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/prometheus/client_golang/prometheus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// isProvider shows whether this url represents the application received the request as server
func isProvider(url *common.URL) bool {
	role := url.GetParam(constant.RegistryRoleKey, "")
	return strings.EqualFold(role, strconv.Itoa(common.PROVIDER))
}

// isConsumer shows whether this url represents the application sent then request as client
func isConsumer(url *common.URL) bool {
	role := url.GetParam(constant.RegistryRoleKey, "")
	return strings.EqualFold(role, strconv.Itoa(common.CONSUMER))
}

func newHistogramVec(name, namespace string, labels []string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      name,
			Buckets:   defaultHistogramBucket,
		},
		labels)
}

func newCounter(name, namespace string) prometheus.Counter {
	return prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      name,
		})
}

func newCounterVec(name, namespace string, labels []string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:      name,
			Namespace: namespace,
		}, labels)
}

func newGauge(name, namespace string) prometheus.Gauge {
	return prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      name,
			Namespace: namespace,
		})
}

func newGaugeVec(name, namespace string, labels []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      name,
			Namespace: namespace,
		}, labels)
}

func newSummary(name, namespace string) prometheus.Summary {
	return prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name:      name,
			Namespace: namespace,
		})
}

// newSummaryVec create SummaryVec, the Namespace is dubbo
// the objectives is from my experience.
func newSummaryVec(name, namespace string, labels []string, maxAge int64) *prometheus.SummaryVec {
	return prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: namespace,
			Name:      name,
			Objectives: map[float64]float64{
				0.5:   0.01,
				0.75:  0.01,
				0.90:  0.005,
				0.98:  0.002,
				0.99:  0.001,
				0.999: 0.0001,
			},
			MaxAge: time.Duration(maxAge),
		},
		labels,
	)
}
