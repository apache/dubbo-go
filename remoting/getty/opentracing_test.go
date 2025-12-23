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

package getty

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestFilterContext(t *testing.T) {
	attachments := map[string]any{
		"string-key": "string-value",
		"int-key":    123,
		"bool-key":   true,
	}

	result := filterContext(attachments)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, "string-value", result["string-key"])
}

func TestFillTraceAttachments(t *testing.T) {
	attachments := map[string]any{"existing": "value"}
	traceAttachment := map[string]string{"trace-id": "123", "span-id": "456"}

	fillTraceAttachments(attachments, traceAttachment)

	assert.Equal(t, "value", attachments["existing"])
	assert.Equal(t, "123", attachments["trace-id"])
	assert.Equal(t, "456", attachments["span-id"])
}
