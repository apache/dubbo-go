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
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	_struct "github.com/golang/protobuf/ptypes/struct"
)

import (
	"dubbo.apache.org/dubbo-go/v3/xds/client/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/xds/client/controller"
	"dubbo.apache.org/dubbo-go/v3/xds/client/load"
	"dubbo.apache.org/dubbo-go/v3/xds/client/pubsub"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

type controllerInterface interface {
	AddWatch(resourceType resource.ResourceType, resourceName string)
	RemoveWatch(resourceType resource.ResourceType, resourceName string)
	ReportLoad(server string) (*load.Store, func())
	SetMetadata(m *_struct.Struct) error
	Close()
}

var newController = func(config *bootstrap.ServerConfig, pubsub *pubsub.Pubsub, validator resource.UpdateValidatorFunc, logger dubbogoLogger.Logger) (controllerInterface, error) {
	return controller.New(config, pubsub, validator, logger)
}
