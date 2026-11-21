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

package rpcerror

// Category is the stable, low-cardinality cross-signal failure taxonomy.
type Category uint8

const (
	CategoryUnknown Category = iota
	CategoryTimeout
	CategoryLimit
	CategoryServiceUnavailable
	CategoryBusiness
	CategoryNetwork
	CategoryCodec
)

func (c Category) String() string {
	switch c {
	case CategoryTimeout:
		return "timeout"
	case CategoryLimit:
		return "limit"
	case CategoryServiceUnavailable:
		return "service_unavailable"
	case CategoryBusiness:
		return "business"
	case CategoryNetwork:
		return "network"
	case CategoryCodec:
		return "codec"
	default:
		return "unknown"
	}
}

func (o Origin) String() string {
	switch o {
	case Network:
		return "network"
	case Codec:
		return "codec"
	default:
		return "unknown"
	}
}

// TripleStatus is implemented by protocol adapters, keeping this package free
// of protocol imports. Codes use the Triple/gRPC wire numbering.
type TripleStatus interface{ RPCTripleCode() uint32 }

type Classification struct {
	Category     Category
	Origin       Origin
	ProtocolCode uint32
}

// Classify uses status, local origin, producer kind, then Unavailable evidence.
// Conflicting evidence is sticky across the entire error tree; text is never parsed.
func Classify(err error) Classification {
	c := Classification{Origin: Of(err)}
	if isNil(err) {
		return c
	}
	var code uint32
	var conflict bool
	var visit func(error)
	visit = func(e error) {
		if isNil(e) {
			return
		}
		if s, ok := e.(TripleStatus); ok {
			n := s.RPCTripleCode()
			if code != 0 && code != n {
				conflict = true
			}
			code = n
		}
		switch e := e.(type) {
		case interface{ Unwrap() []error }:
			for _, child := range e.Unwrap() {
				visit(child)
			}
		case interface{ Unwrap() error }:
			visit(e.Unwrap())
		}
	}
	visit(err)
	var origin Origin
	collectOrigins(err, &origin)
	if conflict || origin == Origin(255) || declaredKind(err) == kindConflict {
		c.Origin = Unknown
		return c
	}
	c.ProtocolCode = code
	switch code {
	case 4:
		c.Category = CategoryTimeout
		return c
	case 8:
		c.Category = CategoryLimit
		return c
	case 7:
		c.Category = CategoryServiceUnavailable
		return c
	case 17:
		c.Category = CategoryBusiness
		return c
	}
	switch c.Origin {
	case Network:
		c.Category = CategoryNetwork
		return c
	case Codec:
		c.Category = CategoryCodec
		return c
	}
	switch KindOf(err) {
	case KindTimeout:
		c.Category = CategoryTimeout
	case KindLimit:
		c.Category = CategoryLimit
	case KindServiceUnavailable:
		c.Category = CategoryServiceUnavailable
	case KindBusiness:
		c.Category = CategoryBusiness
	case KindCodec:
		c.Category = CategoryCodec
	default:
		if code == 14 {
			c.Category = CategoryServiceUnavailable
		}
	}
	return c
}

// Classify combines a terminal error with evidence captured at completion.
func (c Completion) Classify() Classification {
	return Classify(WithKind(Mark(c.Err, c.Origin), c.Kind))
}
