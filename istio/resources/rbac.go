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

package resources

import (
	"bytes"

	"dubbo.apache.org/dubbo-go/v3/istio/resources/rbac"
	envoyrbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	jsonp "github.com/golang/protobuf/jsonpb"
)

// RBACEnvoyFilter definine RBAC filter configuration
type RBACEnvoyFilter struct {
	Name string
	RBAC *rbac.RBAC
}

func ParseJsonToRBAC(jsonConf string) (*envoyrbacv3.RBAC, error) {
	rbac := envoyrbacv3.RBAC{}
	un := jsonp.Unmarshaler{
		AllowUnknownFields: true,
	}
	if err := un.Unmarshal(bytes.NewReader([]byte(jsonConf)), &rbac); err != nil {
		return nil, err
	}
	return &rbac, nil
}
