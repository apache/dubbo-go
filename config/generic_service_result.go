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

import (
	"github.com/apache/dubbo-go/protocol"
)

// ResultGenericService uses for generic invoke for service call,
// Unlike GenericService, it returns the original <protocol.Result> type object called by DubboRPC,
// which contains the Attachments data returned by the DubboProvider interface
type ResultGenericService struct {
	Invoke       func(ctx context.Context, args []interface{}) protocol.Result `dubbo:"$invoke"`
	referenceStr string
}

// NewResultGenericService returns a ResultGenericService instance
func NewResultGenericService(referenceStr string) *ResultGenericService {
	return &ResultGenericService{referenceStr: referenceStr}
}

// Reference gets referenceStr from ResultGenericService
func (u *ResultGenericService) Reference() string {
	return u.referenceStr
}
