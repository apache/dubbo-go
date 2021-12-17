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
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

var (
	ErrMetricsNotFound = errors.New("metrics not found")
)

type Metrics interface {
	// GetMethodMetrics returns method-level metrics, the format of key is "{instance key}.{invoker key}.{method key}.{key}"
	// url is invoker's url, which contains information about instance and invoker.
	// methodName is the method name.
	// key is the key of the metrics.
	GetMethodMetrics(url *common.URL, methodName, key string) (interface{}, error)
	SetMethodMetrics(url *common.URL, methodName, key string, value interface{}) error

	// GetInvokerMetrics returns invoker-level metrics, the format of key is "{instance key}.{invoker key}.{key}"
	// DO NOT IMPLEMENT FOR EARLIER VERSION
	GetInvokerMetrics(url *common.URL, key string) (interface{}, error)
	SetInvokerMetrics(url *common.URL, key string, value interface{}) error

	// GetInstanceMetrics returns instance-level metrics, the format of key is "{instance key}.{key}"
	// DO NOT IMPLEMENT FOR EARLIER VERSION
	GetInstanceMetrics(url *common.URL, key string) (interface{}, error)
	SetInstanceMetrics(url *common.URL, key string, value interface{}) error
}
