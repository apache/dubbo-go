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
	"context"
	"dubbo.apache.org/dubbo-go/v3/config"
	"github.com/prometheus/client_golang/prometheus/push"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/emirpasic/gods/utils"
	"github.com/prometheus/client_golang/prometheus"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	reporterName = "prometheus"

	pullModel = "pull"
	pushModel = "push"

	applicationKey = constant.APPLICATION_KEY
	serviceKey     = constant.SERVICE_KEY
	groupKey       = constant.GROUP_KEY
	versionKey     = constant.VERSION_KEY
	timeoutKey     = constant.TIMEOUT_KEY
	instanceKey    = constant.INSTANCE_KEY
	categoryKey    = constant.CATEGORY_KEY

	// to identify side
	providerKey = "provider"
	consumerKey = "consumer"

	successful = "successful"
	failure    = "failure"

	DIGITS = 2

	histogramSuffix = "handling_second"
)

var (
	labelNames       = []string{applicationKey, serviceKey, groupKey, versionKey, categoryKey}
	reporterInstance *PrometheusReporter
	reporterInitOnce sync.Once
	pusherInitOnce   sync.Once
	// consumer instance virtual port
	virtualPort = strconv.Itoa(rand.Intn(20000) + 10000)
)

// should initialize after loading configuration
func init() {
	//newPrometheusReporter()
	extension.SetMetricReporter(reporterName, newPrometheusReporter)
}

// PrometheusReporter will collect the data for Prometheus
// if you want to use this feature, you need to initialize your prometheus.
// https://prometheus.io/docs/guides/go-application/
type PrometheusReporter struct {
	// report the provider-side's histogram data
	providerHistogramVec *prometheus.HistogramVec
	// report the consumer-side's histogram data
	consumerHistogramVec *prometheus.HistogramVec

	registry *prometheus.Registry
}

// Report reports the duration to Prometheus
// the role in url must be consumer or provider
// or it will be ignored
func (reporter *PrometheusReporter) Report(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation, cost time.Duration, res protocol.Result) {
	url := invoker.GetURL()
	var (
		hisVec   *prometheus.HistogramVec
		category = successful
	)

	if isProvider(url) {
		hisVec = reporter.providerHistogramVec
	} else if isConsumer(url) {
		hisVec = reporter.providerHistogramVec
	} else {
		logger.Warnf("The url belongs neither the consumer nor the provider, "+
			"so the invocation will be ignored. url: %s", url.String())
		return
	}

	if res.Error() != nil {
		category = failure
	}

	labels := prometheus.Labels{
		applicationKey: url.GetParam(applicationKey, ""),
		serviceKey:     url.Service(),
		groupKey:       url.GetParam(groupKey, ""),
		versionKey:     url.GetParam(constant.APP_VERSION_KEY, ""),
		categoryKey:    category,
	}

	hisVec.With(labels).Observe(float64(cost.Nanoseconds()))

	initPusher(url, reporter.registry)
}

func newHistogramVec(namespace, side, name string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: side,
			Name:      name,
			Help:      "This is the dubbo's histogram metrics",
			Buckets:   GetPercentileBucket(),
		},
		labelNames)
}

// isProvider shows whether this url represents the application received the request as server
func isProvider(url *common.URL) bool {
	side := url.GetParam(constant.SIDE_KEY, "")
	return strings.EqualFold(side, providerKey)
}

// isConsumer shows whether this url represents the application sent then request as client
func isConsumer(url *common.URL) bool {
	side := url.GetParam(constant.SIDE_KEY, "")
	return strings.EqualFold(side, consumerKey)
}

// newPrometheusReporter create new prometheusReporter
// it will register the metrics into prometheus
func newPrometheusReporter() metrics.Reporter {
	if reporterInstance == nil {
		reporterInitOnce.Do(func() {
			reporterConfig := config.GetMetricConfig()
			reporterInstance = &PrometheusReporter{
				consumerHistogramVec: newHistogramVec(reporterConfig.Namespace, consumerKey, histogramSuffix),
				providerHistogramVec: newHistogramVec(reporterConfig.Namespace, providerKey, histogramSuffix),
				registry:             prometheus.NewRegistry(),
			}

			reporterInstance.registry.MustRegister(reporterInstance.consumerHistogramVec, reporterInstance.providerHistogramVec)

			metricsExporter, err := ocprom.NewExporter(ocprom.Options{
				Registry: reporterInstance.registry,
			})

			if err != nil {
				logger.Errorf("new prometheus reporter with error = %s", err)
				return
			}
			if reporterConfig.Enable {
				if reporterConfig.Mode == pullModel {
					go func() {
						mux := http.NewServeMux()
						mux.Handle(reporterConfig.Path, metricsExporter)
						if err := http.ListenAndServe(":"+reporterConfig.Port, mux); err != nil {
							logger.Warnf("new prometheus reporter with error = %s", err)
						}
					}()
				}
			}
		})
	}
	return reporterInstance
}

// Delay create the pusher until the service invocation
func initPusher(url *common.URL, registry *prometheus.Registry) {
	pusherInitOnce.Do(func() {
		go func() {
			// pushgateway
			pushgateway := config.GetMetricConfig().Pushgateway

			if pushgateway != nil && pushgateway.Enabled {
				pusher := push.New(pushgateway.BaseUrl, config.GetApplicationConfig().Name).Gatherer(registry)
				if pushgateway.BasicAuth {
					pusher.BasicAuth(pushgateway.Username, pushgateway.Password)
				}

				for k, v := range pushgateway.GroupingKey {
					pusher.Grouping(k, v)
				}
				// need set instance as groupkey
				if isProvider(url) {
					pusher.Grouping(instanceKey, common.GetLocalIp()+":"+url.Port)
				} else {
					pusher.Grouping(instanceKey, common.GetLocalIp()+":"+virtualPort)
				}
				if pushRate, err := time.ParseDuration(pushgateway.PushRate); err == nil {
					go func(ticker *time.Ticker) {
						for {
							select {
							case <-ticker.C:
								if err := pusher.Add(); err != nil {
									logger.Warnf("Could not push to Pushgateway:", err)
								}
							}
						}
					}(time.NewTicker(pushRate))
				}
			}
		}()

	})
}

// The set of buckets is generated by using powers of 4 and incrementing by one-third of the
// previous power of 4 in between as long as the value is less than the next power of 4 minus
// the delta.
//
// <pre>
// Base: 1, 2, 3
//
// 4 (4^1), delta = 1
//     5, 6, 7, ..., 14,
//
// 16 (4^2), delta = 5
//    21, 26, 31, ..., 56,
//
// 64 (4^3), delta = 21
// ...
// </pre>
func GetPercentileBucket() []float64 {
	buckets := make([]float64, 0)
	set := treeset.NewWith(utils.Float64Comparator)
	set.Add(1.0)
	set.Add(2.0)
	set.Add(3.0)
	exp := DIGITS
	for exp < 64 {
		current := 1 << exp
		detal := current / 3
		next := (current << DIGITS) - detal

		for current < next {
			set.Add(float64(current))
			current += detal
		}
		exp += DIGITS
	}

	set = set.Select(func(index int, value interface{}) bool {
		if value.(float64) > float64(time.Millisecond) &&
			value.(float64) < float64(60*time.Second) {
			return true
		}
		return false

	})
	for _, x := range set.Values() {
		buckets = append(buckets, x.(float64))
	}
	return buckets
}
