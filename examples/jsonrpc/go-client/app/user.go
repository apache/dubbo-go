package main

import (
	"context"
	"fmt"
)

import (
	"github.com/AlexStocks/goext/time"
	"github.com/dubbo/go-for-apache-dubbo/config"
)

func init() {
	config.SetConService(new(UserProvider))
}

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

type UserProvider struct {
	GetUser  func(ctx context.Context, req []interface{}, rsp *JsonRPCUser) error
	GetUser0 func(ctx context.Context, req interface{}) (*JsonRPCUser, error) `dubbo:"GetUser"`
	GetUser1 func(ctx context.Context, req []interface{}, rsp *JsonRPCUser) error
	GetUser2 func(ctx context.Context, req []interface{}, rsp *JsonRPCUser) error `dubbo:"getUser"`
	Echo     func(ctx context.Context, req interface{}) (interface{}, error)      // Echo represent EchoFilter will be used
}

func (u *UserProvider) Service() string {
	return "com.ikurento.user.UserProvider"
}

func (u *UserProvider) Version() string {
	return ""
}
