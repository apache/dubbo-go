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

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import com.alibaba.dubbo.rpc.service.EchoService;
import java.util.List;

public class Consumer {
    // Define a private variable (Required in Spring)
    private UserProvider userProvider;
    private UserProvider userProvider1;
    private UserProvider userProvider2;

    // Spring DI (Required in Spring)
    public void setUserProvider(UserProvider u) {
        this.userProvider = u;
    }
    public void setUserProvider1(UserProvider u) {
        this.userProvider1 = u;
    }
    public void setUserProvider2(UserProvider u) {
        this.userProvider2 = u;
    }

    // Start the entry function for consumer (Specified in the configuration file)
    public void start() throws Exception {
        System.out.println("\n\ntest");
        testGetUser();
        testGetUsers();
        System.out.println("\n\ntest1");
        testGetUser1();
        testGetUsers1();
        System.out.println("\n\ntest2");
        testGetUser2();
        testGetUsers2();
        Thread.sleep(2000);
    }

    private void testGetUser() throws Exception {
        try {
            EchoService echoService = (EchoService)userProvider;
            Object status = echoService.$echo("OK");
            System.out.println("echo: "+status);
            User user1 = userProvider.GetUser("A003");
            System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                    " UserInfo, Id:" + user1.getId() + ", name:" + user1.getName() + ", sex:" + user1.getSex().toString()
                    + ", age:" + user1.getAge() + ", time:" + user1.getTime().toString());
            User user2 = userProvider.GetUser0("A003","Moorse");
            System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                    " UserInfo, Id:" + user2.getId() + ", name:" + user2.getName() + ", sex:" + user2.getSex().toString()
                    + ", age:" + user2.getAge() + ", time:" + user2.getTime().toString());
            User user3 = userProvider.getUser(1);
            System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                    " UserInfo, Id:" + user3.getId() + ", name:" + user3.getName() + ", sex:" + user3.getSex().toString()
                    + ", age:" + user3.getAge() + ", time:" + user3.getTime().toString());
            userProvider.GetUser3();
            System.out.println("GetUser3 succ");

            User user9 = userProvider.GetUser1("A003");
        } catch (Exception e) {
            System.out.println("*************exception***********");
            e.printStackTrace();
        }
        try {
            userProvider.GetErr("A003");
        } catch (Throwable t) {
            System.out.println("*************exception***********");
            t.printStackTrace();
        }
    }

    private void testGetUsers() throws Exception {
        try {
            List<String> userIDList = new ArrayList<String>();
            userIDList.add("A001");
            userIDList.add("A002");
            userIDList.add("A003");

            List<User> userList = userProvider.GetUsers(userIDList);

            for (int i = 0; i < userList.size(); i++) {
                User user = userList.get(i);
                System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                        " UserInfo, Id:" + user.getId() + ", name:" + user.getName() + ", sex:" + user.getSex().toString()
                        + ", age:" + user.getAge() + ", time:" + user.getTime().toString());
            }
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    private void testGetUser1() throws Exception {
        try {
            EchoService echoService = (EchoService)userProvider1;
            Object status = echoService.$echo("OK");
            System.out.println("echo: "+status);
            User user1 = userProvider1.GetUser("A003");
            System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                    " UserInfo, Id:" + user1.getId() + ", name:" + user1.getName() + ", sex:" + user1.getSex().toString()
                    + ", age:" + user1.getAge() + ", time:" + user1.getTime().toString());
            User user2 = userProvider1.GetUser0("A003","Moorse");
            System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                    " UserInfo, Id:" + user2.getId() + ", name:" + user2.getName() + ", sex:" + user2.getSex().toString()
                    + ", age:" + user2.getAge() + ", time:" + user2.getTime().toString());
            User user3 = userProvider1.getUser(1);
            System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                    " UserInfo, Id:" + user3.getId() + ", name:" + user3.getName() + ", sex:" + user3.getSex().toString()
                    + ", age:" + user3.getAge() + ", time:" + user3.getTime().toString());
            userProvider1.GetUser3();
            System.out.println("GetUser3 succ");

            User user9 = userProvider1.GetUser1("A003");
        } catch (Exception e) {
            System.out.println("*************exception***********");
            e.printStackTrace();
        }
        try {
            userProvider1.GetErr("A003");
        } catch (Throwable t) {
            System.out.println("*************exception***********");
            t.printStackTrace();
        }
    }

    private void testGetUsers1() throws Exception {
        try {
            List<String> userIDList = new ArrayList<String>();
            userIDList.add("A001");
            userIDList.add("A002");
            userIDList.add("A003");

            List<User> userList = userProvider1.GetUsers(userIDList);

            for (int i = 0; i < userList.size(); i++) {
                User user = userList.get(i);
                System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                        " UserInfo, Id:" + user.getId() + ", name:" + user.getName() + ", sex:" + user.getSex().toString()
                        + ", age:" + user.getAge() + ", time:" + user.getTime().toString());
            }
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    private void testGetUser2() throws Exception {
        try {
            EchoService echoService = (EchoService)userProvider2;
            Object status = echoService.$echo("OK");
            System.out.println("echo: "+status);
            User user1 = userProvider2.GetUser("A003");
            System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                    " UserInfo, Id:" + user1.getId() + ", name:" + user1.getName() + ", sex:" + user1.getSex().toString()
                    + ", age:" + user1.getAge() + ", time:" + user1.getTime().toString());
            User user2 = userProvider2.GetUser0("A003","Moorse");
            System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                    " UserInfo, Id:" + user2.getId() + ", name:" + user2.getName() + ", sex:" + user2.getSex().toString()
                    + ", age:" + user2.getAge() + ", time:" + user2.getTime().toString());
            User user3 = userProvider2.getUser(1);
            System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                    " UserInfo, Id:" + user3.getId() + ", name:" + user3.getName() + ", sex:" + user3.getSex().toString()
                    + ", age:" + user3.getAge() + ", time:" + user3.getTime().toString());
            userProvider2.GetUser3();
            System.out.println("GetUser3 succ");

            User user9 = userProvider2.GetUser1("A003");
        } catch (Exception e) {
            System.out.println("*************exception***********");
            e.printStackTrace();
        }
        try {
            userProvider2.GetErr("A003");
        } catch (Throwable t) {
            System.out.println("*************exception***********");
            t.printStackTrace();
        }
    }

    private void testGetUsers2() throws Exception {
        try {
            List<String> userIDList = new ArrayList<String>();
            userIDList.add("A001");
            userIDList.add("A002");
            userIDList.add("A003");

            List<User> userList = userProvider2.GetUsers(userIDList);

            for (int i = 0; i < userList.size(); i++) {
                User user = userList.get(i);
                System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                        " UserInfo, Id:" + user.getId() + ", name:" + user.getName() + ", sex:" + user.getSex().toString()
                        + ", age:" + user.getAge() + ", time:" + user.getTime().toString());
            }
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}
