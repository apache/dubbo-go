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

package getty

import (
	"testing"
)

import (
	perrors "github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
)

// The client returns these sentinels wrapped by perrors.WithStack.
func TestClientSentinelClassification(t *testing.T) {
	timeout := perrors.WithStack(errClientReadTimeout)
	assert.Equal(t, rpcerror.KindTimeout, rpcerror.KindOf(timeout))
	assert.Equal(t, "maybe the client read timeout or fail to decode tcp stream in Writer.Write", errClientReadTimeout.Error())

	for _, err := range []error{errSessionNotExist, errClientClosed} {
		assert.Equal(t, rpcerror.Network, rpcerror.Of(perrors.WithStack(err)), err.Error())
		assert.Equal(t, rpcerror.KindUnknown, rpcerror.KindOf(err), err.Error())
		require.ErrorIs(t, perrors.WithStack(err), err, "sentinel identity must be preserved")
	}
	assert.Equal(t, "session not exist", errSessionNotExist.Error())
	assert.Equal(t, "client closed", errClientClosed.Error())
}
