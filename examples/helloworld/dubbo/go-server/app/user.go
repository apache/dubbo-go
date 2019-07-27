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
	"fmt"
	"time"
)

type (
	User struct {
		// !!! Cannot define lowercase names of variable
		Id   string
		Name string
		Age  int32
		Time time.Time
	}
)

var (
	DefaultUser = User{
		Id: "0", Name: "Alex Stocks", Age: 31,
	}

	userMap = make(map[string]User)
)

func init() {
	userMap["A000"] = DefaultUser
	userMap["A001"] = User{Id: "001", Name: "ZhangSheng", Age: 18}
	userMap["A002"] = User{Id: "002", Name: "Lily", Age: 20}
	userMap["A003"] = User{Id: "113", Name: "Moorse", Age: 30}
	for k, v := range userMap {
		v.Time = time.Now()
		userMap[k] = v
	}
}

func (u User) String() string {
	return fmt.Sprintf(
		"User{Id:%s, Name:%s, Age:%d, Time:%s}",
		u.Id, u.Name, u.Age, u.Time,
	)
}

func (u User) JavaClassName() string {
	return "com.ikurento.user.User"
}

func println(format string, args ...interface{}) {
	fmt.Printf("\033[32;40m"+format+"\033[0m\n", args...)
}
