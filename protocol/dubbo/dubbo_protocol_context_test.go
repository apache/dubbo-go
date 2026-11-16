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
 * See the License for the specific language regarding permissions and
 * limitations under the License.
 */

package dubbo

import (
	"context"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestRebuildCtxUsesInvocationParent(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	inv := invocation.NewRPCInvocation("Echo", nil, map[string]any{"k": "v"})
	inv.SetContext(parent)

	ctx := rebuildCtx(inv)
	attachments, ok := ctx.Value(constant.DubboCtxKey("attachment")).(map[string]any)
	require.True(t, ok)
	require.Equal(t, "v", attachments["k"])
	require.NoError(t, ctx.Err())

	cancel()
	select {
	case <-ctx.Done():
		require.ErrorIs(t, ctx.Err(), context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("rebuild context was not canceled with its parent")
	}
}

func TestRebuildCtxWithoutParentStillHasAttachments(t *testing.T) {
	inv := invocation.NewRPCInvocation("Echo", nil, map[string]any{"k": "v"})
	ctx := rebuildCtx(inv)
	attachments, ok := ctx.Value(constant.DubboCtxKey("attachment")).(map[string]any)
	require.True(t, ok)
	require.Equal(t, "v", attachments["k"])
	require.NoError(t, ctx.Err())
}
