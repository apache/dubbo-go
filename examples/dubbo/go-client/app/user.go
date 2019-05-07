package main

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

import (
	"github.com/dubbogo/hessian2"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/config/support"
)

type Gender hessian.JavaEnum

func init() {
	support.SetConService(new(UserProvider))
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
	GetUser  func(ctx context.Context, req []interface{}, rsp *User) error
	GetUser1 func(ctx context.Context, req []interface{}, rsp *User) error
	Echo     func(ctx context.Context, req []interface{}, rsp *string) error // Echo represent EchoFilter will be used
}

func (u *UserProvider) Service() string {
	return "com.ikurento.user.UserProvider"
}

func (u *UserProvider) Version() string {
	return ""
}
