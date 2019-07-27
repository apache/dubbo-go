package com.ikurento.user;

// ref: https://github.com/JoeCao/dubbo_jsonrpc_example/tree/master/dubbo_server/src/main/java/com/ofpay/demo/api

import java.util.HashMap;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class UserProviderImpl implements UserProvider {
    private static final Logger LOG = LoggerFactory.getLogger("UserLogger"); // 输出到user-server.log
    Map<String, User> userMap = new HashMap<String, User>();

    public UserProviderImpl() {
        userMap.put("A001", new User("A001", "demo-zhangsan", 18));
        userMap.put("A002", new User("A002", "demo-lisi", 20));
        userMap.put("A003", new User("A003", "demo-lily", 23));
        userMap.put("A004", new User("A004", "demo-lisa", 32));
    }


    public User GetUser(String userId) {
        return new User(userId, "zhangsan", 18);
    }

}
