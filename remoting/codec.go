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

package remoting

import (
	"bytes"
	"sync"
)

// Codec is the interface that wrap EncodeRequest、 EncodeResponse and Decode method
// for exchangeClient.
type Codec interface {
	EncodeRequest(request *Request) (*bytes.Buffer, error)
	EncodeResponse(response *Response) (*bytes.Buffer, error)
	// Decode returns one frame and its consumed length. For an incomplete header
	// it returns nil, 0, nil; with a complete header and partial body, nil and
	// the required total frame length. A complete frame followed by more bytes
	// consumes only that frame. Errors never identify a successfully decoded frame.
	Decode(data []byte) (*DecodeResult, int, error)
}

type DecodeResult struct {
	IsRequest bool // indicates whether the current request is a heartbeat request
	Result    any
}

var codec = make(map[string]Codec, 2)
var codecMu sync.RWMutex
var legacyCodecOverrides = make(map[string]Codec)

func RegistryCodec(protocol string, codecTmp Codec) {
	codecMu.Lock()
	defer codecMu.Unlock()
	codec[protocol] = codecTmp
	codecFactories.RLock()
	_, namedDefault := codecFactories.items[protocol+"/"+protocol]
	codecFactories.RUnlock()
	if namedDefault {
		legacyCodecOverrides[protocol] = codecTmp
	}
}

func GetCodec(protocol string) Codec {
	codecMu.RLock()
	defer codecMu.RUnlock()
	return codec[protocol]
}
