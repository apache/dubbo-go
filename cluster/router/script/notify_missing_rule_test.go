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

package script

import (
	"fmt"
	"sync"
	"testing"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"
)

import (
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

// missingRuleConfig is a config center that holds no rules.
type missingRuleConfig struct {
	config_center.DynamicConfiguration
}

func (missingRuleConfig) GetRule(string, ...config_center.Option) (string, error) { return "", nil }
func (missingRuleConfig) AddListener(string, config_center.ConfigurationListener, ...config_center.Option) {
}
func (missingRuleConfig) RemoveListener(string, config_center.ConfigurationListener, ...config_center.Option) {
}

// errorRecorder records Errorf calls and delegates the rest.
type errorRecorder struct {
	logger.Logger
	mu     sync.Mutex
	errors []string
}

func (r *errorRecorder) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *errorRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string{}, r.errors...)
}

// TestNotifyWithoutRuleResetsRouterQuietly checks that a provider application
// without a script rule resets the router instead of logging a parse error.
func TestNotifyWithoutRuleResetsRouterQuietly(t *testing.T) {
	env := conf.GetEnvInstance()
	prevConfig := env.GetDynamicConfiguration()
	env.SetDynamicConfiguration(missingRuleConfig{})
	t.Cleanup(func() { env.SetDynamicConfiguration(prevConfig) })

	prevLogger := logger.GetLogger()
	rec := &errorRecorder{Logger: prevLogger}
	logger.SetLogger(rec)
	t.Cleanup(func() { logger.SetLogger(prevLogger) })

	// The router still holds a rule of a previous provider application.
	s := NewScriptRouter()
	s.enabled = true
	s.scriptType = "javascript"
	s.rawScript = "old script"

	s.Notify([]base.Invoker{base.NewBaseInvoker(url1())})

	assert.Empty(t, rec.snapshot())
	assert.False(t, s.enabled)
	assert.Empty(t, s.scriptType)
	assert.Empty(t, s.rawScript)
}
