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

package model

// Parameter describes a value supplied outside an OpenAPI request body.
type Parameter struct {
	Name          string  `json:"name"`
	In            string  `json:"in"`
	Required      bool    `json:"required,omitempty"`
	Description   string  `json:"description,omitempty"`
	Schema        *Schema `json:"schema,omitempty"`
	AllowReserved bool    `json:"allowReserved,omitempty"`
}

func NewParameter(name, location string) *Parameter {
	return &Parameter{Name: name, In: location}
}

func (p *Parameter) SetRequired(required bool) *Parameter {
	p.Required = required
	return p
}

func (p *Parameter) SetSchema(schema *Schema) *Parameter {
	p.Schema = schema
	return p
}

func (p *Parameter) SetDescription(description string) *Parameter {
	p.Description = description
	return p
}

func (p *Parameter) SetAllowReserved(allowReserved bool) *Parameter {
	p.AllowReserved = allowReserved
	return p
}
