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


import java.util.HashMap;
import java.util.List;
import java.util.ArrayList;
import java.util.Map;
import java.util.Iterator;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class UserProviderImpl implements UserProvider {
    // private static final Logger logger = LoggerFactory.getLogger(getClass()); // Only output to dubbo's log(logs/server.log)
    private static final Logger LOG = LoggerFactory.getLogger("UserLogger"); // Output to user-server.log
    Map<String, User> userMap = new HashMap<String, User>();

    public UserProviderImpl() {
        userMap.put("A001", new User("A001", "demo-zhangsan", 18));
        userMap.put("A002", new User("A002", "demo-lisi", 20));
        userMap.put("A003", new User("A003", "demo-lily", 23));
        userMap.put("A004", new User("A004", "demo-lisa", 32));
    }

    public boolean isLimit(Gender gender, String name) {
        return Gender.WOMAN == gender;
    }

    public User GetUser(String userId) {
        return new User(userId, "zhangsan", 18);
    }
    public User GetErr(String userId) throws Exception {
        throw new Exception("exception");
    }
    public User GetUser0(String userId, String name) {
            return new User(userId, name, 18);
    }

    public List<User> GetUsers(List<String> userIdList) {
        Iterator it = userIdList.iterator();
        List<User> userList = new ArrayList<User>();
        LOG.warn("@userIdList size:" + userIdList.size());

        while(it.hasNext()) {
            String id = (String)(it.next());
            LOG.info("GetUsers(@uid:" + id + ")");
            if (userMap.containsKey(id)) {
                userList.add(userMap.get(id));
                LOG.info("id:" + id + ", user:" + userMap.get(id));
            }
        }

        return userList;
    }

    public void GetUser3() {
    }

    public Map<String, User> GetUserMap(List<String> userIdList) {
        Iterator it = userIdList.iterator();
        Map<String, User> map = new HashMap<String, User>();
        LOG.warn("@userIdList size:" + userIdList.size());

        while(it.hasNext()) {
            String id = (String)(it.next());
            LOG.info("GetUsers(@uid:" + id + ")");
            if (userMap.containsKey(id)) {
                map.put(id, userMap.get(id));
                LOG.info("id:" + id + ", user:" + userMap.get(id));
            }
        }

        return map;
    }

    public User queryUser(User user) {
        return new User(user.getId(), "hello:" +user.getName(), user.getAge() + 18);
    }

    public Map<String, User> queryAll() {
        return userMap;
    }


    public User getUser(int userCode) {
        return new User(String.valueOf(userCode), "userCode get", 48);
    }


    public int Calc(int a,int b) {
        return a + b;
    }

     public Response<Integer> Sum(int a,int b) {
        return Response.ok(a+b);
    }
}
