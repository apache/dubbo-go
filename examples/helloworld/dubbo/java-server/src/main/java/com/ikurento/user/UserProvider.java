package com.ikurento.user;
// https://github.com/JoeCao/dubbo_jsonrpc_example/tree/master/dubbo_server/src/main/java/com/ofpay/demo/api


public interface UserProvider {

    User GetUser(String userId); // the first alpha is Upper case to compatible with golang.

}
