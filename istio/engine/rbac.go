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

package engine

import (
	"dubbo.apache.org/dubbo-go/v3/istio/resources/rbac"
)

type RBACResult struct {
	ReqOK           bool
	MatchPolicyName string
}

type RBACFilterEngine struct {
	RBAC *rbac.RBAC
}

func NewRBACFilterEngine(rbac *rbac.RBAC) *RBACFilterEngine {
	rbacFilterEngine := &RBACFilterEngine{
		RBAC: rbac,
	}
	return rbacFilterEngine
}

func (r *RBACFilterEngine) Filter(headers map[string]string) (*RBACResult, error) {
	var allowed bool
	var matchPolicyName string
	var err error

	// rbac shadow rules handle only for testing only
	//if r.RBAC.ShadowRules != nil {
	//	allowed, matchPolicyName, err = r.RBAC.ShadowRules.Match(headers)
	//}

	// rbac rules handle
	if r.RBAC.Rules != nil {
		allowed, matchPolicyName, err = r.RBAC.Rules.Match(headers)
	} else {
		allowed = true
		err = nil
	}

	return &RBACResult{
		ReqOK:           allowed,
		MatchPolicyName: matchPolicyName,
	}, err
}
