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

package imports

import (
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster_impl"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	_ "dubbo.apache.org/dubbo-go/v3/common/proxy/proxy_factory"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/apollo"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/zookeeper"
	_ "dubbo.apache.org/dubbo-go/v3/filter/filter_impl"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/mapping/metadata"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/etcd"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/zookeeper"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/service/exporter/configurable"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/service/local"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/service/remote"
	_ "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo3"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/grpc"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/jsonrpc"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/rest"
	_ "dubbo.apache.org/dubbo-go/v3/registry/etcdv3"
	_ "dubbo.apache.org/dubbo-go/v3/registry/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/registry/protocol"
	_ "dubbo.apache.org/dubbo-go/v3/registry/servicediscovery"
	_ "dubbo.apache.org/dubbo-go/v3/registry/zookeeper"
)
