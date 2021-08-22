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
	_ "dubbo.apache.org/dubbo-go/v3/imports/dubbo"
	_ "dubbo.apache.org/dubbo-go/v3/imports/dubbo3"
	_ "dubbo.apache.org/dubbo-go/v3/imports/grpc"
	_ "dubbo.apache.org/dubbo-go/v3/imports/jsonrpc"
	_ "dubbo.apache.org/dubbo-go/v3/imports/rest"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/mapping/dynamic"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/etcd"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/zookeeper"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/service/local"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/service/remote"
	_ "dubbo.apache.org/dubbo-go/v3/registry/servicediscovery"
)
