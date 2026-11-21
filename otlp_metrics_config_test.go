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

package dubbo

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestOtlpMetricsYAMLConfigAndURL(t *testing.T) {
	conf := NewLoaderConf(WithGenre("yaml"), WithBytes([]byte(`
dubbo:
  application:
    name: otlp-yaml
  metrics:
    enable: true
    otlp:
      enabled: true
      endpoint: otel-collector:4317
      protocol: grpc
      insecure: true
`)))
	rc, err := loadInstanceOptions(conf)
	require.NoError(t, err)
	require.NoError(t, rc.initGlobalMetrics(false))
	u := metricsURL(rc)
	assert.True(t, u.GetParamBool(constant.OtlpMetricsEnabledKey, false))
	assert.Equal(t, "otel-collector:4317", u.GetParam(constant.OtlpMetricsEndpointKey, ""))
	assert.Equal(t, constant.OtlpMetricsProtocolGrpc, u.GetParam(constant.OtlpMetricsProtocolKey, ""))
	assert.True(t, u.GetParamBool(constant.OtlpMetricsInsecureKey, false))
	assert.Equal(t, "30", u.GetParam(constant.OtlpMetricsExportIntervalKey, ""), "default export interval")
}

func TestOtlpMetricsDisabledByDefault(t *testing.T) {
	conf := NewLoaderConf(WithGenre("yaml"), WithBytes([]byte(`
dubbo:
  application:
    name: otlp-default
  metrics:
    enable: true
`)))
	rc, err := loadInstanceOptions(conf)
	require.NoError(t, err)
	require.NoError(t, rc.initGlobalMetrics(false))
	assert.False(t, metricsURL(rc).GetParamBool(constant.OtlpMetricsEnabledKey, false))
}
