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

import java.io.*;

public final class Response<T> implements Serializable {
    private static final long serialVersionUID = 3727205004706510648L;
    public static final Integer OK = 200;
    public static final Integer ERR = 500;
    private Integer Status;
    private String Err;
    private T Data;

    public Response() {
    }

    public static <T> Response<T> ok() {
        Response r = new Response();
        r.Status = OK;
        return r;
    }

    public static <T> Response<T> ok(Object Data) {
        Response r = new Response();
        r.Status = OK;
        r.Data = Data;
        return r;
    }

    public static <T> Response<T> notOk(String Err) {
        Response r = new Response();
        r.Status = ERR;
        r.Err = Err;
        return r;
    }

    public static <T> Response<T> notOk(Integer Status, String Err) {
        Response r = new Response();
        r.Status = Status;
        r.Err = Err;
        return r;
    }

//    public Boolean isSuccess() {
//        return Objects.equals(this.Status, OK);
//    }

    public Integer getStatus() {
        return this.Status;
    }

    public void setStatus(Integer Status) {
        this.Status = Status;
    }

    public String getErr() {
        return this.Err;
    }

    public void setErr(String Err) {
        this.Err = Err;
    }

    public T getData() {
        return this.Data;
    }

    public void setData(T Data) {
        this.Status = OK;
        this.Data = Data;
    }

    public String toString() {
        return "Response{Status=" + this.Status + ", Err='" + this.Err + '\'' + ", Data=" + this.Data + '}';
    }
}