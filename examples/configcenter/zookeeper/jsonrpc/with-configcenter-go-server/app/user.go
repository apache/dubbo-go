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
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/config"
)

type Gender int

func init() {
	config.SetProviderService(new(UserProvider))
}

const (
	MAN = iota
	WOMAN
)

var genderStrings = [...]string{
	"MAN",
	"WOMAN",
}

func (g Gender) String() string {
	return genderStrings[g]
}

type (
	User struct {
		Id    string `json:"id"`
		Name  string `json:"name"`
		Age   int    `json:"age"`
		sex   Gender
		Birth int    `json:"time"`
		Sex   string `json:"sex"`
	}

	UserProvider struct {
		user map[string]User
	}
)

var (
	DefaultUser = User{
		Id: "0", Name: "Alex Stocks", Age: 31,
		// Birth: int(time.Date(1985, time.November, 10, 23, 0, 0, 0, time.UTC).Unix()),
		Birth: int(time.Date(1985, 11, 24, 15, 15, 0, 0, time.Local).Unix()),
		sex:   Gender(MAN),
	}

	userMap = UserProvider{user: make(map[string]User)}
)

func init() {
	DefaultUser.Sex = DefaultUser.sex.String()
	userMap.user["A000"] = DefaultUser
	userMap.user["A001"] = User{Id: "001", Name: "ZhangSheng", Age: 18, sex: MAN}
	userMap.user["A002"] = User{Id: "002", Name: "Lily", Age: 20, sex: WOMAN}
	userMap.user["A003"] = User{Id: "113", Name: "Moorse", Age: 30, sex: MAN}
	for k, v := range userMap.user {
		v.Birth = int(time.Now().AddDate(-1*v.Age, 0, 0).Unix())
		v.Sex = userMap.user[k].sex.String()
		userMap.user[k] = v
	}
}

func (u *UserProvider) getUser(userId string) (*User, error) {
	if user, ok := userMap.user[userId]; ok {
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

func (u *UserProvider) GetUsers(req []interface{}) ([]User, error) {
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

	return []User{*user, *user1}, err
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}

func println(format string, args ...interface{}) {
	fmt.Printf("\033[32;40m"+format+"\033[0m\n", args...)
}
