package main

import (
	// "encoding/json"
	"context"
	"fmt"
	"strconv"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
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

type (
	DubboUser struct {
		// !!! Cannot define lowercase names of variable
		Id   string
		Name string
		Age  int32
		Time time.Time
		Sex  Gender // 注意此处，java enum Object <--> go string
	}

	UserProvider struct {
		user map[string]DubboUser
	}
)

func (u DubboUser) String() string {
	return fmt.Sprintf(
		"User{Id:%s, Name:%s, Age:%d, Time:%s, Sex:%s}",
		u.Id, u.Name, u.Age, u.Time, u.Sex,
	)
}

func (DubboUser) JavaClassName() string {
	return "com.ikurento.user.User"
}

var (
	DefaultUser = DubboUser{
		Id: "0", Name: "Alex Stocks", Age: 31,
		Sex: Gender(MAN),
	}

	userMap = UserProvider{user: make(map[string]DubboUser)}
)

func init() {
	//DefaultUser.Sex = DefaultUser.sex.String()
	userMap.user["A000"] = DefaultUser
	userMap.user["A001"] = DubboUser{Id: "001", Name: "ZhangSheng", Age: 18, Sex: Gender(MAN)}
	userMap.user["A002"] = DubboUser{Id: "002", Name: "Lily", Age: 20, Sex: Gender(WOMAN)}
	userMap.user["A003"] = DubboUser{Id: "113", Name: "Moorse", Age: 30, Sex: Gender(WOMAN)}
	for k, v := range userMap.user {
		userMap.user[k] = v
	}
}

func (u *UserProvider) getUser(userId string) (*DubboUser, error) {
	if user, ok := userMap.user[userId]; ok {
		return &user, nil
	}

	return nil, fmt.Errorf("invalid user id:%s", userId)
}

/*
	!!! req must be []interface{}
*/
func (u *UserProvider) GetUser(ctx context.Context, req []interface{}, rsp *DubboUser) error {
	var (
		err  error
		user *DubboUser
	)

	gxlog.CInfo("req:%#v", req)
	user, err = u.getUser(req[0].(string))
	if err == nil {
		*rsp = *user
		gxlog.CInfo("rsp:%#v", rsp)
	}
	return err
}

func (u *UserProvider) Service() string {
	return "com.ikurento.user.UserProvider"
}

func (u *UserProvider) Version() string {
	return ""
}
