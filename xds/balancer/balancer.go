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

// Package balancer installs all the xds balancers.
package balancer

import (
	_ "google.golang.org/grpc/balancer/weightedtarget" // Register the weighted_target balancer
)

import (
	_ "dubbo.apache.org/dubbo-go/v3/xds/balancer/cdsbalancer"     // Register the CDS balancer
	_ "dubbo.apache.org/dubbo-go/v3/xds/balancer/clusterimpl"     // Register the xds_cluster_impl balancer
	_ "dubbo.apache.org/dubbo-go/v3/xds/balancer/clustermanager"  // Register the xds_cluster_manager balancer
	_ "dubbo.apache.org/dubbo-go/v3/xds/balancer/clusterresolver" // Register the xds_cluster_resolver balancer
	_ "dubbo.apache.org/dubbo-go/v3/xds/balancer/priority"        // Register the priority balancer
)
