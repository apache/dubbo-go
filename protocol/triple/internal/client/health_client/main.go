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

package main

import (
	"context"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	health "dubbo.apache.org/dubbo-go/v3/protocol/triple/health/triple_health"
)

func main() {
	cli, err := client.NewClient(
		client.WithClientURL("tri://127.0.0.1:20000"),
	)
	if err != nil {
		panic(err)
	}
	svc, err := health.NewHealth(cli)
	if err != nil {
		panic(err)
	}

	check, err := svc.Check(context.Background(), &health.HealthCheckRequest{Service: "dubbo.health.v1.Health"})
	if err != nil {
		logger.Error(err)
	} else {
		logger.Info("dubbo.health.v1.Health's health", check.String())
	}
	check, err = svc.Check(context.Background(), &health.HealthCheckRequest{Service: "greet.GreetService"})
	if err != nil {
		logger.Error(err)
	} else {
		logger.Info("greet.GreetService's health", check.String())
	}

	watch, err := svc.Watch(context.Background(), &health.HealthCheckRequest{Service: "dubbo.health.v1.Health"})
	if err != nil {
		logger.Error(err)
	} else {
		if watch.Recv() {
			logger.Info("dubbo.health.v1.Health's health", watch.Msg().String())
		}
	}
	watch, err = svc.Watch(context.Background(), &health.HealthCheckRequest{Service: "greet.GreetService"})
	if err != nil {
		logger.Error(err)
	} else {
		if watch.Recv() {
			logger.Info("greet.GreetService's health", watch.Msg().String())
		}
	}
}
