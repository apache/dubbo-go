/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dubbo

import (
	"errors"
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo/impl"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func TestResponseExceptionKind(t *testing.T) {
	for status, want := range map[uint8]rpcerror.Kind{
		impl.Response_OK:                  rpcerror.KindBusiness,
		impl.Response_SERIALIZATION_ERROR: rpcerror.KindCodec,
		impl.Response_CLIENT_TIMEOUT:      rpcerror.KindTimeout,
		impl.Response_SERVER_TIMEOUT:      rpcerror.KindTimeout,
		impl.Response_SERVICE_NOT_FOUND:   rpcerror.KindServiceUnavailable,
		impl.Response_BAD_REQUEST:         rpcerror.KindCodec,
		impl.Response_BAD_RESPONSE:        rpcerror.KindCodec,
		impl.Response_SERVICE_ERROR:       rpcerror.KindUnknown,
		impl.Response_SERVER_ERROR:        rpcerror.KindUnknown,
		impl.Response_CLIENT_ERROR:        rpcerror.KindUnknown,
		0:                                 rpcerror.KindUnknown,
	} {
		assert.Equal(t, want, responseExceptionKind(status), "status %d", status)
	}
}

// encodeDecodeResponse round-trips a response through the Dubbo codec and
// returns the decoded transport response.
func encodeDecodeResponse(t *testing.T, id int64, status uint8, err error) *remoting.Response {
	t.Helper()
	response := remoting.NewResponse(id, "2.0.2")
	response.SerialID = constant.SHessian2
	response.Status = status
	response.Result = &result.RPCResult{Err: err}
	buf, encodeErr := (&DubboCodec{}).EncodeResponse(response)
	require.NoError(t, encodeErr)

	decoded, _, decodeErr := (&DubboCodec{}).Decode(buf.Bytes())
	require.NoError(t, decodeErr)
	rsp, ok := decoded.Result.(*remoting.Response)
	require.True(t, ok)
	rpcResult, ok := rsp.Result.(*result.RPCResult)
	require.True(t, ok)
	require.Error(t, rpcResult.Err)
	assert.Equal(t, rpcResult.Err, rsp.Error, "sync and async paths must see the same error")
	assert.Equal(t, rpcerror.KindUnknown, rpcerror.KindOf(rsp.Error), "the decoded exception must not be wrapped")
	return rsp
}

func TestDecodeResponseClassifiesStatus(t *testing.T) {
	for status, want := range map[uint8]rpcerror.Kind{
		impl.Response_SERIALIZATION_ERROR: rpcerror.KindCodec,
		impl.Response_SERVER_TIMEOUT:      rpcerror.KindTimeout,
		impl.Response_SERVICE_NOT_FOUND:   rpcerror.KindServiceUnavailable,
		impl.Response_BAD_REQUEST:         rpcerror.KindCodec,
		impl.Response_SERVER_ERROR:        rpcerror.KindUnknown,
	} {
		rsp := encodeDecodeResponse(t, int64(status), status, errors.New("boom"))
		assert.Equal(t, want, rsp.Kind, "status %d", status)
		assert.Equal(t, "java exception:boom", rsp.Error.Error())
	}
}

// Async callbacks receive the decoded *remoting.Response directly, so the
// provider exception must keep its concrete type there.
func TestDecodeResponseKeepsBusinessExceptionType(t *testing.T) {
	const id = 3337
	remoting.AddPendingResponse(remoting.NewPendingResponse(id))
	defer remoting.RemovePendingResponse(id)

	thrown := &hessian.GenericException{ExceptionClass: "com.example.BizException", ExceptionMessage: "insufficient balance"}
	rsp := encodeDecodeResponse(t, id, impl.Response_OK, thrown)
	assert.Equal(t, rpcerror.KindBusiness, rsp.Kind)

	decoded, ok := rsp.Error.(*hessian.GenericException)
	require.True(t, ok, "got %T, want *hessian.GenericException", rsp.Error)
	assert.Equal(t, thrown.ExceptionClass, decoded.ExceptionClass)
	assert.Equal(t, thrown.ExceptionMessage, decoded.ExceptionMessage)
}
