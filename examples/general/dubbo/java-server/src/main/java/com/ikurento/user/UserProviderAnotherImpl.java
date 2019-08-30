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

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

public class UserProviderAnotherImpl implements UserProvider {
    // private static final Logger logger = LoggerFactory.getLogger(getClass()); // Only output to dubbo's log(logs/server.log)
    private static final Logger logger = LoggerFactory.getLogger("userLogger"); // Output to user-server.log

    private Map<String, User> userMap = new HashMap<String, User>();

    public UserProviderAnotherImpl() {
        // userMap.put("001", new User("001", "other-zhangsan", 18, new Date(1998-1900, 1, 2, 3, 4, 5), Gender.MAN));
        userMap.put("001", new User("001", "other-zhangsan", 18, new Date(0x12345678), Gender.MAN));
        userMap.put("002", new User("002", "other-lisi", 20, new Date(1996-1900, 1, 2, 3, 4, 5), Gender.MAN));
        userMap.put("003", new User("003", "other-lily", 23, new Date(1993-1900, 1, 2, 3, 4, 5), Gender.WOMAN));
        userMap.put("004", new User("004", "other-lisa", 32, new Date(1985-1900, 1, 2, 3, 4, 5), Gender.WOMAN));
    }

    public boolean isLimit(Gender gender, String name) {
        logger.info(String.format("input gender=%sand name=%s", gender, name));
        return Gender.MAN == gender;
    }

    public User GetUser(String userId) {
        logger.info("input userId = " + userId);
        return new User(userId, "Joe", 48);
    }

    public User GetUser0(String userId, String name) {
                return new User(userId, name, 48);
    }
    public void GetUser3() {
    }
    public User GetErr(String userId) throws Exception {
        throw new Exception("exception");
    }
    public List<User> GetUsers(ArrayList<String> userIdList) {
        Iterator it = userIdList.iterator();
        List<User> userList = new ArrayList<User>();
        logger.warn("@userIdList size:" + userIdList.size());

        while(it.hasNext()) {
            String id = (String)(it.next());
            logger.info("GetUsers(@uid:" + id + ")");
            if (userMap.containsKey(id)) {
                userList.add(userMap.get(id));
                logger.info("id:" + id + ", user:" + userMap.get(id));
            }
        }

        return userList;
    }

    public Map<String, User> GetUserMap(List<String> userIdList) {
        Iterator it = userIdList.iterator();
        Map<String, User> map = new HashMap<String, User>();
        logger.warn("@userIdList size:" + userIdList.size());

        while(it.hasNext()) {
            String id = (String)(it.next());
            logger.info("GetUsers(@uid:" + id + ")");
            if (userMap.containsKey(id)) {
                map.put(id, userMap.get(id));
                logger.info("id:" + id + ", user:" + userMap.get(id));
            }
        }

        return map;
    }

    public List<User> GetUsers(List<String> userIdList) {
        Iterator it = userIdList.iterator();
        List<User> userList = new ArrayList<User>();
        logger.warn("@userIdList size:" + userIdList.size());

        while(it.hasNext()) {
            String id = (String)(it.next());
            logger.info("GetUsers(@uid:" + id + ")");
            if (userMap.containsKey(id)) {
                userList.add(userMap.get(id));
                logger.info("id:" + id + ", user:" + userMap.get(id));
            }
        }

        return userList;
    }

    // @Override
    public User getUser(int userCode) {
        logger.info("input userCode = " + userCode);
        return new User(String.valueOf(userCode), "userCode get", 48);
    }

    public User queryUser(User user) {
        logger.info("input user = " + user);
        return new User(user.getId(), "get:" + user.getName(), user.getAge() + 18);
    }

    public Map<String, User> queryAll() {
        logger.info("input");
        Map<String, User> map = new HashMap<String, User>();
        map.put("001", new User("001", "Joe", 18));
        map.put("002", new User("002", "Wen", 20));

        return map;
    }

    public int Calc(int a,int b) {
        return a + b + 100;
    }

    public Response<Integer> Sum(int a,int b) {
        return Response.ok(a+b);
    }
}
