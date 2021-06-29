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

package config

import (
	"context"
)

// GenericService2 uses for generic invoke for service call,
// Unlike GenericService, it returns a tuple (interface{}, map[string]interface{}, error) called by DubboRPC,
// which contains the Attachments data returned by the DubboProvider interface
type GenericService2 struct {
	// Invoke this field will inject impl by proxy, returns (result interface{}, attachments map[string]interface{}, err error)
	Invoke       func(ctx context.Context, args []interface{}) (interface{}, map[string]interface{}, error) `dubbo:"$invoke"`
	referenceStr string
}

// NewGenericService2 returns a GenericService2 instance
func NewGenericService2(referenceStr string) *GenericService2 {
	return &GenericService2{referenceStr: referenceStr}
}

// Reference gets referenceStr from GenericService2
func (u *GenericService2) Reference() string {
	return u.referenceStr
}
