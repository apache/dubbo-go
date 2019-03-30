package main

import (
	"fmt"
	"github.com/dubbogo/hessian2"
	"strconv"
	"time"
)

import (
	"github.com/AlexStocks/goext/time"
)

type JsonRPCUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int64  `json:"age"`
	Time int64  `json:"time"`
	Sex  string `json:"sex"`
}

func (u JsonRPCUser) String() string {
	return fmt.Sprintf(
		"User{ID:%s, Name:%s, Age:%d, Time:%s, Sex:%s}",
		u.ID, u.Name, u.Age, gxtime.YMDPrint(int(u.Time), 0), u.Sex,
	)
}

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

type Response struct {
	Status int
	Err    string
	Data   int
}

func (r Response) String() string {
	return fmt.Sprintf(
		"Response{Status:%d, Err:%s, Data:%d}",
		r.Status, r.Err, r.Data,
	)
}

func (Response) JavaClassName() string {
	return "com.ikurento.user.Response"
}
