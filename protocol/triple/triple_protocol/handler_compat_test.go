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

package triple_protocol

import (
	"testing"
)

import (
	"github.com/dubbogo/grpc-go/codes"
	"github.com/dubbogo/grpc-go/status"

	"github.com/stretchr/testify/assert"
)

func TestCompatError(t *testing.T) {
	err := status.Error(codes.Code(1234), "user defined")
	triErr, ok := compatError(err)
	assert.True(t, ok)
	assert.Equal(t, Code(1234), triErr.Code())
	assert.Equal(t, "user defined", triErr.Message())
	assert.Equal(t, 1, len(triErr.Details()))
}
