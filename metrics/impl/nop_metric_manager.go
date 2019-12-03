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

package impl

import (
	"time"

	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/metrics"
)

func init() {
	extension.SetMetricManager(constant.DEFAULT_KEY, newNopMetricManager())
}

type NopMetricManager struct {

}

func (manager *NopMetricManager) GetFastCompass(name string, metricName metrics.MetricName) metrics.FastCompass {
	return &NopFastCompass{}
}

func newNopMetricManager() metrics.MetricManager {
	return &NopMetricManager{}
}

type NopFastCompass struct {

}

func (compass *NopFastCompass) record(duration int64, subCategory string) {
}

func (compass *NopFastCompass) LastUpdateTime() int64 {
	return time.Now().UnixNano()/time.Millisecond.Nanoseconds()
}





