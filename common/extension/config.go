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

package extension

import (
	"fmt"
	"reflect"
	"strings"
)

// Scope identifies the lifecycle in which an extension is initialized.
// Each initialization receives exactly one concrete scope.
type Scope uint8

const (
	// InstanceScope is the lifecycle of a dubbo.Instance.
	InstanceScope Scope = iota + 1
	// ClientScope is the lifecycle of a client/consumer.
	ClientScope
	// ServerScope is the lifecycle of a server/provider.
	ServerScope
)

func (s Scope) valid() bool {
	return s == InstanceScope || s == ClientScope || s == ServerScope
}

// Config is the configuration and initialization contract implemented by an
// external extension. A registered Config is an immutable prototype; New must
// return an independent configuration initialized with the extension defaults.
type Config interface {
	Prefix() string
	New() Config
	Init(scope Scope) error
	FilterNames(scope Scope) []string
}

// Option applies typed configuration to one extension. The core groups
// options by Prefix and applies them in declaration order.
type Option interface {
	Prefix() string
	Apply(config Config) error
}

var (
	configs = NewRegistry[Config]("config")
)

// RegisterConfig registers an immutable Config prototype. Duplicate prefixes
// are rejected so side-effect import order cannot change extension behavior.
func RegisterConfig(config Config) error {
	if configIsNil(config) {
		return fmt.Errorf("extension: config is nil")
	}
	rawPrefix := config.Prefix()
	prefix := strings.TrimSpace(rawPrefix)
	if prefix == "" {
		return fmt.Errorf("extension: config prefix is required")
	}
	if prefix != rawPrefix {
		return fmt.Errorf("extension: config prefix %q must not contain surrounding whitespace", rawPrefix)
	}
	created := config.New()
	if configIsNil(created) {
		return fmt.Errorf("extension %q: new config returned nil", prefix)
	}
	if created.Prefix() != prefix {
		return fmt.Errorf("extension %q: new config returned prefix %q", prefix, created.Prefix())
	}
	if !configs.RegisterIfAbsent(prefix, config) {
		return fmt.Errorf("extension %q: config is already registered", prefix)
	}
	return nil
}

// MustRegisterConfig registers a Config and panics on failure. It is intended
// for init functions in side-effect imported extension packages.
func MustRegisterConfig(config Config) {
	if err := RegisterConfig(config); err != nil {
		panic(err)
	}
}

// LookupConfig returns the Config prototype registered for prefix.
func LookupConfig(prefix string) (Config, bool) {
	return configs.Get(prefix)
}

// UnregisterConfig removes a Config. It is intended for isolated tests.
func UnregisterConfig(prefix string) {
	configs.Unregister(prefix)
}

func configIsNil(config Config) bool {
	if config == nil {
		return true
	}
	value := reflect.ValueOf(config)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
