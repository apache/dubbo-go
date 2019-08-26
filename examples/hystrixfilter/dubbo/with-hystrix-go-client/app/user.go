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
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"github.com/apache/dubbo-go/config"
)

type Gender hessian.JavaEnum

var (
	userProvider  = new(UserProvider)
	userProvider1 = new(UserProvider1)
	userProvider2 = new(UserProvider2)
)

func init() {
	config.SetConsumerService(userProvider)
	config.SetConsumerService(userProvider1)
	config.SetConsumerService(userProvider2)
}

const (
	MAN hessian.JavaEnum = iota
	WOMAN
)

var genderName = map[hessian.JavaEnum]string{
	MAN:   "MAN",
	WOMAN: "WOMAN",
}

var genderValue = map[string]hessian.JavaEnum{
	"MAN":   MAN,
	"WOMAN": WOMAN,
}

func (g Gender) JavaClassName() string {
	return "com.ikurento.user.Gender"
}

func (g Gender) String() string {
	s, ok := genderName[hessian.JavaEnum(g)]
	if ok {
		return s
	}

	return strconv.Itoa(int(g))
}

func (g Gender) EnumValue(s string) hessian.JavaEnum {
	v, ok := genderValue[s]
	if ok {
		return v
	}

	return hessian.InvalidJavaEnum
}

type User struct {
	// !!! Cannot define lowercase names of variable
	Id   string
	Name string
	Age  int32
	Time time.Time
	Sex  Gender // 注意此处，java enum Object <--> go string
}

func (u User) String() string {
	return fmt.Sprintf(
		"User{Id:%s, Name:%s, Age:%d, Time:%s, Sex:%s}",
		u.Id, u.Name, u.Age, u.Time, u.Sex,
	)
}

func (User) JavaClassName() string {
	return "com.ikurento.user.User"
}

type UserProvider struct {
	GetUsers func(req []interface{}) ([]interface{}, error)
	GetErr   func(ctx context.Context, req []interface{}, rsp *User) error
	GetUser  func(ctx context.Context, req []interface{}, rsp *User) error
	GetUser0 func(id string, name string) (User, error)
	GetUser1 func(ctx context.Context, req []interface{}, rsp *User) error
	GetUser2 func(ctx context.Context, req []interface{}, rsp *User) error `dubbo:"getUser"`
	GetUser3 func() error
	Echo     func(ctx context.Context, req interface{}) (interface{}, error) // Echo represent EchoFilter will be used
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}

type UserProvider1 struct {
	GetUsers func(req []interface{}) ([]interface{}, error)
	GetErr   func(ctx context.Context, req []interface{}, rsp *User) error
	GetUser  func(ctx context.Context, req []interface{}, rsp *User) error
	GetUser0 func(id string, name string) (User, error)
	GetUser1 func(ctx context.Context, req []interface{}, rsp *User) error
	GetUser2 func(ctx context.Context, req []interface{}, rsp *User) error `dubbo:"getUser"`
	GetUser3 func() error
	Echo     func(ctx context.Context, req interface{}) (interface{}, error) // Echo represent EchoFilter will be used
}

func (u *UserProvider1) Reference() string {
	return "UserProvider1"
}

type UserProvider2 struct {
	GetUsers func(req []interface{}) ([]interface{}, error)
	GetErr   func(ctx context.Context, req []interface{}, rsp *User) error
	GetUser  func(ctx context.Context, req []interface{}, rsp *User) error
	GetUser0 func(id string, name string) (User, error)
	GetUser1 func(ctx context.Context, req []interface{}, rsp *User) error
	GetUser2 func(ctx context.Context, req []interface{}, rsp *User) error `dubbo:"getUser"`
	GetUser3 func() error
	Echo     func(ctx context.Context, req interface{}) (interface{}, error) // Echo represent EchoFilter will be used
}

func (u *UserProvider2) Reference() string {
	return "UserProvider2"
}
