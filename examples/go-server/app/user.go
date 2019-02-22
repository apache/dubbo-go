/*****************************************************
# DESC    : UserProvider Service
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-07-21 19:22
# FILE    : user.go
******************************************************/

package main

import (
	// "encoding/json"
	"context"
	"fmt"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
	"github.com/AlexStocks/goext/time"
)

type Gender int

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
func (this *UserId) UnmarshalJSON(value []byte) error {
	this.Id = string(value)
	this.Id = strings.TrimPrefix(this.Id, "\"")
	this.Id = strings.TrimSuffix(this.Id, `"`)

	return nil
}
*/

func (this *UserProvider) getUser(userId string) (*User, error) {
	if user, ok := userMap.user[userId]; ok {
		return &user, nil
	}

	return nil, fmt.Errorf("invalid user id:%s", userId)
}

/*
// can not work
func (this *UserProvider) GetUser(ctx context.Context, req *UserId, rsp *User) error {
	var (
		err  error
		user *User
	)
	user, err = this.getUser(req.Id)
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
func (this *UserProvider) GetUser(ctx context.Context, req *string, rsp *User) error {
	var (
		err  error
		user *User
	)

	gxlog.CInfo("req:%#v", *req)
	user, err = this.getUser(*req)
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

func (this *UserProvider) GetUser(ctx context.Context, req []string, rsp *User) error {
	var (
		err  error
		user *User
	)

	gxlog.CInfo("req:%#v", req)
	user, err = this.getUser(req[0])
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

func (this *UserProvider) Service() string {
	return "com.ikurento.user.UserProvider"
}

func (this *UserProvider) Version() string {
	return ""
}
