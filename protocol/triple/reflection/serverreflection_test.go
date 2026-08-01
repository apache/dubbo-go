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

package reflection

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	rpb "dubbo.apache.org/dubbo-go/v3/protocol/triple/reflection/triple_reflection"
)

type recvErrorStream struct {
	rpb.ServerReflection_ServerReflectionInfoServer
	err error
}

func (s recvErrorStream) Recv() (*rpb.ServerReflectionRequest, error) { return nil, s.err }

func TestServerReflectionInfoRecvError(t *testing.T) {
	server := &ReflectionServer{}
	require.NoError(t, server.ServerReflectionInfo(context.Background(), recvErrorStream{err: fmt.Errorf("transport: %w", io.EOF)}))

	recvErr := errors.New("receive failed")
	require.ErrorIs(t, server.ServerReflectionInfo(context.Background(), recvErrorStream{err: recvErr}), recvErr)
}
