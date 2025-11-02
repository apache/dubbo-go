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
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// User 用户信息结构体
type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// UserService 用户服务接口实现
type UserService struct{}

// GetUser 获取用户信息
func (u *UserService) GetUser(ctx context.Context, userID int64) (*User, error) {
	logger.Infof("GetUser called with userID: %d", userID)

	// 模拟从数据库获取用户信息
	user := &User{
		ID:    userID,
		Name:  fmt.Sprintf("User_%d", userID),
		Email: fmt.Sprintf("user%d@example.com", userID),
		Age:   25 + int(userID%50),
	}

	return user, nil
}

// CreateUser 创建用户
func (u *UserService) CreateUser(ctx context.Context, user *User) (*User, error) {
	logger.Infof("CreateUser called with user: %+v", user)

	// 模拟创建用户，设置ID
	user.ID = time.Now().Unix()

	return user, nil
}

// UpdateUser 更新用户信息
func (u *UserService) UpdateUser(ctx context.Context, userID int64, updates map[string]interface{}) (*User, error) {
	logger.Infof("UpdateUser called with userID: %d, updates: %+v", userID, updates)

	// 模拟更新用户信息
	user := &User{
		ID:    userID,
		Name:  fmt.Sprintf("Updated_%d", userID),
		Email: "updated@example.com",
		Age:   30,
	}

	// 应用更新
	if name, ok := updates["name"].(string); ok {
		user.Name = name
	}
	if email, ok := updates["email"].(string); ok {
		user.Email = email
	}
	if age, ok := updates["age"].(float64); ok {
		user.Age = int(age)
	}

	return user, nil
}

// BatchGetUsers 批量获取用户
func (u *UserService) BatchGetUsers(ctx context.Context, userIDs []int64) ([]*User, error) {
	logger.Infof("BatchGetUsers called with userIDs: %v", userIDs)

	users := make([]*User, len(userIDs))
	for i, id := range userIDs {
		users[i] = &User{
			ID:    id,
			Name:  fmt.Sprintf("BatchUser_%d", id),
			Email: fmt.Sprintf("batch%d@example.com", id),
			Age:   20 + int(id%30),
		}
	}

	return users, nil
}

// Reference 返回服务引用
func (u *UserService) Reference() string {
	return "com.example.UserService"
}

func main() {
	fmt.Println("🚀 启动 Triple 协议 Producer (UserService)")
	fmt.Println("====================================")

	// 创建服务器
	srv, err := server.NewServer(
		server.WithServerProtocol(
			protocol.WithTriple(),
			protocol.WithPort(20001),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("创建服务器失败: %v", err))
	}

	// 注册用户服务
	userService := &UserService{}
	if err := srv.RegisterService(userService,
		server.WithInterface("com.example.UserService"),
		server.WithSerialization("hessian2"),
	); err != nil {
		panic(fmt.Sprintf("注册服务失败: %v", err))
	}

	// 将服务添加到全局配置中，确保$invoke方法能被正确注册
	config.SetProviderService(userService)

	// 创建ServiceConfig并添加到ProviderConfig中
	serviceConfig := config.NewServiceConfigBuilder().
		SetInterface("com.example.UserService").
		SetProtocolIDs("tri").
		SetSerialization("hessian2").
		Build()

	providerConfig := config.GetProviderConfig()
	if providerConfig.Services == nil {
		providerConfig.Services = make(map[string]*config.ServiceConfig)
	}
	providerConfig.Services["com.example.UserService"] = serviceConfig

	fmt.Println("✅ UserService 注册成功")
	fmt.Println("📋 可用方法:")
	fmt.Println("  - GetUser(userID int64) (*User, error)")
	fmt.Println("  - CreateUser(user *User) (*User, error)")
	fmt.Println("  - UpdateUser(userID int64, updates map[string]interface{}) (*User, error)")
	fmt.Println("  - BatchGetUsers(userIDs []int64) ([]*User, error)")
	fmt.Println("")
	fmt.Println("🌐 服务监听地址: localhost:20001")
	fmt.Println("🔧 协议: Triple (非IDL模式)")
	fmt.Println("🎯 支持泛化调用: ✅")
	fmt.Println("")
	fmt.Println("⚡ 服务器启动中...")

	// 启动服务器
	if err := srv.Serve(); err != nil {
		panic(fmt.Sprintf("启动服务器失败: %v", err))
	}
}
