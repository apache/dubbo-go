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

var configs = NewRegistry[Config]("config")

// Config describes an extension's typed configuration without exposing its
// fields to the core. The registered value is an immutable descriptor. New
// must return a fresh configuration instance for each Instance, Client, or
// Server; Init receives the scope selected by that entry point.
type Config interface {
	Prefix() string
	Scopes() Scope
	New() any
	Init(scope Scope, config any) error
}

// SetConfig registers an extension configuration descriptor. Registration is
// intentionally keyed by Prefix so YAML and typed options can locate it.
func SetConfig(c Config) {
	configs.Register(c.Prefix(), c)
}

// GetConfig returns the descriptor registered for prefix.
func GetConfig(prefix string) (Config, bool) {
	return configs.Get(prefix)
}

// MustGetConfig returns the descriptor registered for prefix or panics when it
// has not been imported.
func MustGetConfig(prefix string) Config {
	return configs.MustGet(prefix)
}

// UnregisterConfig removes a descriptor. It is intended for isolated tests.
func UnregisterConfig(prefix string) {
	configs.Unregister(prefix)
}
