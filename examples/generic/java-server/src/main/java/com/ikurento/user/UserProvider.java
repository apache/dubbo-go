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

package com.ikurento.user;

import java.util.List;
import java.util.Map;

public interface UserProvider {

    boolean isLimit(Gender gender, String name);

    User GetUser(String userId); // the first alpha is Upper case to compatible with golang.

    List<User> GetUsers(List<String> userIdList);

    void GetUser3();

    User GetUser0(String userId, String name);

	User GetErr(String userId) throws Exception;

    Map<String, User> GetUserMap(List<String> userIdList);

    User getUser(int usercode);

    User queryUser(User user);

    Map<String, User> queryAll();

    int Calc(int a,int b);

    Response<Integer> Sum(int a, int b);
}
