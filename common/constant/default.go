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

package constant

const (
	DUBBO             = "dubbo"
	PROVIDER_PROTOCOL = "provider"
	//compatible with 2.6.x
	OVERRIDE_PROTOCOL = "override"
	EMPTY_PROTOCOL    = "empty"
	ROUTER_PROTOCOL   = "router"
)

const (
	DEFAULT_WEIGHT = 100     //
	DEFAULT_WARMUP = 10 * 60 // in java here is 10*60*1000 because of System.currentTimeMillis() is measured in milliseconds & in go time.Unix() is second
)

const (
	DEFAULT_LOADBALANCE        = "random"
	DEFAULT_RETRIES            = "2"
	DEFAULT_RETRIES_INT        = 2
	DEFAULT_PROTOCOL           = "dubbo"
	DEFAULT_REG_TIMEOUT        = "10s"
	DEFAULT_CLUSTER            = "failover"
	DEFAULT_FAILBACK_TIMES     = "3"
	DEFAULT_FAILBACK_TIMES_INT = 3
	DEFAULT_FAILBACK_TASKS     = 100
)

const (
	DEFAULT_KEY               = "default"
	PREFIX_DEFAULT_KEY        = "default."
	DEFAULT_SERVICE_FILTERS   = "echo,token"
	DEFAULT_REFERENCE_FILTERS = ""
	GENERIC_REFERENCE_FILTERS = "generic"
	GENERIC                   = "$invoke"
	ECHO                      = "$echo"
)

const (
	ANY_VALUE           = "*"
	ANYHOST_VALUE       = "0.0.0.0"
	REMOVE_VALUE_PREFIX = "-"
)

const (
	CONFIGURATORS_CATEGORY             = "configurators"
	ROUTER_CATEGORY                    = "category"
	DEFAULT_CATEGORY                   = PROVIDER_CATEGORY
	DYNAMIC_CONFIGURATORS_CATEGORY     = "dynamicconfigurators"
	APP_DYNAMIC_CONFIGURATORS_CATEGORY = "appdynamicconfigurators"
	PROVIDER_CATEGORY                  = "providers"
)
