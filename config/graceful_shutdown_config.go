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
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// ShutdownConfig is kept as a compatibility alias for global.ShutdownConfig.
// Use global.ShutdownConfig directly in new code. Field documentation lives on
// global.ShutdownConfig because it owns the graceful shutdown configuration.
type ShutdownConfig = global.ShutdownConfig

type ShutdownConfigBuilder struct {
	shutdownConfig *ShutdownConfig
}

func NewShutDownConfigBuilder() *ShutdownConfigBuilder {
	return &ShutdownConfigBuilder{shutdownConfig: &ShutdownConfig{}}
}

func (scb *ShutdownConfigBuilder) SetTimeout(timeout string) *ShutdownConfigBuilder {
	scb.shutdownConfig.Timeout = timeout
	return scb
}

func (scb *ShutdownConfigBuilder) SetStepTimeout(stepTimeout string) *ShutdownConfigBuilder {
	scb.shutdownConfig.StepTimeout = stepTimeout
	return scb
}

func (scb *ShutdownConfigBuilder) SetNotifyTimeout(notifyTimeout string) *ShutdownConfigBuilder {
	scb.shutdownConfig.NotifyTimeout = notifyTimeout
	return scb
}

func (scb *ShutdownConfigBuilder) SetRejectRequestHandler(rejectRequestHandler string) *ShutdownConfigBuilder {
	scb.shutdownConfig.RejectRequestHandler = rejectRequestHandler
	return scb
}

func (scb *ShutdownConfigBuilder) SetRejectRequest(rejectRequest bool) *ShutdownConfigBuilder {
	scb.shutdownConfig.RejectRequest.Store(rejectRequest)
	return scb
}

func (scb *ShutdownConfigBuilder) SetInternalSignal(internalSignal bool) *ShutdownConfigBuilder {
	scb.shutdownConfig.InternalSignal = &internalSignal
	return scb
}

func (scb *ShutdownConfigBuilder) Build() *ShutdownConfig {
	defaults.MustSet(scb.shutdownConfig)
	return scb.shutdownConfig
}

func (scb *ShutdownConfigBuilder) SetOfflineRequestWindowTimeout(offlineRequestWindowTimeout string) *ShutdownConfigBuilder {
	scb.shutdownConfig.OfflineRequestWindowTimeout = offlineRequestWindowTimeout
	return scb
}
