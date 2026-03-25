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

const (
	OpenAPIVersion30 = "3.0.1"
)

type Info struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}

func NewInfo() *Info {
	return &Info{}
}

func (i *Info) SetTitle(title string) *Info {
	i.Title = title
	return i
}

func (i *Info) SetDescription(desc string) *Info {
	i.Description = desc
	return i
}

func (i *Info) SetVersion(version string) *Info {
	i.Version = version
	return i
}

type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Options *Operation `json:"options,omitempty"`
	Head    *Operation `json:"head,omitempty"`
	Patch   *Operation `json:"patch,omitempty"`
	Trace   *Operation `json:"trace,omitempty"`
}

func NewPathItem() *PathItem {
	return &PathItem{}
}

func (p *PathItem) SetOperation(method string, op *Operation) *PathItem {
	switch method {
	case "GET":
		p.Get = op
	case "PUT":
		p.Put = op
	case "POST":
		p.Post = op
	case "DELETE":
		p.Delete = op
	case "OPTIONS":
		p.Options = op
	case "HEAD":
		p.Head = op
	case "PATCH":
		p.Patch = op
	case "TRACE":
		p.Trace = op
	}
	return p
}

func (p *PathItem) GetOperation(method string) *Operation {
	switch method {
	case "GET":
		return p.Get
	case "PUT":
		return p.Put
	case "POST":
		return p.Post
	case "DELETE":
		return p.Delete
	case "OPTIONS":
		return p.Options
	case "HEAD":
		return p.Head
	case "PATCH":
		return p.Patch
	case "TRACE":
		return p.Trace
	}
	return nil
}

func (p *PathItem) GetOperations() map[string]*Operation {
	ops := make(map[string]*Operation)
	if p.Get != nil {
		ops["GET"] = p.Get
	}
	if p.Put != nil {
		ops["PUT"] = p.Put
	}
	if p.Post != nil {
		ops["POST"] = p.Post
	}
	if p.Delete != nil {
		ops["DELETE"] = p.Delete
	}
	if p.Options != nil {
		ops["OPTIONS"] = p.Options
	}
	if p.Head != nil {
		ops["HEAD"] = p.Head
	}
	if p.Patch != nil {
		ops["PATCH"] = p.Patch
	}
	if p.Trace != nil {
		ops["TRACE"] = p.Trace
	}
	return ops
}

type Components struct {
	Schemas map[string]*Schema `json:"schemas,omitempty"`
}

func NewComponents() *Components {
	return &Components{
		Schemas: make(map[string]*Schema),
	}
}

func (c *Components) AddSchema(name string, schema *Schema) *Components {
	if c.Schemas == nil {
		c.Schemas = make(map[string]*Schema)
	}
	c.Schemas[name] = schema
	return c
}

type OpenAPI struct {
	OpenAPI    string               `json:"openapi"`
	Info       *Info                `json:"info"`
	Paths      map[string]*PathItem `json:"paths,omitempty"`
	Components *Components          `json:"components,omitempty"`

	Group string `json:"-"`
}

func NewOpenAPI() *OpenAPI {
	return &OpenAPI{
		OpenAPI: OpenAPIVersion30,
		Info:    NewInfo(),
	}
}

func (o *OpenAPI) SetOpenAPI(version string) *OpenAPI {
	o.OpenAPI = version
	return o
}

func (o *OpenAPI) SetInfo(info *Info) *OpenAPI {
	o.Info = info
	return o
}

func (o *OpenAPI) SetComponents(components *Components) *OpenAPI {
	o.Components = components
	return o
}

func (o *OpenAPI) SetGroup(group string) *OpenAPI {
	o.Group = group
	return o
}

func (o *OpenAPI) AddPath(path string, item *PathItem) *OpenAPI {
	if o.Paths == nil {
		o.Paths = make(map[string]*PathItem)
	}
	o.Paths[path] = item
	return o
}

func (o *OpenAPI) GetOrAddPath(path string) *PathItem {
	if o.Paths == nil {
		o.Paths = make(map[string]*PathItem)
	}
	if _, ok := o.Paths[path]; !ok {
		o.Paths[path] = NewPathItem()
	}
	return o.Paths[path]
}
