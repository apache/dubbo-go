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

package rest_interface

type RestConfig struct {
	InterfaceName        string              `required:"true"  yaml:"interface"  json:"interface,omitempty" property:"interface"`
	Url                  string              `yaml:"url"  json:"url,omitempty" property:"url"`
	Path                 string              `yaml:"rest_path"  json:"rest_path,omitempty" property:"rest_path"`
	Produces             string              `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes             string              `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	MethodType           string              `yaml:"rest_method"  json:"rest_method,omitempty" property:"rest_method"`
	Client               string              `yaml:"rest_client" json:"rest_client,omitempty" property:"rest_client"`
	Server               string              `yaml:"rest_server" json:"rest_server,omitempty" property:"rest_server"`
	RestMethodConfigs    []*RestMethodConfig `yaml:"methods" json:"methods,omitempty" property:"methods"`
	RestMethodConfigsMap map[string]*RestMethodConfig
}

type RestConsumerConfig struct {
	Client        string                 `default:"resty" yaml:"rest_client" json:"rest_client,omitempty" property:"rest_client"`
	Produces      string                 `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes      string                 `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	RestConfigMap map[string]*RestConfig `yaml:"references" json:"references,omitempty" property:"references"`
}

type RestProviderConfig struct {
	Server        string                 `default:"go-restful" yaml:"rest_server" json:"rest_server,omitempty" property:"rest_server"`
	Produces      string                 `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes      string                 `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	RestConfigMap map[string]*RestConfig `yaml:"services" json:"services,omitempty" property:"services"`
}

type RestMethodConfig struct {
	InterfaceName  string
	MethodName     string `required:"true" yaml:"name"  json:"name,omitempty" property:"name"`
	Url            string `yaml:"url"  json:"url,omitempty" property:"url"`
	Path           string `yaml:"rest_path"  json:"rest_path,omitempty" property:"rest_path"`
	Produces       string `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes       string `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	MethodType     string `yaml:"rest_method"  json:"rest_method,omitempty" property:"rest_method"`
	PathParams     string `yaml:"rest_path_params"  json:"rest_path_params,omitempty" property:"rest_path_params"`
	PathParamsMap  map[int]string
	QueryParams    string `yaml:"rest_query_params"  json:"rest_query_params,omitempty" property:"rest_query_params"`
	QueryParamsMap map[int]string
	Body           int    `yaml:"rest_body"  json:"rest_body,omitempty" property:"rest_body"`
	Headers        string `yaml:"rest_headers"  json:"rest_headers,omitempty" property:"rest_headers"`
	HeadersMap     map[int]string
}
