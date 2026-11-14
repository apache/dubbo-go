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

// Package serialization defines protocol-independent serialization contracts.
package serialization

// Codec instances are concurrent-safe. Marshal returns owned bytes; Unmarshal
// must not retain its input. Ordinary invalid values return errors.
type Codec interface {
	Name() string
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
}

type MarshalAppender interface {
	MarshalAppend([]byte, any) ([]byte, error)
}

type Mode uint8

const (
	MessageMode Mode = iota + 1
	WrappedObjectMode
	StreamMode
)

// ConfigProvider returns an independent runtime snapshot after Config.Init.
type ConfigProvider interface{ CodecConfig() any }

type Options struct {
	Protocol   string
	Side       string
	Parameters map[string]string
	Config     any
}
type Factory func(Options) (Codec, error)
type Registration struct {
	Protocol     string
	Name         string
	Mode         Mode
	WireName     string
	ConfigPrefix string
	DubboID      *byte
	New          Factory
}

// ArgumentTypeResolver optionally supplies declared remote class names for
// values without invocation type information (for example external Go structs).
type ArgumentTypeResolver interface{ ArgumentType(any) (string, error) }
