package main

import (
	"fmt"
	"strconv"
	"time"
)

import (
	"github.com/dubbogo/hessian2"
)

type Gender hessian.JavaEnum

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

type DubboUser struct {
	Id   string
	Name string
	Age  int32
	Time time.Time
	Sex  Gender // 注意此处，java enum Object <--> go string
}

func (u DubboUser) String() string {
	return fmt.Sprintf(
		"User{Id:%s, Name:%s, Age:%d, Time:%s, Sex:%s}",
		u.Id, u.Name, u.Age, u.Time, u.Sex,
	)
}

func (DubboUser) JavaClassName() string {
	return "com.ikurento.user.User"
}

type UserProvider struct {
	GetUser  func(userID string) (user DubboUser, err error)
	GetUsers func(userIDs []string) (user []DubboUser, err error)
}

type UserProviderNoErr struct {
	GetUser  func(userID string) (user DubboUser)
	GetUsers func(userIDs []string) (user []DubboUser)
}

type UserProviderRetPtr struct {
	GetUser  func(userID string) (user *DubboUser, err error)
	GetUsers func(userIDs []string) (user []DubboUser, err error)
}
