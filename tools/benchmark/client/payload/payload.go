/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package payload

import (
	"crypto/rand"
	"sync"
)

type PayloadGenerator struct {
	payloads sync.Map
}

func NewPayloadGenerator() *PayloadGenerator {
	return &PayloadGenerator{}
}

func (pg *PayloadGenerator) Generate(size int) []byte {
	if cached, ok := pg.payloads.Load(size); ok {
		return cached.([]byte)
	}

	data := make([]byte, size)
	_, err := rand.Read(data)
	if err != nil {
		for i := range data {
			data[i] = byte(i % 256)
		}
	}

	pg.payloads.Store(size, data)
	return data
}

func (pg *PayloadGenerator) GetCachedSize(size int) ([]byte, bool) {
	data, ok := pg.payloads.Load(size)
	if !ok {
		return nil, false
	}
	return data.([]byte), true
}

func (pg *PayloadGenerator) Clear() {
	pg.payloads.Range(func(key, value any) bool {
		pg.payloads.Delete(key)
		return true
	})
}
