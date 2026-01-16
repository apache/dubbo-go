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
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCloneConfig tests the Clone method of the *_config.go files
func TestCloneConfig(t *testing.T) {
	t.Run("ApplicationConfig", func(t *testing.T) {
		c := DefaultApplicationConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("ConfigCenterConfig", func(t *testing.T) {
		c := DefaultCenterConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("ConsumerConfig", func(t *testing.T) {
		c := DefaultConsumerConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("CustomConfig", func(t *testing.T) {
		c := DefaultCustomConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("LoggerConfig", func(t *testing.T) {
		c := DefaultLoggerConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("MetadataReportConfig", func(t *testing.T) {
		c := DefaultMetadataReportConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("MethodConfig", func(t *testing.T) {
		c := &MethodConfig{}
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("MetricConfig", func(t *testing.T) {
		c := DefaultMetricsConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)

		c2 := &AggregateConfig{}
		InitCheckCompleteInequality(t, c2)
		clone2 := c2.Clone()
		CheckCompleteInequality(t, c2, clone2)

		c3 := &PrometheusConfig{}
		InitCheckCompleteInequality(t, c3)
		clone3 := c3.Clone()
		CheckCompleteInequality(t, c3, clone3)

		c4 := &Exporter{}
		InitCheckCompleteInequality(t, c4)
		clone4 := c4.Clone()
		CheckCompleteInequality(t, c4, clone4)

		c5 := &PushgatewayConfig{}
		InitCheckCompleteInequality(t, c5)
		clone5 := c5.Clone()
		CheckCompleteInequality(t, c5, clone5)
	})

	t.Run("OtelConfig", func(t *testing.T) {
		c := DefaultOtelConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)

		c2 := &OtelTraceConfig{}
		InitCheckCompleteInequality(t, c2)
		clone2 := c2.Clone()
		CheckCompleteInequality(t, c2, clone2)
	})

	t.Run("ProfilesConfig", func(t *testing.T) {
		c := DefaultProfilesConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("ProtocolConfig", func(t *testing.T) {
		c := DefaultProtocolConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("ClientProtocolConfig", func(t *testing.T) {
		c := DefaultClientProtocolConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("ProviderConfig", func(t *testing.T) {
		c := DefaultProviderConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("ReferenceConfig", func(t *testing.T) {
		c := DefaultReferenceConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("RegistryConfig", func(t *testing.T) {
		c := DefaultRegistryConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("ServiceConfig", func(t *testing.T) {
		c := DefaultServiceConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("ShutdownConfig", func(t *testing.T) {
		c := DefaultShutdownConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("TLSConfig", func(t *testing.T) {
		c := &TLSConfig{}
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("TripleConfig", func(t *testing.T) {
		c := DefaultTripleConfig()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})

	t.Run("Http3Config", func(t *testing.T) {
		c := DefaultHttp3Config()
		InitCheckCompleteInequality(t, c)
		clone := c.Clone()
		CheckCompleteInequality(t, c, clone)
	})
}

func InitCheckCompleteInequality(t *testing.T, origin any) {
	if origin == nil {
		t.Errorf("Invalid parameters")
		return
	}

	originValue := reflect.ValueOf(origin).Elem()

	if originValue.Kind() != reflect.Struct {
		t.Errorf("Parameters should be struct types.")
		return
	}

	for i := 0; i < originValue.NumField(); i++ {
		originField := originValue.Field(i)

		if !originField.CanSet() {
			// Field '%s' is unexported, skipping checking
			continue
		}

		switch originField.Kind() {
		case reflect.String:
			originField.SetString("origin")

		case reflect.Bool:
			originField.SetBool(true)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			originField.SetInt(1)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			originField.SetUint(1)

		case reflect.Float32, reflect.Float64:
			originField.SetFloat(1.5)

		case reflect.Map:
			originField.Set(reflect.MakeMap(originField.Type()))

		case reflect.Slice, reflect.Ptr:

		default:
			// Field '%s' is of unsupported type '%s', skipping checking
		}
	}
}

func CheckCompleteInequality(t *testing.T, origin any, clone any) {
	if origin == nil || clone == nil {
		t.Errorf("Invalid parameters")
		return
	}

	assert.NotSame(t, origin, clone)

	originValue := reflect.ValueOf(origin).Elem()
	cloneValue := reflect.ValueOf(clone).Elem()

	if originValue.Kind() != reflect.Struct || cloneValue.Kind() != reflect.Struct {
		t.Errorf("Both parameters should be struct types.")
		return
	}

	if originValue.Type() != cloneValue.Type() {
		t.Errorf("Parameters should be of the same type.")
		return
	}

	for i := 0; i < originValue.NumField(); i++ {
		originField := originValue.Field(i)
		cloneField := cloneValue.Field(i)

		if !originField.CanSet() {
			t.Logf("Field '%s' is unexported, skipping checking", originValue.Type().Field(i).Name)
			continue
		}

		switch originField.Kind() {
		case reflect.String:
			assert.Equal(t, "origin", cloneField.String())
			cloneField.SetString("clone")
			assert.Equal(t, "clone", cloneField.String())
			assert.Equal(t, "origin", originField.String())

		case reflect.Bool:
			assert.True(t, cloneField.Bool())
			cloneField.SetBool(false)
			assert.False(t, cloneField.Bool())
			assert.True(t, originField.Bool())

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			assert.EqualValues(t, 1, cloneField.Int())
			cloneField.SetInt(2)
			assert.EqualValues(t, 2, cloneField.Int())
			assert.EqualValues(t, 1, originField.Int())

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			assert.EqualValues(t, 1, cloneField.Uint())
			cloneField.SetUint(2)
			assert.EqualValues(t, 2, cloneField.Uint())
			assert.EqualValues(t, 1, originField.Uint())

		case reflect.Float32, reflect.Float64:
			assert.InEpsilon(t, 1.5, cloneField.Float(), 1e-9)
			cloneField.SetFloat(2.5)
			assert.InEpsilon(t, 2.5, cloneField.Float(), 1e-9)
			assert.InEpsilon(t, 1.5, originField.Float(), 1e-9)

		case reflect.Map, reflect.Ptr:
			if originField.IsNil() {
				assert.Zero(t, cloneField.Pointer())
			} else {
				assert.NotZero(t, cloneField.Pointer())
				assert.NotEqual(t, originField.Pointer(), cloneField.Pointer())
			}

		case reflect.Slice:
			assert.NotEqual(t, originField.Pointer(), cloneField.Pointer())

		default:
			t.Logf("Field '%s' is of unsupported type '%s', skipping checking", originValue.Type().Field(i).Name, originField.Kind())
		}
	}
}
func TestReferenceConfigGetOptions(t *testing.T) {
	t.Run("full_reference_config", func(t *testing.T) {
		ref := &ReferenceConfig{
			InterfaceName:     "com.test.Service",
			URL:               "localhost:20880",
			Filter:            "echo",
			Protocol:          "dubbo",
			Cluster:           "failover",
			Loadbalance:       "random",
			Retries:           "3",
			Group:             "test",
			Version:           "1.0.0",
			Serialization:     "hessian2",
			ProvidedBy:        "provider",
			Async:             true,
			Generic:           "true",
			Sticky:            true,
			RequestTimeout:    "5s",
			TracingKey:        "jaeger",
			MeshProviderPort:  8080,
			KeepAliveInterval: "1m",
			KeepAliveTimeout:  "30s",
			Params: map[string]string{
				"key": "value",
			},
			Check:       func() *bool { b := true; return &b }(),
			RegistryIDs: []string{"reg1", "reg2"},
			MethodsConfig: []*MethodConfig{
				{Name: "method1"},
			},
			ProtocolClientConfig: DefaultClientProtocolConfig(),
		}

		opts := ref.GetOptions()
		assert.NotNil(t, opts, "opts should not be nil")
		assert.NotEmpty(t, opts, "opts should have items")
	})

	t.Run("empty_reference_config", func(t *testing.T) {
		emptyRef := &ReferenceConfig{}
		emptyOpts := emptyRef.GetOptions()
		// An empty config will return nil from GetOptions since all fields are empty
		if emptyOpts != nil {
			assert.Empty(t, emptyOpts, "empty ref config should have no options")
		}
	})

	t.Run("reference_config_with_retries_parsing", func(t *testing.T) {
		ref := &ReferenceConfig{
			Retries: "5",
		}
		opts := ref.GetOptions()
		assert.NotNil(t, opts)
		// Verify that a valid retries value produces at least one option
		assert.NotEmpty(t, opts, "should have at least one option from Retries")
	})

	t.Run("reference_config_with_invalid_retries", func(t *testing.T) {
		ref := &ReferenceConfig{
			Retries: "invalid",
		}
		opts := ref.GetOptions()
		// Invalid retries should not be added to options
		// This is expected behavior - invalid values are silently skipped
		// Verify the returned options do not contain a retries option
		// by confirming the invalid string is not converted

		// GetOptions returns nil (or empty slice) when no options are present
		// Since Retries is the only field and it's invalid, no options should be produced
		if opts != nil {
			// If opts is returned as a slice, it should be empty since Retries is invalid
			assert.Empty(t, opts, "invalid retries value should not produce any options")
		}
		// The fact that invalid retries is not in opts confirms it was rejected
	})
}

// TestReferenceConfigClone tests the Clone method of ReferenceConfig
func TestReferenceConfigClone(t *testing.T) {
	t.Run("nil_reference_config_clone", func(t *testing.T) {
		var nilRef *ReferenceConfig
		cloned := nilRef.Clone()
		assert.Nil(t, cloned)
	})
}

// TestReferenceConfigOptions tests the option functions
func TestReferenceConfigOptions(t *testing.T) {
	t.Run("WithReference_InterfaceName", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_InterfaceName("com.test.Service")
		opt(ref)
		assert.Equal(t, "com.test.Service", ref.InterfaceName)
	})

	t.Run("WithReference_Check_true", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Check(true)
		opt(ref)
		assert.NotNil(t, ref.Check)
		assert.True(t, *ref.Check)
	})

	t.Run("WithReference_Check_false", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Check(false)
		opt(ref)
		assert.NotNil(t, ref.Check)
		assert.False(t, *ref.Check)
	})

	t.Run("WithReference_URL", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_URL("localhost:20880")
		opt(ref)
		assert.Equal(t, "localhost:20880", ref.URL)
	})

	t.Run("WithReference_Filter", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Filter("echo")
		opt(ref)
		assert.Equal(t, "echo", ref.Filter)
	})

	t.Run("WithReference_Protocol", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Protocol("tri")
		opt(ref)
		assert.Equal(t, "tri", ref.Protocol)
	})

	t.Run("WithReference_RegistryIDs", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_RegistryIDs([]string{"reg1", "reg2"})
		opt(ref)
		assert.Len(t, ref.RegistryIDs, 2)
		assert.Equal(t, "reg1", ref.RegistryIDs[0])
		assert.Equal(t, "reg2", ref.RegistryIDs[1])
	})

	t.Run("WithReference_RegistryIDs_empty", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_RegistryIDs([]string{})
		opt(ref)
		assert.Empty(t, ref.RegistryIDs)
	})

	t.Run("WithReference_Cluster", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Cluster("failover")
		opt(ref)
		assert.Equal(t, "failover", ref.Cluster)
	})

	t.Run("WithReference_LoadBalance", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_LoadBalance("random")
		opt(ref)
		assert.Equal(t, "random", ref.Loadbalance)
	})

	t.Run("WithReference_Retries", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Retries(3)
		opt(ref)
		assert.Equal(t, "3", ref.Retries)
	})

	t.Run("WithReference_Group", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Group("test")
		opt(ref)
		assert.Equal(t, "test", ref.Group)
	})

	t.Run("WithReference_Version", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Version("1.0.0")
		opt(ref)
		assert.Equal(t, "1.0.0", ref.Version)
	})

	t.Run("WithReference_Serialization", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Serialization("hessian2")
		opt(ref)
		assert.Equal(t, "hessian2", ref.Serialization)
	})

	t.Run("WithReference_ProviderBy", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_ProviderBy("provider")
		opt(ref)
		assert.Equal(t, "provider", ref.ProvidedBy)
	})

	t.Run("WithReference_Async_true", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Async(true)
		opt(ref)
		assert.True(t, ref.Async)
	})

	t.Run("WithReference_Async_false", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Async(false)
		opt(ref)
		assert.False(t, ref.Async)
	})

	t.Run("WithReference_Params", func(t *testing.T) {
		ref := &ReferenceConfig{}
		params := map[string]string{"key": "value", "key2": "value2"}
		opt := WithReference_Params(params)
		opt(ref)
		assert.Equal(t, "value", ref.Params["key"])
		assert.Equal(t, "value2", ref.Params["key2"])
	})

	t.Run("WithReference_Generic", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Generic("true")
		opt(ref)
		assert.Equal(t, "true", ref.Generic)
	})

	t.Run("WithReference_Sticky", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Sticky(true)
		opt(ref)
		assert.True(t, ref.Sticky)
	})

	t.Run("WithReference_RequestTimeout", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_RequestTimeout("5s")
		opt(ref)
		assert.Equal(t, "5s", ref.RequestTimeout)
	})

	t.Run("WithReference_Force", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_Force(true)
		opt(ref)
		assert.True(t, ref.ForceTag)
	})

	t.Run("WithReference_TracingKey", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_TracingKey("jaeger")
		opt(ref)
		assert.Equal(t, "jaeger", ref.TracingKey)
	})

	t.Run("WithReference_MeshProviderPort", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_MeshProviderPort(8080)
		opt(ref)
		assert.Equal(t, 8080, ref.MeshProviderPort)
	})

	t.Run("WithReference_KeepAliveInterval", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_KeepAliveInterval("1m")
		opt(ref)
		assert.Equal(t, "1m", ref.KeepAliveInterval)
	})

	t.Run("WithReference_KeepAliveTimeout", func(t *testing.T) {
		ref := &ReferenceConfig{}
		opt := WithReference_KeepAliveTimeout("30s")
		opt(ref)
		assert.Equal(t, "30s", ref.KeepAliveTimeout)
	})

	t.Run("WithReference_ProtocolClientConfig", func(t *testing.T) {
		ref := &ReferenceConfig{}
		pcc := DefaultClientProtocolConfig()
		opt := WithReference_ProtocolClientConfig(pcc)
		opt(ref)
		assert.NotNil(t, ref.ProtocolClientConfig)
	})
}
func TestRegistryConfigUseAsMetadataReport(t *testing.T) {
	t.Run("empty_use_as_meta_report_defaults_to_true", func(t *testing.T) {
		reg := &RegistryConfig{UseAsMetaReport: ""}
		result, err := reg.UseAsMetadataReport()
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("true_string", func(t *testing.T) {
		reg := &RegistryConfig{UseAsMetaReport: "true"}
		result, err := reg.UseAsMetadataReport()
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("false_string", func(t *testing.T) {
		reg := &RegistryConfig{UseAsMetaReport: "false"}
		result, err := reg.UseAsMetadataReport()
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("TRUE_uppercase", func(t *testing.T) {
		reg := &RegistryConfig{UseAsMetaReport: "TRUE"}
		result, err := reg.UseAsMetadataReport()
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("FALSE_uppercase", func(t *testing.T) {
		reg := &RegistryConfig{UseAsMetaReport: "FALSE"}
		result, err := reg.UseAsMetadataReport()
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("invalid_use_as_meta_report", func(t *testing.T) {
		reg := &RegistryConfig{UseAsMetaReport: "invalid"}
		result, err := reg.UseAsMetadataReport()
		require.Error(t, err)
		assert.False(t, result)
	})

	t.Run("1_as_true", func(t *testing.T) {
		reg := &RegistryConfig{UseAsMetaReport: "1"}
		result, err := reg.UseAsMetadataReport()
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("0_as_false", func(t *testing.T) {
		reg := &RegistryConfig{UseAsMetaReport: "0"}
		result, err := reg.UseAsMetadataReport()
		require.NoError(t, err)
		assert.False(t, result)
	})
}

// TestRegistryConfigClone tests the Clone method of RegistryConfig
func TestRegistryConfigClone(t *testing.T) {
	t.Run("clone_full_registry_config", func(t *testing.T) {
		reg := &RegistryConfig{
			Protocol:          "zookeeper",
			Timeout:           "5s",
			Group:             "test",
			Namespace:         "ns",
			TTL:               "15m",
			Address:           "localhost:2181",
			Username:          "user",
			Password:          "pass",
			Simplified:        true,
			Preferred:         true,
			Zone:              "zone1",
			Weight:            100,
			UseAsMetaReport:   "true",
			UseAsConfigCenter: "true",
			RegistryType:      "nacos",
			Params: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}
		cloned := reg.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, reg.Protocol, cloned.Protocol)
		assert.Equal(t, reg.Timeout, cloned.Timeout)
		assert.Equal(t, reg.Group, cloned.Group)
		assert.Equal(t, reg.Namespace, cloned.Namespace)
		assert.Equal(t, reg.TTL, cloned.TTL)
		assert.Equal(t, reg.Address, cloned.Address)
		assert.Equal(t, reg.Username, cloned.Username)
		assert.Equal(t, reg.Password, cloned.Password)
		assert.Equal(t, reg.Simplified, cloned.Simplified)
		assert.Equal(t, reg.Preferred, cloned.Preferred)
		assert.Equal(t, reg.Zone, cloned.Zone)
		assert.Equal(t, reg.Weight, cloned.Weight)
		assert.Equal(t, reg.UseAsMetaReport, cloned.UseAsMetaReport)
		assert.Equal(t, reg.UseAsConfigCenter, cloned.UseAsConfigCenter)
		assert.NotSame(t, reg, cloned)
		assert.NotSame(t, reg.Params, cloned.Params)
		assert.Equal(t, "value1", cloned.Params["key1"])
		assert.Equal(t, "value2", cloned.Params["key2"])
	})

	t.Run("clone_nil_registry_config", func(t *testing.T) {
		var reg *RegistryConfig
		cloned := reg.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_empty_registry_config", func(t *testing.T) {
		reg := &RegistryConfig{}
		cloned := reg.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, reg, cloned)
	})

	t.Run("clone_with_empty_params", func(t *testing.T) {
		reg := &RegistryConfig{
			Protocol: "nacos",
			Params:   make(map[string]string),
		}
		cloned := reg.Clone()
		assert.NotNil(t, cloned)
		assert.Empty(t, cloned.Params)
		assert.NotSame(t, reg.Params, cloned.Params)
	})
}

// TestDefaultRegistryConfig tests DefaultRegistryConfig function
func TestDefaultRegistryConfig(t *testing.T) {
	t.Run("default_registry_config", func(t *testing.T) {
		reg := DefaultRegistryConfig()
		assert.NotNil(t, reg)
		assert.Equal(t, "true", reg.UseAsMetaReport)
		assert.Equal(t, "true", reg.UseAsConfigCenter)
		assert.Equal(t, "5s", reg.Timeout)
		assert.Equal(t, "15m", reg.TTL)
	})
}

// TestDefaultRegistriesConfig tests DefaultRegistriesConfig function
func TestDefaultRegistriesConfig(t *testing.T) {
	t.Run("default_registries_config", func(t *testing.T) {
		regs := DefaultRegistriesConfig()
		assert.NotNil(t, regs)
		assert.Empty(t, regs)
	})
}

// TestCloneRegistriesConfig tests CloneRegistriesConfig function
func TestCloneRegistriesConfig(t *testing.T) {
	t.Run("clone_multiple_registries", func(t *testing.T) {
		regs := map[string]*RegistryConfig{
			"reg1": {
				Protocol: "zookeeper",
				Address:  "localhost:2181",
				Params: map[string]string{
					"key": "value",
				},
			},
			"reg2": {
				Protocol: "nacos",
				Address:  "localhost:8848",
				Params: map[string]string{
					"key2": "value2",
				},
			},
		}
		cloned := CloneRegistriesConfig(regs)
		assert.Len(t, cloned, 2)
		assert.Equal(t, "zookeeper", cloned["reg1"].Protocol)
		assert.Equal(t, "nacos", cloned["reg2"].Protocol)
		assert.NotSame(t, regs["reg1"], cloned["reg1"])
		assert.NotSame(t, regs["reg2"], cloned["reg2"])
		assert.Equal(t, "value", cloned["reg1"].Params["key"])
		assert.Equal(t, "value2", cloned["reg2"].Params["key2"])
	})

	t.Run("clone_nil_registries_config", func(t *testing.T) {
		var regs map[string]*RegistryConfig
		cloned := CloneRegistriesConfig(regs)
		assert.Nil(t, cloned)
	})

	t.Run("clone_empty_registries_config", func(t *testing.T) {
		regs := make(map[string]*RegistryConfig)
		cloned := CloneRegistriesConfig(regs)
		assert.NotNil(t, cloned)
		assert.Empty(t, cloned)
		assert.NotSame(t, regs, cloned)
	})

	t.Run("clone_single_registry", func(t *testing.T) {
		regs := map[string]*RegistryConfig{
			"reg1": {
				Protocol:  "etcdv3",
				Timeout:   "10s",
				Weight:    50,
				Preferred: true,
			},
		}
		cloned := CloneRegistriesConfig(regs)
		assert.Len(t, cloned, 1)
		assert.Equal(t, "etcdv3", cloned["reg1"].Protocol)
		assert.Equal(t, "10s", cloned["reg1"].Timeout)
		assert.Equal(t, int64(50), cloned["reg1"].Weight)
		assert.True(t, cloned["reg1"].Preferred)
	})
}
func TestRouterConfigClone(t *testing.T) {
	t.Run("clone_full_router_config", func(t *testing.T) {
		router := &RouterConfig{
			Scope:      "service",
			Key:        "com.test.Service",
			Priority:   10,
			ScriptType: "javascript",
			Script:     "1==1",
			Force: func() *bool {
				b := true
				return &b
			}(),
			Runtime: func() *bool {
				b := false
				return &b
			}(),
			Enabled: func() *bool {
				b := true
				return &b
			}(),
			Valid: func() *bool {
				b := true
				return &b
			}(),
			Conditions: []string{"condition1", "condition2"},
			Tags: []Tag{
				{
					Name:      "tag1",
					Addresses: []string{"addr1", "addr2"},
				},
				{
					Name:      "tag2",
					Addresses: []string{"addr3"},
				},
			},
		}

		cloned := router.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, router.Scope, cloned.Scope)
		assert.Equal(t, router.Key, cloned.Key)
		assert.Equal(t, router.Priority, cloned.Priority)
		assert.Equal(t, router.ScriptType, cloned.ScriptType)
		assert.Equal(t, router.Script, cloned.Script)
		assert.NotSame(t, router, cloned)
		assert.NotSame(t, router.Conditions, cloned.Conditions)
		assert.NotSame(t, router.Tags, cloned.Tags)
		assert.Len(t, cloned.Conditions, 2)
		assert.Len(t, cloned.Tags, 2)
		assert.Equal(t, "tag1", cloned.Tags[0].Name)
		assert.Equal(t, "tag2", cloned.Tags[1].Name)
		assert.Len(t, cloned.Tags[0].Addresses, 2)
		assert.Len(t, cloned.Tags[1].Addresses, 1)
	})

	t.Run("clone_nil_router_config", func(t *testing.T) {
		var router *RouterConfig
		cloned := router.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_empty_router_config", func(t *testing.T) {
		router := &RouterConfig{}
		cloned := router.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, router, cloned)
	})

	t.Run("clone_router_config_with_nil_pointers", func(t *testing.T) {
		router := &RouterConfig{
			Scope:   "service",
			Key:     "com.test.Service",
			Force:   nil,
			Runtime: nil,
		}
		cloned := router.Clone()
		assert.NotNil(t, cloned)
		assert.Nil(t, cloned.Force)
		assert.Nil(t, cloned.Runtime)
	})

	t.Run("clone_router_config_preserves_pointer_values", func(t *testing.T) {
		forceTrue := true
		runtimeFalse := false
		router := &RouterConfig{
			Scope:   "service",
			Key:     "test",
			Force:   &forceTrue,
			Runtime: &runtimeFalse,
		}
		cloned := router.Clone()
		assert.NotNil(t, cloned.Force)
		assert.NotNil(t, cloned.Runtime)
		assert.True(t, *cloned.Force)
		assert.False(t, *cloned.Runtime)
		assert.NotSame(t, router.Force, cloned.Force)
		assert.NotSame(t, router.Runtime, cloned.Runtime)
	})
}

// TestDefaultRouterConfig tests DefaultRouterConfig function
func TestDefaultRouterConfig(t *testing.T) {
	t.Run("default_router_config", func(t *testing.T) {
		router := DefaultRouterConfig()
		assert.NotNil(t, router)
		assert.Empty(t, router.Conditions)
		assert.Empty(t, router.Tags)
	})
}

// TestTagStructure tests Tag structure and cloning
func TestTagStructure(t *testing.T) {
	t.Run("tag_with_addresses", func(t *testing.T) {
		tag := Tag{
			Name:      "test",
			Addresses: []string{"addr1", "addr2"},
		}
		assert.Equal(t, "test", tag.Name)
		assert.Len(t, tag.Addresses, 2)
	})

	t.Run("tag_with_empty_addresses", func(t *testing.T) {
		tag := Tag{
			Name:      "test",
			Addresses: []string{},
		}
		assert.Empty(t, tag.Addresses)
	})
}

// TestRouterConfigFields tests individual fields of RouterConfig
func TestRouterConfigFields(t *testing.T) {
	t.Run("router_config_scope", func(t *testing.T) {
		router := &RouterConfig{
			Scope: "application",
		}
		assert.Equal(t, "application", router.Scope)
	})

	t.Run("router_config_priority", func(t *testing.T) {
		router := &RouterConfig{
			Priority: 100,
		}
		assert.Equal(t, 100, router.Priority)
	})

	t.Run("router_config_script_type", func(t *testing.T) {
		router := &RouterConfig{
			ScriptType: "groovy",
			Script:     "1 == 1",
		}
		assert.Equal(t, "groovy", router.ScriptType)
		assert.Equal(t, "1 == 1", router.Script)
	})

	t.Run("router_config_conditions_multiple", func(t *testing.T) {
		router := &RouterConfig{
			Conditions: []string{"cond1", "cond2", "cond3"},
		}
		assert.Len(t, router.Conditions, 3)
		assert.Equal(t, "cond1", router.Conditions[0])
		assert.Equal(t, "cond3", router.Conditions[2])
	})
}
func TestProtocolConfigClone(t *testing.T) {
	t.Run("clone_full_protocol_config", func(t *testing.T) {
		proto := &ProtocolConfig{
			Name:                 "tri",
			Ip:                   "localhost",
			Port:                 "20880",
			MaxServerSendMsgSize: "1mb",
			MaxServerRecvMsgSize: "4mb",
			TripleConfig: &TripleConfig{
				MaxServerSendMsgSize: "2mb",
				MaxServerRecvMsgSize: "4mb",
				KeepAliveInterval:    "1m",
				KeepAliveTimeout:     "30s",
				Http3:                DefaultHttp3Config(),
			},
			Params: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}

		cloned := proto.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, proto.Name, cloned.Name)
		assert.Equal(t, proto.Ip, cloned.Ip)
		assert.Equal(t, proto.Port, cloned.Port)
		assert.Equal(t, proto.MaxServerSendMsgSize, cloned.MaxServerSendMsgSize)
		assert.Equal(t, proto.MaxServerRecvMsgSize, cloned.MaxServerRecvMsgSize)
		assert.NotSame(t, proto, cloned)
		assert.NotSame(t, proto.TripleConfig, cloned.TripleConfig)
		assert.NotNil(t, cloned.TripleConfig)
	})

	t.Run("clone_nil_protocol_config", func(t *testing.T) {
		var proto *ProtocolConfig
		cloned := proto.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_protocol_config_with_nil_triple_config", func(t *testing.T) {
		proto := &ProtocolConfig{
			Name: "dubbo",
			Port: "20880",
		}
		cloned := proto.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, "dubbo", cloned.Name)
	})

	t.Run("clone_protocol_config_preserves_all_fields", func(t *testing.T) {
		proto := &ProtocolConfig{
			Name:                 "http",
			Ip:                   "localhost",
			Port:                 "8080",
			MaxServerSendMsgSize: "10mb",
			MaxServerRecvMsgSize: "10mb",
		}
		cloned := proto.Clone()
		assert.Equal(t, "http", cloned.Name)
		assert.Equal(t, "localhost", cloned.Ip)
		assert.Equal(t, "8080", cloned.Port)
		assert.Equal(t, "10mb", cloned.MaxServerSendMsgSize)
		assert.Equal(t, "10mb", cloned.MaxServerRecvMsgSize)
	})
}

// TestDefaultProtocolConfig tests DefaultProtocolConfig function
func TestDefaultProtocolConfig(t *testing.T) {
	t.Run("default_protocol_config", func(t *testing.T) {
		proto := DefaultProtocolConfig()
		assert.NotNil(t, proto)
		assert.NotEmpty(t, proto.Name)
		assert.NotEmpty(t, proto.Port)
		assert.NotNil(t, proto.TripleConfig)
	})
}

// TestProtocolConfigFields tests individual fields of ProtocolConfig
func TestProtocolConfigFields(t *testing.T) {
	t.Run("protocol_config_name", func(t *testing.T) {
		proto := &ProtocolConfig{Name: "dubbo"}
		assert.Equal(t, "dubbo", proto.Name)
	})

	t.Run("protocol_config_ip", func(t *testing.T) {
		proto := &ProtocolConfig{Ip: "localhost"}
		assert.Equal(t, "localhost", proto.Ip)
	})

	t.Run("protocol_config_port", func(t *testing.T) {
		proto := &ProtocolConfig{Port: "9090"}
		assert.Equal(t, "9090", proto.Port)
	})

	t.Run("protocol_config_with_params", func(t *testing.T) {
		params := map[string]string{
			"param1": "value1",
			"param2": "value2",
		}
		proto := &ProtocolConfig{
			Name:   "tri",
			Params: params,
		}
		assert.NotNil(t, proto.Params)
	})

	t.Run("protocol_config_with_triple_config", func(t *testing.T) {
		tripleConfig := &TripleConfig{
			KeepAliveInterval: "2m",
			KeepAliveTimeout:  "60s",
		}
		proto := &ProtocolConfig{
			Name:         "tri",
			TripleConfig: tripleConfig,
		}
		assert.NotNil(t, proto.TripleConfig)
		assert.Equal(t, "2m", proto.TripleConfig.KeepAliveInterval)
		assert.Equal(t, "60s", proto.TripleConfig.KeepAliveTimeout)
	})
}

// TestTripleConfigClone tests the Clone method of TripleConfig
func TestTripleConfigClone(t *testing.T) {
	t.Run("clone_full_triple_config", func(t *testing.T) {
		triple := &TripleConfig{
			MaxServerSendMsgSize: "5mb",
			MaxServerRecvMsgSize: "5mb",
			KeepAliveInterval:    "3m",
			KeepAliveTimeout:     "45s",
			Http3:                DefaultHttp3Config(),
		}
		cloned := triple.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, triple.MaxServerSendMsgSize, cloned.MaxServerSendMsgSize)
		assert.Equal(t, triple.MaxServerRecvMsgSize, cloned.MaxServerRecvMsgSize)
		assert.Equal(t, triple.KeepAliveInterval, cloned.KeepAliveInterval)
		assert.Equal(t, triple.KeepAliveTimeout, cloned.KeepAliveTimeout)
		assert.NotSame(t, triple, cloned)
		assert.NotSame(t, triple.Http3, cloned.Http3)
	})

	t.Run("clone_nil_triple_config", func(t *testing.T) {
		var triple *TripleConfig
		cloned := triple.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_triple_config_with_nil_http3", func(t *testing.T) {
		triple := &TripleConfig{
			MaxServerSendMsgSize: "1mb",
		}
		cloned := triple.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, "1mb", cloned.MaxServerSendMsgSize)
	})
}

// TestDefaultTripleConfig tests DefaultTripleConfig function
func TestDefaultTripleConfig(t *testing.T) {
	t.Run("default_triple_config", func(t *testing.T) {
		triple := DefaultTripleConfig()
		assert.NotNil(t, triple)
		assert.NotNil(t, triple.Http3)
	})
}

// TestHttp3ConfigClone tests the Clone method of Http3Config
func TestHttp3ConfigClone(t *testing.T) {
	t.Run("clone_http3_config", func(t *testing.T) {
		http3 := &Http3Config{
			Enable:      true,
			Negotiation: false,
		}
		cloned := http3.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, http3.Enable, cloned.Enable)
		assert.Equal(t, http3.Negotiation, cloned.Negotiation)
	})

	t.Run("clone_nil_http3_config", func(t *testing.T) {
		var http3 *Http3Config
		cloned := http3.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_http3_config_with_defaults", func(t *testing.T) {
		http3 := DefaultHttp3Config()
		cloned := http3.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, http3.Enable, cloned.Enable)
		assert.Equal(t, http3.Negotiation, cloned.Negotiation)
	})
}

// TestDefaultHttp3Config tests DefaultHttp3Config function
func TestDefaultHttp3Config(t *testing.T) {
	t.Run("default_http3_config", func(t *testing.T) {
		http3 := DefaultHttp3Config()
		assert.NotNil(t, http3)
	})
}
func TestConsumerConfigClone(t *testing.T) {
	t.Run("clone_full_consumer_config", func(t *testing.T) {
		consumer := &ConsumerConfig{
			Filter:                         "echo",
			Protocol:                       "dubbo",
			RequestTimeout:                 "5s",
			ProxyFactory:                   "default",
			Check:                          true,
			AdaptiveService:                true,
			TracingKey:                     "jaeger",
			MeshEnabled:                    true,
			MaxWaitTimeForServiceDiscovery: "5s",
			RegistryIDs:                    []string{"reg1", "reg2"},
			References: map[string]*ReferenceConfig{
				"ref1": {
					InterfaceName: "com.test.Service1",
				},
				"ref2": {
					InterfaceName: "com.test.Service2",
				},
			},
		}

		cloned := consumer.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, consumer.Filter, cloned.Filter)
		assert.Equal(t, consumer.Protocol, cloned.Protocol)
		assert.Equal(t, consumer.RequestTimeout, cloned.RequestTimeout)
		assert.Equal(t, consumer.Check, cloned.Check)
		assert.Equal(t, consumer.AdaptiveService, cloned.AdaptiveService)
		assert.NotSame(t, consumer, cloned)
		assert.NotSame(t, consumer.References, cloned.References)
		assert.Len(t, cloned.References, 2)
		assert.Equal(t, "com.test.Service1", cloned.References["ref1"].InterfaceName)
	})

	t.Run("clone_nil_consumer_config", func(t *testing.T) {
		var consumer *ConsumerConfig
		cloned := consumer.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_consumer_config_with_empty_references", func(t *testing.T) {
		consumer := &ConsumerConfig{
			Filter: "echo",
		}
		cloned := consumer.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, "echo", cloned.Filter)
		assert.NotSame(t, consumer, cloned)
	})
}

// TestDefaultConsumerConfig tests DefaultConsumerConfig function
func TestDefaultConsumerConfig(t *testing.T) {
	t.Run("default_consumer_config", func(t *testing.T) {
		consumer := DefaultConsumerConfig()
		assert.NotNil(t, consumer)
		assert.True(t, consumer.Check)
		assert.NotNil(t, consumer.References)
		assert.Empty(t, consumer.References)
	})
}

// TestConsumerConfigFields tests individual fields of ConsumerConfig
func TestConsumerConfigFields(t *testing.T) {
	t.Run("consumer_config_filter", func(t *testing.T) {
		consumer := &ConsumerConfig{
			Filter: "echo",
		}
		assert.Equal(t, "echo", consumer.Filter)
	})

	t.Run("consumer_config_protocol", func(t *testing.T) {
		consumer := &ConsumerConfig{
			Protocol: "tri",
		}
		assert.Equal(t, "tri", consumer.Protocol)
	})

	t.Run("consumer_config_timeout", func(t *testing.T) {
		consumer := &ConsumerConfig{
			RequestTimeout: "10s",
		}
		assert.Equal(t, "10s", consumer.RequestTimeout)
	})

	t.Run("consumer_config_check", func(t *testing.T) {
		consumer := &ConsumerConfig{
			Check: false,
		}
		assert.False(t, consumer.Check)
	})

	t.Run("consumer_config_adaptive_service", func(t *testing.T) {
		consumer := &ConsumerConfig{
			AdaptiveService: true,
		}
		assert.True(t, consumer.AdaptiveService)
	})

	t.Run("consumer_config_registry_ids", func(t *testing.T) {
		consumer := &ConsumerConfig{
			RegistryIDs: []string{"reg1", "reg2", "reg3"},
		}
		assert.Len(t, consumer.RegistryIDs, 3)
	})

	t.Run("consumer_config_mesh_enabled", func(t *testing.T) {
		consumer := &ConsumerConfig{
			MeshEnabled: true,
		}
		assert.True(t, consumer.MeshEnabled)
	})
}
func TestServiceConfigClone(t *testing.T) {
	t.Run("clone_full_service_config", func(t *testing.T) {
		service := &ServiceConfig{
			Filter:                      "echo",
			Interface:                   "com.test.Service",
			Group:                       "test",
			Version:                     "1.0.0",
			Cluster:                     "failover",
			Loadbalance:                 "random",
			NotRegister:                 true,
			Weight:                      100,
			TracingKey:                  "jaeger",
			Auth:                        "token",
			Token:                       "abc123",
			AccessLog:                   "true",
			TpsLimiter:                  "default",
			TpsLimitInterval:            "1000",
			TpsLimitRate:                "100",
			TpsLimitStrategy:            "default",
			ExecuteLimit:                "10",
			ExecuteLimitRejectedHandler: "default",
			ParamSign:                   "true",
			Tag:                         "prod",
			Warmup:                      "60",
			Retries:                     "3",
			Serialization:               "hessian2",
			ProtocolIDs:                 []string{"proto1"},
			RegistryIDs:                 []string{"reg1"},
			Methods: []*MethodConfig{
				{Name: "method1"},
			},
			Params: map[string]string{
				"key": "value",
			},
			RCProtocolsMap: map[string]*ProtocolConfig{
				"proto1": {Name: "tri"},
			},
			RCRegistriesMap: map[string]*RegistryConfig{
				"reg1": {Protocol: "zookeeper"},
			},
		}

		cloned := service.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, service.Filter, cloned.Filter)
		assert.Equal(t, service.Interface, cloned.Interface)
		assert.Equal(t, service.Group, cloned.Group)
		assert.Equal(t, service.Version, cloned.Version)
		assert.NotSame(t, service, cloned)
		assert.NotSame(t, service.Params, cloned.Params)
		assert.Equal(t, "value", cloned.Params["key"])
		assert.Len(t, cloned.ProtocolIDs, 1)
		assert.Len(t, cloned.RegistryIDs, 1)
	})

	t.Run("clone_nil_service_config", func(t *testing.T) {
		var service *ServiceConfig
		cloned := service.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_empty_service_config", func(t *testing.T) {
		service := &ServiceConfig{}
		cloned := service.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, service, cloned)
	})

	t.Run("clone_service_config_with_maps", func(t *testing.T) {
		service := &ServiceConfig{
			Interface: "com.test.Service",
			RCProtocolsMap: map[string]*ProtocolConfig{
				"proto1": {Name: "tri", Port: "20880"},
				"proto2": {Name: "dubbo", Port: "20881"},
			},
			RCRegistriesMap: map[string]*RegistryConfig{
				"reg1": {Protocol: "zookeeper", Address: "localhost:2181"},
				"reg2": {Protocol: "nacos", Address: "localhost:8848"},
			},
		}
		cloned := service.Clone()
		assert.Len(t, cloned.RCProtocolsMap, 2)
		assert.Len(t, cloned.RCRegistriesMap, 2)
		assert.NotSame(t, service.RCProtocolsMap, cloned.RCProtocolsMap)
		assert.NotSame(t, service.RCRegistriesMap, cloned.RCRegistriesMap)
	})
}

// TestDefaultServiceConfig tests DefaultServiceConfig function
func TestDefaultServiceConfig(t *testing.T) {
	t.Run("default_service_config", func(t *testing.T) {
		service := DefaultServiceConfig()
		assert.NotNil(t, service)
		assert.NotNil(t, service.Methods)
		assert.NotNil(t, service.Params)
		assert.NotNil(t, service.RCProtocolsMap)
		assert.NotNil(t, service.RCRegistriesMap)
		assert.Empty(t, service.Methods)
		assert.Empty(t, service.Params)
	})
}

// TestServiceConfigFields tests individual fields of ServiceConfig
func TestServiceConfigFields(t *testing.T) {
	t.Run("service_config_interface", func(t *testing.T) {
		service := &ServiceConfig{
			Interface: "com.test.Service",
		}
		assert.Equal(t, "com.test.Service", service.Interface)
	})

	t.Run("service_config_version", func(t *testing.T) {
		service := &ServiceConfig{
			Version: "2.0.0",
		}
		assert.Equal(t, "2.0.0", service.Version)
	})

	t.Run("service_config_group", func(t *testing.T) {
		service := &ServiceConfig{
			Group: "production",
		}
		assert.Equal(t, "production", service.Group)
	})

	t.Run("service_config_cluster", func(t *testing.T) {
		service := &ServiceConfig{
			Cluster: "forking",
		}
		assert.Equal(t, "forking", service.Cluster)
	})

	t.Run("service_config_loadbalance", func(t *testing.T) {
		service := &ServiceConfig{
			Loadbalance: "leastactive",
		}
		assert.Equal(t, "leastactive", service.Loadbalance)
	})

	t.Run("service_config_weight", func(t *testing.T) {
		service := &ServiceConfig{
			Weight: 200,
		}
		assert.Equal(t, int64(200), service.Weight)
	})

	t.Run("service_config_not_register", func(t *testing.T) {
		service := &ServiceConfig{
			NotRegister: true,
		}
		assert.True(t, service.NotRegister)
	})

	t.Run("service_config_protocol_ids", func(t *testing.T) {
		service := &ServiceConfig{
			ProtocolIDs: []string{"proto1", "proto2"},
		}
		assert.Len(t, service.ProtocolIDs, 2)
	})

	t.Run("service_config_registry_ids", func(t *testing.T) {
		service := &ServiceConfig{
			RegistryIDs: []string{"reg1"},
		}
		assert.Len(t, service.RegistryIDs, 1)
	})
}

// TestServiceOptionFunc tests the ServiceOption function type
func TestServiceOptionFunc(t *testing.T) {
	t.Run("apply_service_option", func(t *testing.T) {
		service := &ServiceConfig{}
		opt := func(cfg *ServiceConfig) {
			cfg.Interface = "com.test.Service"
			cfg.Version = "1.0.0"
		}
		opt(service)
		assert.Equal(t, "com.test.Service", service.Interface)
		assert.Equal(t, "1.0.0", service.Version)
	})
}
func TestProviderConfigClone(t *testing.T) {
	t.Run("clone_full_provider_config", func(t *testing.T) {
		provider := &ProviderConfig{
			Filter:                 "echo",
			Register:               true,
			RegistryIDs:            []string{"reg1", "reg2"},
			ProtocolIDs:            []string{"proto1"},
			TracingKey:             "jaeger",
			ProxyFactory:           "default",
			AdaptiveService:        true,
			AdaptiveServiceVerbose: true,
			ConfigType:             map[string]string{"key": "value"},
			Services: map[string]*ServiceConfig{
				"service1": {Interface: "com.test.Service1"},
			},
			ServiceConfig: ServiceConfig{
				Interface: "com.test.BaseService",
				Group:     "test",
				Version:   "1.0.0",
			},
		}

		cloned := provider.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, provider.Filter, cloned.Filter)
		assert.Equal(t, provider.Register, cloned.Register)
		assert.NotSame(t, provider, cloned)
		assert.NotSame(t, provider.Services, cloned.Services)

		// Verify RegistryIDs is a true deep copy by mutating the clone
		assert.Len(t, cloned.RegistryIDs, len(provider.RegistryIDs))
		assert.Equal(t, provider.RegistryIDs, cloned.RegistryIDs)
		if len(cloned.RegistryIDs) > 0 {
			cloned.RegistryIDs[0] = "modified"
			assert.NotEqual(t, cloned.RegistryIDs[0], provider.RegistryIDs[0], "Modifying clone should not affect original")
		}
		cloned.RegistryIDs = append(cloned.RegistryIDs, "new_registry")
		assert.NotEqual(t, len(provider.RegistryIDs), len(cloned.RegistryIDs))

		// Verify ProtocolIDs is a true deep copy by mutating the clone
		assert.Len(t, cloned.ProtocolIDs, len(provider.ProtocolIDs))
		assert.Equal(t, provider.ProtocolIDs, cloned.ProtocolIDs)
		if len(cloned.ProtocolIDs) > 0 {
			cloned.ProtocolIDs[0] = "modified"
			assert.NotEqual(t, cloned.ProtocolIDs[0], provider.ProtocolIDs[0], "Modifying clone should not affect original")
		}
		cloned.ProtocolIDs = append(cloned.ProtocolIDs, "new_protocol")
		assert.NotEqual(t, len(provider.ProtocolIDs), len(cloned.ProtocolIDs))

		// Verify Services is a true deep copy by mutating the clone
		assert.Len(t, cloned.Services, len(provider.Services))
		cloned.Services["service2"] = &ServiceConfig{Interface: "com.test.Service2"}
		assert.NotEqual(t, len(provider.Services), len(cloned.Services))
		assert.NotContains(t, provider.Services, "service2")
		assert.Contains(t, cloned.Services, "service2")
	})

	t.Run("clone_nil_provider_config", func(t *testing.T) {
		var provider *ProviderConfig
		cloned := provider.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_default_provider_config", func(t *testing.T) {
		provider := DefaultProviderConfig()
		cloned := provider.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, provider, cloned)

		// Verify RegistryIDs is a true deep copy
		if len(provider.RegistryIDs) > 0 {
			assert.Equal(t, provider.RegistryIDs, cloned.RegistryIDs)
			cloned.RegistryIDs[0] = "modified"
			assert.NotEqual(t, provider.RegistryIDs[0], cloned.RegistryIDs[0], "Modifying clone should not affect original")
		}

		// Verify ProtocolIDs is a true deep copy
		if len(provider.ProtocolIDs) > 0 {
			assert.Equal(t, provider.ProtocolIDs, cloned.ProtocolIDs)
			cloned.ProtocolIDs[0] = "modified"
			assert.NotEqual(t, provider.ProtocolIDs[0], cloned.ProtocolIDs[0], "Modifying clone should not affect original")
		}

		// Verify Services is a true deep copy
		originalServicesLen := len(provider.Services)
		cloned.Services["new_service"] = &ServiceConfig{Interface: "com.test.NewService"}
		assert.Len(t, provider.Services, originalServicesLen)
		assert.Greater(t, len(cloned.Services), len(provider.Services))
	})
}

// TestDefaultProviderConfig tests DefaultProviderConfig function
func TestDefaultProviderConfig(t *testing.T) {
	t.Run("default_provider_config", func(t *testing.T) {
		provider := DefaultProviderConfig()
		assert.NotNil(t, provider)
		assert.NotNil(t, provider.RegistryIDs)
		assert.NotNil(t, provider.ProtocolIDs)
		assert.NotNil(t, provider.Services)
	})
}

// TestProviderConfigFields tests individual fields of ProviderConfig
func TestProviderConfigFields(t *testing.T) {
	t.Run("provider_config_filter", func(t *testing.T) {
		provider := &ProviderConfig{}
		provider.Filter = "echo"
		assert.Equal(t, "echo", provider.Filter)
	})

	t.Run("provider_config_register", func(t *testing.T) {
		provider := &ProviderConfig{
			Register: false,
		}
		assert.False(t, provider.Register)
	})

	t.Run("provider_config_registry_ids", func(t *testing.T) {
		provider := &ProviderConfig{
			RegistryIDs: []string{"reg1", "reg2"},
		}
		assert.Len(t, provider.RegistryIDs, 2)
	})

	t.Run("provider_config_protocol_ids", func(t *testing.T) {
		provider := &ProviderConfig{
			ProtocolIDs: []string{"proto1"},
		}
		assert.Len(t, provider.ProtocolIDs, 1)
	})

	t.Run("provider_config_tracing_key", func(t *testing.T) {
		provider := &ProviderConfig{
			TracingKey: "zipkin",
		}
		assert.Equal(t, "zipkin", provider.TracingKey)
	})

	t.Run("provider_config_adaptive_service", func(t *testing.T) {
		provider := &ProviderConfig{
			AdaptiveService:        true,
			AdaptiveServiceVerbose: true,
		}
		assert.True(t, provider.AdaptiveService)
		assert.True(t, provider.AdaptiveServiceVerbose)
	})

	t.Run("provider_config_services_map", func(t *testing.T) {
		provider := &ProviderConfig{
			Services: map[string]*ServiceConfig{
				"service1": {Interface: "com.test.Service1"},
				"service2": {Interface: "com.test.Service2"},
			},
		}
		assert.Len(t, provider.Services, 2)
		assert.Equal(t, "com.test.Service1", provider.Services["service1"].Interface)
	})

	t.Run("provider_config_proxy_factory", func(t *testing.T) {
		provider := &ProviderConfig{
			ProxyFactory: "javassist",
		}
		assert.Equal(t, "javassist", provider.ProxyFactory)
	})
}
func TestApplicationConfigClone(t *testing.T) {
	t.Run("clone_full_application_config", func(t *testing.T) {
		app := &ApplicationConfig{
			Organization:            "apache",
			Name:                    "dubbo-app",
			Module:                  "sample",
			Group:                   "test",
			Version:                 "1.0.0",
			Owner:                   "team",
			Environment:             "production",
			MetadataType:            "local",
			Tag:                     "v1",
			MetadataServicePort:     "20881",
			MetadataServiceProtocol: "dubbo",
		}
		cloned := app.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, app.Organization, cloned.Organization)
		assert.Equal(t, app.Name, cloned.Name)
		assert.NotSame(t, app, cloned)
	})

	t.Run("clone_nil_application_config", func(t *testing.T) {
		var app *ApplicationConfig
		cloned := app.Clone()
		assert.Nil(t, cloned)
	})
}

// TestDefaultApplicationConfig tests DefaultApplicationConfig function
func TestDefaultApplicationConfig(t *testing.T) {
	t.Run("default_application_config", func(t *testing.T) {
		app := DefaultApplicationConfig()
		assert.NotNil(t, app)
	})
}

// TestApplicationConfigFields tests individual fields
func TestApplicationConfigFields(t *testing.T) {
	t.Run("application_config_organization", func(t *testing.T) {
		app := &ApplicationConfig{
			Organization: "company",
		}
		assert.Equal(t, "company", app.Organization)
	})

	t.Run("application_config_name", func(t *testing.T) {
		app := &ApplicationConfig{
			Name: "my-app",
		}
		assert.Equal(t, "my-app", app.Name)
	})

	t.Run("application_config_metadata_type", func(t *testing.T) {
		app := &ApplicationConfig{
			MetadataType: "remote",
		}
		assert.Equal(t, "remote", app.MetadataType)
	})
}
func TestCustomConfigClone(t *testing.T) {
	t.Run("clone_full_custom_config", func(t *testing.T) {
		custom := &CustomConfig{
			ConfigMap: map[string]any{
				"key1": "value1",
				"key2": 123,
			},
		}
		cloned := custom.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, custom, cloned)
		if custom.ConfigMap != nil {
			assert.NotSame(t, custom.ConfigMap, cloned.ConfigMap)
		}
	})

	t.Run("clone_nil_custom_config", func(t *testing.T) {
		var custom *CustomConfig
		cloned := custom.Clone()
		assert.Nil(t, cloned)
	})
}

// TestDefaultCustomConfig tests DefaultCustomConfig function
func TestDefaultCustomConfig(t *testing.T) {
	t.Run("default_custom_config", func(t *testing.T) {
		custom := DefaultCustomConfig()
		assert.NotNil(t, custom)
	})
}
func TestMethodConfigClone(t *testing.T) {
	t.Run("clone_full_method_config", func(t *testing.T) {
		method := &MethodConfig{
			Name:           "testMethod",
			Retries:        "3",
			RequestTimeout: "5s",
		}
		cloned := method.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, method.Name, cloned.Name)
		assert.NotSame(t, method, cloned)
	})

	t.Run("clone_nil_method_config", func(t *testing.T) {
		var method *MethodConfig
		cloned := method.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_empty_method_config", func(t *testing.T) {
		method := &MethodConfig{}
		cloned := method.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, method, cloned)
	})
}

// TestMethodConfigFields tests individual fields
func TestMethodConfigFields(t *testing.T) {
	t.Run("method_config_name", func(t *testing.T) {
		method := &MethodConfig{
			Name: "method1",
		}
		assert.Equal(t, "method1", method.Name)
	})

	t.Run("method_config_retries", func(t *testing.T) {
		method := &MethodConfig{
			Retries: "5",
		}
		assert.Equal(t, "5", method.Retries)
	})

	t.Run("method_config_request_timeout", func(t *testing.T) {
		method := &MethodConfig{
			RequestTimeout: "10s",
		}
		assert.Equal(t, "10s", method.RequestTimeout)
	})
}
func TestMetricConfigClone(t *testing.T) {
	t.Run("clone_metrics_config", func(t *testing.T) {
		metrics := DefaultMetricsConfig()
		if metrics != nil {
			cloned := metrics.Clone()
			assert.NotNil(t, cloned)
			assert.NotSame(t, metrics, cloned)
		}
	})

	t.Run("clone_nil_metrics_config", func(t *testing.T) {
		var metrics *MetricsConfig
		cloned := metrics.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_aggregate_config", func(t *testing.T) {
		agg := &AggregateConfig{}
		cloned := agg.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, agg, cloned)
	})

	t.Run("clone_prometheus_config", func(t *testing.T) {
		prom := &PrometheusConfig{}
		cloned := prom.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, prom, cloned)
	})

	t.Run("clone_exporter", func(t *testing.T) {
		exp := &Exporter{}
		cloned := exp.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, exp, cloned)
	})

	t.Run("clone_pushgateway_config", func(t *testing.T) {
		push := &PushgatewayConfig{}
		cloned := push.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, push, cloned)
	})
}

// TestDefaultMetricsConfig tests DefaultMetricsConfig function
func TestDefaultMetricsConfig(t *testing.T) {
	t.Run("default_metrics_config", func(t *testing.T) {
		metrics := DefaultMetricsConfig()
		assert.NotNil(t, metrics)
	})
}
func TestOtelConfigClone(t *testing.T) {
	t.Run("clone_otel_config", func(t *testing.T) {
		otel := DefaultOtelConfig()
		if otel != nil {
			cloned := otel.Clone()
			assert.NotNil(t, cloned)
			assert.NotSame(t, otel, cloned)
		}
	})

	t.Run("clone_nil_otel_config", func(t *testing.T) {
		var otel *OtelConfig
		cloned := otel.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("clone_otel_trace_config", func(t *testing.T) {
		trace := &OtelTraceConfig{}
		cloned := trace.Clone()
		assert.NotNil(t, cloned)
		assert.NotSame(t, trace, cloned)
	})
}

// TestDefaultOtelConfig tests DefaultOtelConfig function
func TestDefaultOtelConfig(t *testing.T) {
	t.Run("default_otel_config", func(t *testing.T) {
		otel := DefaultOtelConfig()
		assert.NotNil(t, otel)
	})
}

// TestProfilesConfigClone tests Clone method
func TestProfilesConfigClone(t *testing.T) {
	t.Run("clone_profiles_config", func(t *testing.T) {
		profiles := DefaultProfilesConfig()
		if profiles != nil {
			cloned := profiles.Clone()
			assert.NotNil(t, cloned)
			assert.NotSame(t, profiles, cloned)
		}
	})

	t.Run("clone_nil_profiles_config", func(t *testing.T) {
		var profiles *ProfilesConfig
		cloned := profiles.Clone()
		assert.Nil(t, cloned)
	})
}

// TestDefaultProfilesConfig tests DefaultProfilesConfig function
func TestDefaultProfilesConfig(t *testing.T) {
	t.Run("default_profiles_config", func(t *testing.T) {
		profiles := DefaultProfilesConfig()
		assert.NotNil(t, profiles)
	})
}

// TestConfigCenterConfigClone tests Clone method
func TestConfigCenterConfigClone(t *testing.T) {
	t.Run("clone_center_config", func(t *testing.T) {
		center := DefaultCenterConfig()
		if center != nil {
			cloned := center.Clone()
			assert.NotNil(t, cloned)
			assert.NotSame(t, center, cloned)
		}
	})

	t.Run("clone_nil_center_config", func(t *testing.T) {
		var center *CenterConfig
		cloned := center.Clone()
		assert.Nil(t, cloned)
	})
}

// TestDefaultCenterConfig tests DefaultCenterConfig function
func TestDefaultCenterConfig(t *testing.T) {
	t.Run("default_center_config", func(t *testing.T) {
		center := DefaultCenterConfig()
		assert.NotNil(t, center)
	})
}

// TestMetadataReportConfigClone tests Clone method
func TestMetadataReportConfigClone(t *testing.T) {
	t.Run("clone_metadata_report_config", func(t *testing.T) {
		metadata := DefaultMetadataReportConfig()
		if metadata != nil {
			cloned := metadata.Clone()
			assert.NotNil(t, cloned)
			assert.NotSame(t, metadata, cloned)
		}
	})

	t.Run("clone_nil_metadata_report_config", func(t *testing.T) {
		var metadata *MetadataReportConfig
		cloned := metadata.Clone()
		assert.Nil(t, cloned)
	})
}

// TestDefaultMetadataReportConfig tests DefaultMetadataReportConfig function
func TestDefaultMetadataReportConfig(t *testing.T) {
	t.Run("default_metadata_report_config", func(t *testing.T) {
		metadata := DefaultMetadataReportConfig()
		assert.NotNil(t, metadata)
	})
}
func TestShutdownConfigClone(t *testing.T) {
	t.Run("clone_shutdown_config", func(t *testing.T) {
		shutdown := DefaultShutdownConfig()
		if shutdown != nil {
			cloned := shutdown.Clone()
			assert.NotNil(t, cloned)
			assert.NotSame(t, shutdown, cloned)
		}
	})

	t.Run("clone_nil_shutdown_config", func(t *testing.T) {
		var shutdown *ShutdownConfig
		cloned := shutdown.Clone()
		assert.Nil(t, cloned)
	})
}

// TestDefaultShutdownConfig tests DefaultShutdownConfig function
func TestDefaultShutdownConfig(t *testing.T) {
	t.Run("default_shutdown_config", func(t *testing.T) {
		shutdown := DefaultShutdownConfig()
		assert.NotNil(t, shutdown)
	})
}

// TestTLSConfigClone tests Clone method
func TestTLSConfigClone(t *testing.T) {
	t.Run("clone_tls_config", func(t *testing.T) {
		tls := &TLSConfig{
			CACertFile:    "/path/to/ca",
			TLSCertFile:   "/path/to/cert",
			TLSKeyFile:    "/path/to/key",
			TLSServerName: "localhost",
		}
		cloned := tls.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, tls.CACertFile, cloned.CACertFile)
		assert.Equal(t, tls.TLSCertFile, cloned.TLSCertFile)
		assert.NotSame(t, tls, cloned)
	})

	t.Run("clone_nil_tls_config", func(t *testing.T) {
		var tls *TLSConfig
		cloned := tls.Clone()
		assert.Nil(t, cloned)
	})
}

// TestClientProtocolConfigClone tests Clone method
func TestClientProtocolConfigClone(t *testing.T) {
	t.Run("clone_client_protocol_config", func(t *testing.T) {
		client := DefaultClientProtocolConfig()
		if client != nil {
			cloned := client.Clone()
			assert.NotNil(t, cloned)
			assert.NotSame(t, client, cloned)
		}
	})

	t.Run("clone_nil_client_protocol_config", func(t *testing.T) {
		var client *ClientProtocolConfig
		cloned := client.Clone()
		assert.Nil(t, cloned)
	})
}

// TestDefaultClientProtocolConfig tests DefaultClientProtocolConfig function
func TestDefaultClientProtocolConfig(t *testing.T) {
	t.Run("default_client_protocol_config", func(t *testing.T) {
		client := DefaultClientProtocolConfig()
		assert.NotNil(t, client)
	})
}
func TestLoggerConfigClone(t *testing.T) {
	t.Run("clone_full_logger_config", func(t *testing.T) {
		logger := &LoggerConfig{
			Level: "info",
		}
		cloned := logger.Clone()
		assert.NotNil(t, cloned)
		assert.Equal(t, logger.Level, cloned.Level)
		assert.NotSame(t, logger, cloned)
	})

	t.Run("clone_nil_logger_config", func(t *testing.T) {
		var logger *LoggerConfig
		cloned := logger.Clone()
		assert.Nil(t, cloned)
	})
}

// TestDefaultLoggerConfig tests DefaultLoggerConfig function
func TestDefaultLoggerConfig(t *testing.T) {
	t.Run("default_logger_config", func(t *testing.T) {
		logger := DefaultLoggerConfig()
		assert.NotNil(t, logger)
	})
}

// TestLoggerConfigFields tests individual fields
func TestLoggerConfigFields(t *testing.T) {
	t.Run("logger_config_level", func(t *testing.T) {
		logger := &LoggerConfig{
			Level: "debug",
		}
		assert.Equal(t, "debug", logger.Level)
	})
}
