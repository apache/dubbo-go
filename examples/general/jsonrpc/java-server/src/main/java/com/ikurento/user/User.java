package com.ikurento.user;
// ref: https://github.com/JoeCao/dubbo_jsonrpc_example/tree/master/dubbo_server/src/main/java/com/ofpay/demo/api

import java.util.Date;
import java.io.Serializable;

public class User implements Serializable  {

    private String id;

    private String name;

    private int age;

    private Date time = new Date();

    private Gender sex = Gender.MAN;

    public User() {
    }

    public User(String id, String name, int age) {
        this.id = id;
        this.name = name;
        this.age = age;
    }

    public User(String id, String name, int age, Date time, Gender sex) {
        this.id = id;
        this.name = name;
        this.age = age;
        this.time = time;
        this.sex = sex;
    }

    public String getId() {
        return id;
    }

    public void setId(String id) {
        this.id = id;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public int getAge() {
        return age;
    }

    public void setAge(int age) {
        this.age = age;
    }

    public Date getTime() {
        return time;
    }

    public void setTime(Date time) {
        this.time = time;
    }

    public Gender getSex() {
        return sex;
    }

    public void setSex(Gender sex) {
        this.sex = sex;
    }

    public String toString() {
        return "User{id:" + id + ", name:" + name + ", age:" + age + ", time:" + time + ", gender:" + sex + "}";
    }
}
