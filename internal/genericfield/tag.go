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

// Package genericfield defines the field-name contract shared by generic
// value conversion and interface-level service definitions.
package genericfield

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Tag is the interpreted form of a struct field's m tag.
type Tag struct {
	Name           string
	Ignore         bool
	OmitEmpty      bool
	Squash         bool
	UnknownOptions []string
}

// ParseMTag interprets the m tag options understood by dubbo-go's generic
// conversion. Unknown options are preserved so schema producers can reject
// options they cannot represent while value conversion remains compatible.
func ParseMTag(field reflect.StructField) Tag {
	tag := Tag{Name: DefaultName(field.Name)}
	tagValue := field.Tag.Get("m")
	name, options, hasOptions := strings.Cut(tagValue, ",")
	if name == "-" {
		tag.Ignore = true
		return tag
	}
	if name != "" {
		tag.Name = name
	}
	if !hasOptions {
		return tag
	}

	for option := range strings.SplitSeq(options, ",") {
		switch option {
		case "":
		case "omitempty":
			tag.OmitEmpty = true
		case "squash":
			tag.Squash = true
		default:
			tag.UnknownOptions = append(tag.UnknownOptions, option)
		}
	}
	return tag
}

// DefaultName returns the generic wire name for an untagged Go field.
func DefaultName(name string) string {
	if name == "" {
		return name
	}
	first, size := utf8.DecodeRuneInString(name)
	return string(unicode.ToLower(first)) + name[size:]
}
