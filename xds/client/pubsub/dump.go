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
 * Copyright 2021 gRPC authors.
 *
 */

package pubsub

import (
	anypb "github.com/golang/protobuf/ptypes/any"
)

import (
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

func rawFromCache(s string, cache interface{}) *anypb.Any {
	switch c := cache.(type) {
	case map[string]resource.ListenerUpdate:
		if v, ok := c[s]; ok {
			return v.Raw
		}
		return nil
	case map[string]resource.RouteConfigUpdate:
		if v, ok := c[s]; ok {
			return v.Raw
		}
		return nil
	case map[string]resource.ClusterUpdate:
		if v, ok := c[s]; ok {
			return v.Raw
		}
		return nil
	case map[string]resource.EndpointsUpdate:
		if v, ok := c[s]; ok {
			return v.Raw
		}
		return nil
	default:
		return nil
	}
}

// Dump dumps the resource for the given type.
func (pb *Pubsub) Dump(t resource.ResourceType) map[string]resource.UpdateWithMD {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	var (
		md    map[string]resource.UpdateMetadata
		cache interface{}
	)
	switch t {
	case resource.ListenerResource:
		md = pb.ldsMD
		cache = pb.ldsCache
	case resource.RouteConfigResource:
		md = pb.rdsMD
		cache = pb.rdsCache
	case resource.ClusterResource:
		md = pb.cdsMD
		cache = pb.cdsCache
	case resource.EndpointsResource:
		md = pb.edsMD
		cache = pb.edsCache
	default:
		pb.logger.Errorf("dumping resource of unknown type: %v", t)
		return nil
	}

	ret := make(map[string]resource.UpdateWithMD, len(md))
	for s, md := range md {
		ret[s] = resource.UpdateWithMD{
			MD:  md,
			Raw: rawFromCache(s, cache),
		}
	}
	return ret
}
