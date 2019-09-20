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
	"strconv"
)

import (
	"github.com/apache/dubbo-go-hessian2/java_exception"
	"github.com/apache/dubbo-go/config"
	perrors "github.com/pkg/errors"
)

func init() {
	config.SetProviderService(new(UserProvider))
}

type UserProvider struct {
}

func (u *UserProvider) getUser(userId string) (*User, error) {
	if user, ok := userMap[userId]; ok {
		return &user, nil
	}

	return nil, fmt.Errorf("invalid user id:%s", userId)
}

func (u *UserProvider) GetUser(ctx context.Context, req []interface{}, rsp *User) error {
	var (
		err  error
		user *User
	)

	println("req:%#v", req)
	user, err = u.getUser(req[0].(string))
	if err == nil {
		*rsp = *user
		println("rsp:%#v", rsp)
	}
	return err
}

func (u *UserProvider) GetUser0(id string, name string) (User, error) {
	var err error

	println("id:%s, name:%s", id, name)
	user, err := u.getUser(id)
	if err != nil {
		return User{}, err
	}
	if user.Name != name {
		return User{}, perrors.New("name is not " + user.Name)
	}
	return *user, err
}

func (u *UserProvider) GetUser2(ctx context.Context, req []interface{}, rsp *User) error {
	var err error

	println("req:%#v", req)
	rsp.Id = strconv.Itoa(int(req[0].(int32)))
	return err
}

func (u *UserProvider) GetUser3() error {
	return nil
}

func (u *UserProvider) GetErr(ctx context.Context, req []interface{}, rsp *User) error {
	return java_exception.NewThrowable("exception")
}

func (u *UserProvider) GetUsers(req []interface{}) ([]interface{}, error) {
	var err error

	println("req:%s", req)
	t := req[0].([]interface{})
	user, err := u.getUser(t[0].(string))
	if err != nil {
		return nil, err
	}
	println("user:%v", user)
	user1, err := u.getUser(t[1].(string))
	if err != nil {
		return nil, err
	}
	println("user1:%v", user1)

	return []interface{}{user, user1}, err
}

func (s *UserProvider) MethodMapper() map[string]string {
	return map[string]string{
		"GetUser2": "getUser",
	}
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}
