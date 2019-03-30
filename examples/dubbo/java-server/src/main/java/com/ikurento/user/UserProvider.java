package com.ikurento.user;
// https://github.com/JoeCao/dubbo_jsonrpc_example/tree/master/dubbo_server/src/main/java/com/ofpay/demo/api

import java.util.List;
import java.util.Map;

public interface UserProvider {

    boolean isLimit(Gender gender, String name);

    User GetUser(String userId); // the first alpha is Upper case to compatible with golang.

    List<User> GetUsers(List<String> userIdList);

    Map<String, User> GetUserMap(List<String> userIdList);

    User getUser(int usercode);

    User queryUser(User user);

    Map<String, User> queryAll();

    int Calc(int a,int b);

    Response<Integer> Sum(int a, int b);
}
