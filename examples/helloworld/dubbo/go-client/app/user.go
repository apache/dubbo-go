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
	hessian "github.com/dubbogo/hessian2"
)

import (
	"github.com/apache/dubbo-go/config"
)

type Gender hessian.JavaEnum

var (
	userProvider = new(UserProvider)
)

func init() {
	config.SetConsumerService(userProvider)

}

type User struct {
	// !!! Cannot define lowercase names of variable
	Id   string
	Name string
	Age  int32
	Time time.Time
}

func (u User) String() string {
	return fmt.Sprintf(
		"User{Id:%s, Name:%s, Age:%d, Time:%s}",
		u.Id, u.Name, u.Age, u.Time,
	)
}

func (User) JavaClassName() string {
	return "com.ikurento.user.User"
}

type UserProvider struct {
	GetUser func(ctx context.Context, req []interface{}, rsp *User) error
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}
