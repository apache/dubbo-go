// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple_protocol

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

import (
	"google.golang.org/protobuf/types/known/durationpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

func TestTripleErrorDetailMarshaling(t *testing.T) {
	t.Parallel()
	detail, err := NewErrorDetail(durationpb.New(time.Second))
	assert.Nil(t, err)
	data, err := json.Marshal((*tripleWireDetail)(detail))
	assert.Nil(t, err)
	t.Logf("marshaled error detail: %s", string(data))

	var unmarshaled tripleWireDetail
	assert.Nil(t, json.Unmarshal(data, &unmarshaled))
	assert.Equal(t, unmarshaled.wireJSON, string(data))
	assert.Equal(t, unmarshaled.pb, detail.pb)
}

func TestTripleErrorDetailMarshalingNoDescriptor(t *testing.T) {
	t.Parallel()
	raw := `{"type":"acme.user.v1.User","value":"DEADBF",` +
		`"debug":{"@type":"acme.user.v1.User","email":"someone@triple.build"}}`
	var detail tripleWireDetail
	assert.Nil(t, json.Unmarshal([]byte(raw), &detail))
	assert.Equal(t, detail.pb.TypeUrl, defaultAnyResolverPrefix+"acme.user.v1.User")

	_, err := (*ErrorDetail)(&detail).Value()
	assert.NotNil(t, err)
	assert.True(t, strings.HasSuffix(err.Error(), "not found"))

	encoded, err := json.Marshal(&detail)
	assert.Nil(t, err)
	assert.Equal(t, string(encoded), raw)
}
