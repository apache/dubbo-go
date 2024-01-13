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

// Package internal contains dubbo-go-internal code, to avoid polluting
// the top-level dubbo-go package.  It must not import any dubbo-go symbols
// except internal symbols to avoid circular dependencies.
package internal

import (
	"dubbo.apache.org/dubbo-go/v3/internal/reflection"
)

var (
	// HealthSetServingStatusServing is used to set service serving status
	HealthSetServingStatusServing = func(service string) {}
	// ReflectionRegister is used to register reflection service provider
	ReflectionRegister = func(reflection reflection.ServiceInfoProvider) {}
	// todo: add metadata func
)
