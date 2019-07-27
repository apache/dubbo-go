package main

import (
	"context"
	"fmt"
)

import (
	"github.com/apache/dubbo-go/config"
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

func (u *UserProvider) Reference() string {
	return "UserProvider"
}
