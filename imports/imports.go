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

// Package imports is a one-stop collection of Dubbo SPI implementations that aims to help users with plugin installation
// by leveraging Go package initialization.
package imports

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
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/aliasmethod"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/consistenthashing"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/iwrr"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/leastactive"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/p2c"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/roundrobin"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/condition"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/polaris"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/script"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/tag"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/apollo"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/zookeeper"
	_ "dubbo.apache.org/dubbo-go/v3/filter/accesslog"
	_ "dubbo.apache.org/dubbo-go/v3/filter/active"
	_ "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc"
	_ "dubbo.apache.org/dubbo-go/v3/filter/auth"
	_ "dubbo.apache.org/dubbo-go/v3/filter/context"
	_ "dubbo.apache.org/dubbo-go/v3/filter/echo"
	_ "dubbo.apache.org/dubbo-go/v3/filter/exec_limit"
	_ "dubbo.apache.org/dubbo-go/v3/filter/generic"
	_ "dubbo.apache.org/dubbo-go/v3/filter/graceful_shutdown"
	_ "dubbo.apache.org/dubbo-go/v3/filter/hystrix"
	_ "dubbo.apache.org/dubbo-go/v3/filter/metrics"
	_ "dubbo.apache.org/dubbo-go/v3/filter/otel/trace"
	_ "dubbo.apache.org/dubbo-go/v3/filter/polaris/limit"
	_ "dubbo.apache.org/dubbo-go/v3/filter/seata"
	_ "dubbo.apache.org/dubbo-go/v3/filter/sentinel"
	_ "dubbo.apache.org/dubbo-go/v3/filter/token"
	_ "dubbo.apache.org/dubbo-go/v3/filter/tps"
	_ "dubbo.apache.org/dubbo-go/v3/filter/tps/limiter"
	_ "dubbo.apache.org/dubbo-go/v3/filter/tps/strategy"
	_ "dubbo.apache.org/dubbo-go/v3/filter/tracing"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/mapping/metadata"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/etcd"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/zookeeper"
	_ "dubbo.apache.org/dubbo-go/v3/metrics/app_info"
	_ "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	_ "dubbo.apache.org/dubbo-go/v3/otel/trace/jaeger"
	_ "dubbo.apache.org/dubbo-go/v3/otel/trace/otlp"
	_ "dubbo.apache.org/dubbo-go/v3/otel/trace/stdout"
	_ "dubbo.apache.org/dubbo-go/v3/otel/trace/zipkin"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/jsonrpc"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/rest"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/triple"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/triple/health"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/triple/reflection"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	_ "dubbo.apache.org/dubbo-go/v3/registry/directory"
	_ "dubbo.apache.org/dubbo-go/v3/registry/etcdv3"
	_ "dubbo.apache.org/dubbo-go/v3/registry/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/registry/polaris"
	_ "dubbo.apache.org/dubbo-go/v3/registry/protocol"
	_ "dubbo.apache.org/dubbo-go/v3/registry/servicediscovery"
	_ "dubbo.apache.org/dubbo-go/v3/registry/zookeeper"
)
