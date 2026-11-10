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

package dubboutil

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// EncodeJSON encodes in as JSON.
func EncodeJSON(in any) ([]byte, error) {
	if in == nil {
		return nil, fmt.Errorf("input for encoding is nil")
	}

	data, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("encode JSON: %w", err)
	}
	return data, nil
}

// DecodeJSON decodes JSON data into out.
func DecodeJSON(data []byte, out any) error {
	if len(data) == 0 {
		return fmt.Errorf("'data' being decoded is nil")
	}
	if out == nil {
		return fmt.Errorf("output parameter 'out' is nil")
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	// While decoding JSON values, interpret the integer values as json.Numbers
	// instead of float64.
	dec.UseNumber()
	return dec.Decode(out)
}
