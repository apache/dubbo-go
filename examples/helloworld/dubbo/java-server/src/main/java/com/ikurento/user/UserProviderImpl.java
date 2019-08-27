package com.ikurento.user;

// ref: https://github.com/JoeCao/dubbo_jsonrpc_example/tree/master/dubbo_server/src/main/java/com/ofpay/demo/api

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class UserProviderImpl implements UserProvider {
    private static final Logger LOG = LoggerFactory.getLogger("UserLogger"); //Output to user-server.log

    public User GetUser(String userId) {
        return new User(userId, "zhangsan", 18);
    }

}
