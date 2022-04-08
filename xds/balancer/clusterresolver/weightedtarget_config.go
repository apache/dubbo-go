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

/*
 *
 * Copyright 2020 gRPC authors.
 *
 */

package clusterresolver

import (
	"encoding/json"
)

import (
	"google.golang.org/grpc/serviceconfig"
)

import (
	internalserviceconfig "dubbo.apache.org/dubbo-go/v3/xds/utils/serviceconfig"
)

// Target represents one target with the weight and the child policy.
type Target struct {
	// Weight is the weight of the child policy.
	Weight uint32 `json:"weight,omitempty"`
	// ChildPolicy is the child policy and it's config.
	ChildPolicy *internalserviceconfig.BalancerConfig `json:"childPolicy,omitempty"`
}

// LBConfig is the balancer config for weighted_target.
type TargetLBConfig struct {
	serviceconfig.LoadBalancingConfig `json:"-"`

	Targets map[string]Target `json:"targets,omitempty"`
}

func parseTargetConfig(c json.RawMessage) (*LBConfig, error) {
	var cfg LBConfig
	if err := json.Unmarshal(c, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
