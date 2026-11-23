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

package global

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsConfigCloneCopiesOtlpConfig(t *testing.T) {
	enabled := true
	c := &MetricsConfig{Otlp: &OtlpMetricsConfig{Enabled: &enabled, Endpoint: "collector:4318", Protocol: "http", ExportInterval: 10}}
	clone := c.Clone()
	require.NotNil(t, clone.Otlp)
	assert.Equal(t, *c.Otlp.Enabled, *clone.Otlp.Enabled)
	assert.Equal(t, "collector:4318", clone.Otlp.Endpoint)
	assert.Equal(t, 10, clone.Otlp.ExportInterval)

	*clone.Otlp.Enabled = false
	clone.Otlp.Endpoint = "other:4318"
	assert.True(t, *c.Otlp.Enabled, "the clone does not share the enabled flag")
	assert.Equal(t, "collector:4318", c.Otlp.Endpoint)

	assert.Nil(t, (&MetricsConfig{}).Clone().Otlp)
}
