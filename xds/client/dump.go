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

package client

import (
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

func mergeMaps(maps []map[string]resource.UpdateWithMD) map[string]resource.UpdateWithMD {
	ret := make(map[string]resource.UpdateWithMD)
	for _, m := range maps {
		for k, v := range m {
			ret[k] = v
		}
	}
	return ret
}

func (c *clientImpl) dump(t resource.ResourceType) map[string]resource.UpdateWithMD {
	c.authorityMu.Lock()
	defer c.authorityMu.Unlock()
	maps := make([]map[string]resource.UpdateWithMD, 0, len(c.authorities))
	for _, a := range c.authorities {
		maps = append(maps, a.dump(t))
	}
	return mergeMaps(maps)
}

// DumpLDS returns the status and contents of LDS.
func (c *clientImpl) DumpLDS() map[string]resource.UpdateWithMD {
	return c.dump(resource.ListenerResource)
}

// DumpRDS returns the status and contents of RDS.
func (c *clientImpl) DumpRDS() map[string]resource.UpdateWithMD {
	return c.dump(resource.RouteConfigResource)
}

// DumpCDS returns the status and contents of CDS.
func (c *clientImpl) DumpCDS() map[string]resource.UpdateWithMD {
	return c.dump(resource.ClusterResource)
}

// DumpEDS returns the status and contents of EDS.
func (c *clientImpl) DumpEDS() map[string]resource.UpdateWithMD {
	return c.dump(resource.EndpointsResource)
}
