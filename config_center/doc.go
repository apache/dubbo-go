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

// Package config_center provides Config Center definition and implementations
// for listening service governance rules.
//
// When a config center is enabled, Dubbo-go subscribes to dynamic governance
// rule keys for both application-level and service-level configuration. These
// keys use the configured application or service key plus the configurators
// suffix. For example, an application named "dubbo.io" may subscribe to
// "dubbo.io.configurators", and a service interface named "dubbo.model" may
// subscribe to "dubbo.model.configurators".
//
// These governance rules are optional. If the config center reports that such
// a key does not exist, it means no dynamic override rule has been configured
// for that application or service. The absence of these optional keys should
// not be treated as an application startup failure.
package config_center
