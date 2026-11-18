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

package triple

import (
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

func TestLegacyClientRejectsConfiguredPluginsBeforeConstruction(t *testing.T) {
	const name = "legacy-validation-message"
	made := 0
	require.NoError(t, extension.RegisterSerialization(serialization.Registration{Protocol: "triple", Name: name, WireName: name, Mode: serialization.MessageMode, New: func(serialization.Options) (serialization.Codec, error) {
		made++
		return &codecEpochProbe{name: name}, nil
	}}))
	for _, acceptedOnly := range []bool{false, true} {
		url := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithParamsValue("serialization", name), common.WithAttribute(constant.RpcServiceKey, &struct{}{}))
		if acceptedOnly {
			url.SetParam("serialization", "protobuf")
			url.SetParam("accepted-serializations", "proto,"+name)
		}
		require.ErrorContains(t, (&TripleProtocol{}).PrepareCodec(url, "consumer"), "legacy generated clients")
		require.Zero(t, made)
	}
	for _, name := range []string{"protobuf", "json", "hessian2", "msgpack"} {
		url := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithParamsValue("serialization", name), common.WithAttribute(constant.RpcServiceKey, &struct{}{}))
		require.NoError(t, (&TripleProtocol{}).PrepareCodec(url, "consumer"))
		require.NoError(t, (&TripleProtocol{}).ReleaseCodec(url, "consumer"))
	}
}

func TestLegacyClientRejectsExplicitReceivePolicies(t *testing.T) {
	for _, accepted := range []string{"json", "hessian2", "proto,json,hessian2", "", "custom-unregistered"} {
		t.Run("accepted="+accepted, func(t *testing.T) {
			url := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithParamsValue("serialization", "hessian2"), common.WithParamsValue("accepted-serializations", accepted), common.WithAttribute(constant.RpcServiceKey, &struct{}{}))
			require.ErrorContains(t, (&TripleProtocol{}).PrepareCodec(url, "consumer"), "accepted serializations are not supported")
			_, err := NewDubbo3Invoker(url)
			require.ErrorContains(t, err, "accepted serializations are not supported")
			_, created := url.GetAttribute("triple.serialization.consumer")
			require.False(t, created)
			// Current Connection clients can enforce these policies and must not be
			// rejected by the legacy entry-point guard.
			url.SetAttribute(constant.ClientConnectionKey, true)
			if accepted == "hessian2" || accepted == "proto,json,hessian2" {
				require.NoError(t, (&TripleProtocol{}).PrepareCodec(url, "consumer"))
				require.NoError(t, (&TripleProtocol{}).ReleaseCodec(url, "consumer"))
			}
		})
	}
}
