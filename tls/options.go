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

package tls

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// The consideration of not placing TLSOption in the global package is
// to prevent users from directly using the global package, so I created
// a new tls directory to allow users to establish config through the tls package.

type TLSOption func(*global.TLSConfig)

func WithTLS_CACertFile(file string) TLSOption {
	return func(cfg *global.TLSConfig) {
		cfg.CACertFile = file
	}
}

func WithTLS_TLSCertFile(file string) TLSOption {
	return func(cfg *global.TLSConfig) {
		cfg.TLSCertFile = file
	}
}

func WithTLS_TLSKeyFile(file string) TLSOption {
	return func(cfg *global.TLSConfig) {
		cfg.TLSKeyFile = file
	}
}

func WithTLS_TLSServerName(name string) TLSOption {
	return func(cfg *global.TLSConfig) {
		cfg.TLSServerName = name
	}
}
