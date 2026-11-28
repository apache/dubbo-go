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

package logger

import (
	"context"
	"errors"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestSpanErrorFromContext(t *testing.T) {
	err := errors.New("rejected")

	assert.Equal(t, err, SpanErrorFromContext(WithSpanError(context.Background(), err)))
	assert.Equal(t, err, SpanErrorFromContext(WithSpanError(nil, err))) //nolint:staticcheck // nil context is handled
	assert.NoError(t, SpanErrorFromContext(context.Background()))
	assert.NoError(t, SpanErrorFromContext(nil)) //nolint:staticcheck // nil context is handled
}
