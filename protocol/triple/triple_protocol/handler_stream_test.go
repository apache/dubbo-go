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

//是否需要另行分配？？？---》handler_stream 63
//func TestClientStreamIterator(t *testing.T) {
//	t.Parallel()
//	// The server's view of a client streaming RPC is an iterator. For safety,
//	// and to match grpc-go's behavior, we should allocate a new message for each
//	// iteration.
//	stream := &ClientStream{conn: &nopStreamingHandlerConn{}}
//	assert.True(t, stream.Receive(nil))
//	first := fmt.Sprintf("%p", stream.Msg())
//	assert.True(t, stream.Receive(nil))
//	second := fmt.Sprintf("%p", stream.Msg())
//	assert.NotEqual(t, first, second, assert.Sprintf("should allocate a new message for each iteration"))
//}

type nopStreamingHandlerConn struct {
	StreamingHandlerConn
}

func (nopStreamingHandlerConn) Receive(msg any) error {
	return nil
}
