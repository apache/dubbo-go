package com.ikurento.user;

// ref: https://github.com/JoeCao/dubbo_jsonrpc_example/tree/master/dubbo_server/src/main/java/com/ofpay/demo/api

import java.util.HashMap;
import java.util.List;
import java.util.ArrayList;
import java.util.Map;
import java.util.Iterator;

// import org.apache.log4j.Logger;
// import org.apache.log4j.LoggerFactory;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class UserProviderImpl implements UserProvider {
    // private static final Logger logger = LoggerFactory.getLogger(getClass()); // 只输出到dubbo的log(logs/server.log)
    private static final Logger LOG = LoggerFactory.getLogger("UserLogger"); // 输出到user-server.log
    Map<String, User> userMap = new HashMap<String, User>();

    public UserProviderImpl() {
        userMap.put("001", new User("001", "demo-zhangsan", 18));
        userMap.put("002", new User("002", "demo-lisi", 20));
        userMap.put("003", new User("003", "demo-lily", 23));
        userMap.put("004", new User("004", "demo-lisa", 32));
    }

    public boolean isLimit(Gender gender, String name) {
        return Gender.WOMAN == gender;
    }

    public User GetUser(String userId) {
        return new User(userId, "zhangsan", 18);
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

    @Override
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
