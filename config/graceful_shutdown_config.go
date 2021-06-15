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

package config

import (
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

const (
	defaultTimeout     = 60 * time.Second
	defaultStepTimeout = 10 * time.Second
)

// ShutdownConfig is used as configuration for graceful shutdown
type ShutdownConfig struct {
	/*
	 * Total timeout. Even though we don't release all resources,
	 * the application will shutdown if the costing time is over this configuration. The unit is ms.
	 * default value is 60 * 1000 ms = 1 minutes
	 * In general, it should be bigger than 3 * StepTimeout.
	 */
	Timeout string `default:"60s" yaml:"timeout" json:"timeout,omitempty" property:"timeout"`
	/*
	 * the timeout on each step. You should evaluate the response time of request
	 * and the time that client noticed that server shutdown.
	 * For example, if your client will received the notification within 10s when you start to close server,
	 * and the 99.9% requests will return response in 2s, so the StepTimeout will be bigger than(10+2) * 1000ms,
	 * maybe (10 + 2*3) * 1000ms is a good choice.
	 */
	StepTimeout string `default:"10s" yaml:"step_timeout" json:"step.timeout,omitempty" property:"step.timeout"`
	// when we try to shutdown the application, we will reject the new requests. In most cases, you don't need to configure this.
	RejectRequestHandler string `yaml:"reject_handler" json:"reject_handler,omitempty" property:"reject_handler"`
	// true -> new request will be rejected.
	RejectRequest bool
	// true -> all requests had been processed. In provider side it means that all requests are returned response to clients
	// In consumer side, it means that all requests getting response from servers
	RequestsFinished bool
}

// nolint
func (config *ShutdownConfig) Prefix() string {
	return constant.ShutdownConfigPrefix
}

// nolint
func (config *ShutdownConfig) GetTimeout() time.Duration {
	result, err := time.ParseDuration(config.Timeout)
	if err != nil {
		logger.Errorf("The Timeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			config.Timeout, defaultTimeout.String(), err)
		return defaultTimeout
	}
	return result
}

// nolint
func (config *ShutdownConfig) GetStepTimeout() time.Duration {
	result, err := time.ParseDuration(config.StepTimeout)
	if err != nil {
		logger.Errorf("The StepTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			config.StepTimeout, defaultStepTimeout.String(), err)
		return defaultStepTimeout
	}
	return result
}
