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

package protocol

import (
	"context"
	"github.com/pkg/errors"
)

type rpcExtraDataKey struct{}

const outgoingKey = "outgoingKey"
const incomingKey = "incomingKey"

var _ *rpcExtraDataKey = (*rpcExtraDataKey)(nil)

func SetIncomingData(ctx context.Context, extraData map[string]interface{}) context.Context {
	data, ok := ctx.Value(rpcExtraDataKey{}).(map[string]interface{})
	if !ok {
		data = map[string]interface{}{}
		ctx = context.WithValue(ctx, rpcExtraDataKey{}, data)
	}
	data[incomingKey] = extraData
	return ctx
}

func GetIncomingData(ctx context.Context) (map[string]interface{}, bool) {
	if data, ok := ctx.Value(rpcExtraDataKey{}).(map[string]interface{}); !ok {
		return nil, false
	} else {
		if incomingDataInterface, ok := data[incomingKey]; !ok {
			return nil, false
		} else if incomingData, ok := incomingDataInterface.(map[string]interface{}); !ok {
			return nil, false
		} else {
			// may need copy here ?
			return incomingData, true
		}
	}
}

func AppendIncomingData(ctx context.Context, kv ...string) (context.Context, error) {
	if kv == nil || len(kv)%2 != 0 {
		return ctx, errors.New("kv must be a non-nil slice with an even number of elements")
	}
	incomingData, ok := GetIncomingData(ctx)
	if !ok {
		incomingData = map[string]interface{}{}
	}
	for i := 0; i < len(kv); i += 2 {
		incomingData[kv[i]] = []string{kv[i+1]}
	}
	return SetIncomingData(ctx, incomingData), nil
}

func SetOutgoingData(ctx context.Context, extraData map[string]interface{}) context.Context {
	data, ok := ctx.Value(rpcExtraDataKey{}).(map[string]interface{})
	if !ok {
		data = map[string]interface{}{}
		ctx = context.WithValue(ctx, rpcExtraDataKey{}, data)
	}
	data[outgoingKey] = extraData
	return ctx
}

func GetOutgoingData(ctx context.Context) (map[string]interface{}, bool) {
	if data, ok := ctx.Value(rpcExtraDataKey{}).(map[string]interface{}); !ok {
		return nil, false
	} else {
		if OutgoingDataInterface, ok := data[outgoingKey]; !ok {
			return nil, false
		} else if OutgoingData, ok := OutgoingDataInterface.(map[string]interface{}); !ok {
			return nil, false
		} else {
			// may need copy here ?
			return OutgoingData, true
		}
	}
}

func AppendOutgoingData(ctx context.Context, kv ...string) (context.Context, error) {
	if kv == nil || len(kv)%2 != 0 {
		return ctx, errors.New("kv must be a non-nil slice with an even number of elements")
	}
	OutgoingData, ok := GetOutgoingData(ctx)
	if !ok {
		OutgoingData = map[string]interface{}{}
	}
	for i := 0; i < len(kv); i += 2 {
		OutgoingData[kv[i]] = []string{kv[i+1]}
	}
	return SetOutgoingData(ctx, OutgoingData), nil
}
