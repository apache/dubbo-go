// *****************************************************
// DESC    : dubbo consumer
// AUTHOR  : writtey by 包增辉(https://github.com/baozh)
// VERSION : 1.0
// LICENCE : Apache License 2.0
// EMAIL   : alexstocks@foxmail.com
// MOD     : 2016-10-19 17:03
// FILE    : Consumer.java
// ******************************************************

package com.ikurento.user;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;

public class Consumer {
    //定义一个私有变量 （Spring中要求）
    private UserProvider userProvider;

    //Spring注入（Spring中要求）
    public void setUserProvider(UserProvider u) {
        this.userProvider = u;
    }

    private void benchmarkSayHello() {
        for (int i = 0; i < Integer.MAX_VALUE; i ++) {
            try {
                // String hello = demoService.sayHello("world" + i);
                // System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " + hello);
            } catch (Exception e) {
                e.printStackTrace();
            }

            // Thread.sleep(2000);
        }
    }

    private void testGetUser() throws Exception {
        try {
            User user1 = userProvider.GetUser("003");
            System.out.println("[" + new SimpleDateFormat("HH:mm:ss").format(new Date()) + "] " +
                    " UserInfo, Id:" + user1.getId() + ", name:" + user1.getName() + ", sex:" + user1.getSex().toString()
                    + ", age:" + user1.getAge() + ", time:" + user1.getTime().toString());
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    private void testGetUsers() throws Exception {
        try {
            List<String> userIDList = new ArrayList<String>();
            userIDList.add("001");
            userIDList.add("002");
            userIDList.add("003");

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

    //启动consumer的入口函数(在配置文件中指定)
    public void start() throws Exception {
        testGetUser();
        // testGetUsers();
        Thread.sleep(120000);
    }
}
