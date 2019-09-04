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
	"fmt"
	"time"
)

import (
	"github.com/apache/dubbo-go-hessian2"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	_ "github.com/apache/dubbo-go/protocol/dubbo"
	_ "github.com/apache/dubbo-go/registry/protocol"

	_ "github.com/apache/dubbo-go/filter/impl"

	_ "github.com/apache/dubbo-go/cluster/cluster_impl"
	_ "github.com/apache/dubbo-go/cluster/loadbalance"
	_ "github.com/apache/dubbo-go/registry/zookeeper"
)

// they are necessary:
// 		export CONF_CONSUMER_FILE_PATH="xxx"
// 		export APP_LOG_CONF_FILE="xxx"
func main() {

	hessian.RegisterJavaEnum(Gender(MAN))
	hessian.RegisterJavaEnum(Gender(WOMAN))
	hessian.RegisterPOJO(&User{})
	getUserChan := make(chan string, 32)
	getErrChan := make(chan string, 32)
	getUser1Chan := make(chan string, 32)

	config.Load()
	logger.Debugf("[Start to test GetUser]")
	for i := 0; i < 32; i++ {
		go func() {
			user := &User{}
			err := userProvider.GetUser(context.TODO(), []interface{}{"A003"}, user)
			getUserChan <- fmt.Sprintf("Result: %s ; Error: %v", user.Name, err)
		}()
	}
	time.Sleep(time.Second * 4)

	logger.Debugf("[Start to test GetErr, without error whitelist]")
	for i := 0; i < 32; i++ {
		go func() {
			user := &User{}
			err := userProvider.GetErr(context.TODO(), []interface{}{"A003"}, user)
			getErrChan <- fmt.Sprintf("Result: %s ; Error: %v", user.Name, err)
		}()
	}
	time.Sleep(time.Second * 4)

	logger.Debugf("[Start to test illegal method GetUser1, with error whitelist]")
	for i := 0; i < 32; i++ {
		go func() {
			user := &User{}
			err := userProvider.GetUser1(context.TODO(), []interface{}{"A003"}, user)
			getUser1Chan <- fmt.Sprintf("Result: %s ; Error: %v", user.Name, err)
		}()
	}
	time.Sleep(time.Second * 4)
	for i := 1; i < 32; i++ {
		resGot := <-getUserChan
		logger.Infof("[GetUser] %v", resGot)
	}
	for i := 1; i < 32; i++ {
		resGot := <-getErrChan
		logger.Infof("[GetErr] %v", resGot)
	}
	for i := 1; i < 32; i++ {
		resGot := <-getUser1Chan
		logger.Infof("[GetUser1] %v", resGot)
	}
}
