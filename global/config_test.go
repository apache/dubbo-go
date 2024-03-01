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
}

func InitCheckCompleteInequality(t *testing.T, origin interface{}) {
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

func CheckCompleteInequality(t *testing.T, origin interface{}, clone interface{}) {
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
			assert.Equal(t, true, cloneField.Bool())
			cloneField.SetBool(false)
			assert.Equal(t, false, cloneField.Bool())
			assert.Equal(t, true, originField.Bool())

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
			assert.EqualValues(t, 1.5, cloneField.Float())
			cloneField.SetFloat(2.5)
			assert.EqualValues(t, 2.5, cloneField.Float())
			assert.EqualValues(t, 1.5, originField.Float())

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
