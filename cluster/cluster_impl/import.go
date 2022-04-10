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

// Package cluster_impl is for being compatible with older dubbo-go, please use `imports` package.
// It may be DEPRECATED OR REMOVED in the future.
package cluster_impl

import (
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/adaptivesvc"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/available"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/broadcast"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/failback"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/failfast"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/failover"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/failsafe"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/forking"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/zoneaware"
)
