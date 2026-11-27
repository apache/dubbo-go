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

package base

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

// pointerInvocation and valueInvocation only provide an identity.
type pointerInvocation struct{ Invocation }

type valueInvocation struct {
	Invocation
	name string
}

func TestRetryAttemptNamesTheInvocation(t *testing.T) {
	retried, child := &pointerInvocation{}, &pointerInvocation{}
	ctx := WithRetryAttempt(context.Background(), retried)

	assert.False(t, IsRetryAttempt(context.Background(), retried))
	assert.True(t, IsRetryAttempt(ctx, retried))
	assert.False(t, IsRetryAttempt(ctx, child), "another RPC started with the context is not a retry")

	assert.True(t, IsRetryAttempt(TransferRetryAttempt(ctx, retried, child), child), "a translated invocation takes over the mark")
	assert.False(t, IsRetryAttempt(TransferRetryAttempt(context.Background(), retried, child), child))

	value := valueInvocation{name: "v"}
	assert.False(t, IsRetryAttempt(WithRetryAttempt(context.Background(), value), value), "values have no identity")
}
