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
	"github.com/apache/dubbo-go/filter"
)

var (
	authenticators    = make(map[string]func() filter.Authenticator)
	accesskeyStorages = make(map[string]func() filter.AccessKeyStorage)
)

// SetAuthenticator puts the @fcn into map with name
func SetAuthenticator(name string, fcn func() filter.Authenticator) {
	authenticators[name] = fcn
}

// GetAuthenticator finds the Authenticator with @name
// Panic if not found
func GetAuthenticator(name string) filter.Authenticator {
	if authenticators[name] == nil {
		panic("authenticator for " + name + " is not existing, make sure you have import the package.")
	}
	return authenticators[name]()
}

// SetAccesskeyStorages will set the @fcn into map with this name
func SetAccesskeyStorages(name string, fcn func() filter.AccessKeyStorage) {
	accesskeyStorages[name] = fcn
}

// GetAccesskeyStorages finds the storage with the @name.
// Panic if not found
func GetAccesskeyStorages(name string) filter.AccessKeyStorage {
	if accesskeyStorages[name] == nil {
		panic("accesskeyStorages for " + name + " is not existing, make sure you have import the package.")
	}
	return accesskeyStorages[name]()
}
