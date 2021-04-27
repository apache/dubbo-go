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
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go/config"
)

func init() {
	config.SetProviderService(new(UserProvider))
	// ------for hessian2------
	hessian.RegisterPOJO(&User{})
	hessian.RegisterPOJO(&CallUserStruct{})
}

type UserProvider struct{}

func (u *UserProvider) GetUser(ctx context.Context, userStruct *CallUserStruct) (*User, error) {
	fmt.Printf("=======================\nreq:%#v\n", userStruct)
	rsp := User{"A002", "Alex Stocks", 18, userStruct.SubInfo}
	fmt.Printf("=======================\nrsp:%#v\n", rsp)
	return &rsp, nil
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}

type User struct {
	ID      string
	Name    string
	Age     int32
	SubInfo SubInfo
}

func (u *User) JavaClassName() string {
	return "com.ikurento.user.User"
}

type CallUserStruct struct {
	ID      string
	Male    bool
	SubInfo SubInfo
}

type SubInfo struct {
	SubID   string
	SubMale bool
	SubAge  int
}

func (s SubInfo) JavaClassName() string {
	return "com.ikurento.user.SubInfo"
}

func (cs CallUserStruct) JavaClassName() string {
	return "com.ikurento.user.CallUserStruct"
}
