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

package graceful_shutdown

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"go.uber.org/atomic"
)

func compatShutdownConfig(c *global.ShutdownConfig) *config.ShutdownConfig {
	if c == nil {
		return nil
	}
	cfg := &config.ShutdownConfig{
		Timeout:                     c.Timeout,
		StepTimeout:                 c.StepTimeout,
		ConsumerUpdateWaitTime:      c.ConsumerUpdateWaitTime,
		RejectRequestHandler:        c.RejectRequestHandler,
		InternalSignal:              c.InternalSignal,
		OfflineRequestWindowTimeout: c.OfflineRequestWindowTimeout,
		RejectRequest:               atomic.Bool{},
	}
	cfg.RejectRequest.Store(c.RejectRequest.Load())
	return cfg
}
