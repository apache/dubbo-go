// Copyright 2016-2019 Yincheng Fang, Alex Stocks
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
	"github.com/AlexStocks/goext/time"
	"github.com/dubbo/go-for-apache-dubbo/config"
)

type Gender int

func init() {
	config.SetProService(new(UserProvider))
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

	UserId struct {
		Id string
	}

	UserProvider struct {
		user map[string]User
	}
)

var (
	DefaultUser = User{
		Id: "0", Name: "Alex Stocks", Age: 31,
		// Birth: int(time.Date(1985, time.November, 10, 23, 0, 0, 0, time.UTC).Unix()),
		Birth: gxtime.YMD(1985, 11, 24, 15, 15, 0),
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

/*
// you can define your json unmarshal function here
func (u *UserId) UnmarshalJSON(value []byte) error {
	u.Id = string(value)
	u.Id = strings.TrimPrefix(u.Id, "\"")
	u.Id = strings.TrimSuffix(u.Id, `"`)

	return nil
}
*/

func (u *UserProvider) getUser(userId string) (*User, error) {
	if user, ok := userMap.user[userId]; ok {
		return &user, nil
	}

	return nil, fmt.Errorf("invalid user id:%s", userId)
}

/*
// can not work
func (u *UserProvider) GetUser(ctx context.Context, req *UserId, rsp *User) error {
	var (
		err  error
		user *User
	)
	user, err = u.getUser(req.Id)
	if err == nil {
		*rsp = *user
		gxlog.CInfo("rsp:%#v", rsp)
		// s, _ := json.Marshal(rsp)
		// fmt.Println(string(s))

		// s, _ = json.Marshal(*rsp)
		// fmt.Println(string(s))
	}
	return err
}
*/

/*
// work
func (u *UserProvider) GetUser(ctx context.Context, req *string, rsp *User) error {
	var (
		err  error
		user *User
	)

	gxlog.CInfo("req:%#v", *req)
	user, err = u.getUser(*req)
	if err == nil {
		*rsp = *user
		gxlog.CInfo("rsp:%#v", rsp)
		// s, _ := json.Marshal(rsp)
		// fmt.Println(string(s))

		// s, _ = json.Marshal(*rsp)
		// fmt.Println(string(s))
	}
	return err
}
*/

func (u *UserProvider) GetUser(ctx context.Context, req []interface{}, rsp *User) error {
	var (
		err  error
		user *User
	)

	gxlog.CInfo("req:%#v", req)
	user, err = u.getUser(req[0].(string))
	if err == nil {
		*rsp = *user
		gxlog.CInfo("rsp:%#v", rsp)
		// s, _ := json.Marshal(rsp)
		// fmt.Println("hello0:", string(s))

		// s, _ = json.Marshal(*rsp)
		// fmt.Println("hello1:", string(s))
	}
	return err
}

func (u *UserProvider) Service() string {
	return "com.ikurento.user.UserProvider"
}

func (u *UserProvider) Version() string {
	return ""
}
