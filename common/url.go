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

package common

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
)

/////////////////////////////////
// dubbo role type
/////////////////////////////////

const (
	CONSUMER = iota
	CONFIGURATOR
	ROUTER
	PROVIDER
)

var (
	DubboNodes = [...]string{"consumers", "configurators", "routers", "providers"}
	DubboRole  = [...]string{"consumer", "", "", "provider"}
)

type RoleType int

func (t RoleType) String() string {
	return DubboNodes[t]
}

func (t RoleType) Role() string {
	return DubboRole[t]
}

type baseUrl struct {
	Protocol string
	Location string // ip+port
	Ip       string
	Port     string
	//url.Values is not safe map, add to avoid concurrent map read and map write error
	paramsLock   sync.RWMutex
	Params       url.Values
	PrimitiveURL string
	ctx          context.Context
}

type URL struct {
	baseUrl
	Path     string // like  /com.ikurento.dubbo.UserProvider3
	Username string
	Password string
	Methods  []string
	//special for registry
	SubURL *URL
}

type option func(*URL)

func WithUsername(username string) option {
	return func(url *URL) {
		url.Username = username
	}
}

func WithPassword(pwd string) option {
	return func(url *URL) {
		url.Password = pwd
	}
}

func WithMethods(methods []string) option {
	return func(url *URL) {
		url.Methods = methods
	}
}

func WithParams(params url.Values) option {
	return func(url *URL) {
		url.Params = params
	}
}
func WithParamsValue(key, val string) option {
	return func(url *URL) {
		url.Params.Set(key, val)
	}
}
func WithProtocol(proto string) option {
	return func(url *URL) {
		url.Protocol = proto
	}
}
func WithIp(ip string) option {
	return func(url *URL) {
		url.Ip = ip
	}
}

func WithPort(port string) option {
	return func(url *URL) {
		url.Port = port
	}
}

func WithPath(path string) option {
	return func(url *URL) {
		url.Path = "/" + strings.TrimPrefix(path, "/")
	}
}

func WithLocation(location string) option {
	return func(url *URL) {
		url.Location = location
	}
}
func NewURLWithOptions(opts ...option) *URL {
	url := &URL{}
	for _, opt := range opts {
		opt(url)
	}
	url.Location = url.Ip + ":" + url.Port
	return url
}

func NewURL(ctx context.Context, urlString string, opts ...option) (URL, error) {

	var (
		err          error
		rawUrlString string
		serviceUrl   *url.URL
		s            = URL{baseUrl: baseUrl{ctx: ctx}}
	)

	// new a null instance
	if urlString == "" {
		return s, nil
	}

	rawUrlString, err = url.QueryUnescape(urlString)
	if err != nil {
		return s, perrors.Errorf("url.QueryUnescape(%s),  error{%v}", urlString, err)
	}

	//rawUrlString = "//" + rawUrlString
	if strings.Index(rawUrlString, "//") < 0 {
		t := URL{baseUrl: baseUrl{ctx: ctx}}
		for _, opt := range opts {
			opt(&t)
		}
		rawUrlString = t.Protocol + "://" + rawUrlString
	}
	serviceUrl, err = url.Parse(rawUrlString)
	if err != nil {
		return s, perrors.Errorf("url.Parse(url string{%s}),  error{%v}", rawUrlString, err)
	}

	s.Params, err = url.ParseQuery(serviceUrl.RawQuery)
	if err != nil {
		return s, perrors.Errorf("url.ParseQuery(raw url string{%s}),  error{%v}", serviceUrl.RawQuery, err)
	}

	s.PrimitiveURL = urlString
	s.Protocol = serviceUrl.Scheme
	s.Username = serviceUrl.User.Username()
	s.Password, _ = serviceUrl.User.Password()
	s.Location = serviceUrl.Host
	s.Path = serviceUrl.Path
	if strings.Contains(s.Location, ":") {
		s.Ip, s.Port, err = net.SplitHostPort(s.Location)
		if err != nil {
			return s, perrors.Errorf("net.SplitHostPort(Url.Host{%s}), error{%v}", s.Location, err)
		}
	}
	for _, opt := range opts {
		opt(&s)
	}
	return s, nil
}

func (c URL) URLEqual(url URL) bool {
	c.Ip = ""
	c.Port = ""
	url.Ip = ""
	url.Port = ""
	cGroup := c.GetParam(constant.GROUP_KEY, "")
	urlGroup := url.GetParam(constant.GROUP_KEY, "")
	cKey := c.Key()
	urlKey := url.Key()

	if cGroup == constant.ANY_VALUE {
		cKey = strings.Replace(cKey, "group=*", "group="+urlGroup, 1)
	} else if urlGroup == constant.ANY_VALUE {
		urlKey = strings.Replace(urlKey, "group=*", "group="+cGroup, 1)
	}
	if cKey != urlKey {
		return false
	}
	return true
}

func (c URL) String() string {
	buildString := fmt.Sprintf(
		"%s://%s:%s@%s:%s%s?",
		c.Protocol, c.Username, c.Password, c.Ip, c.Port, c.Path)
	buildString += c.Params.Encode()
	return buildString
}

func (c URL) Key() string {
	buildString := fmt.Sprintf(
		"%s://%s:%s@%s:%s/?interface=%s&group=%s&version=%s",
		c.Protocol, c.Username, c.Password, c.Ip, c.Port, c.Service(), c.GetParam(constant.GROUP_KEY, ""), c.GetParam(constant.VERSION_KEY, ""))
	return buildString
	//return c.ServiceKey()
}

func (c URL) ServiceKey() string {
	intf := c.GetParam(constant.INTERFACE_KEY, strings.TrimPrefix(c.Path, "/"))
	if intf == "" {
		return ""
	}
	buf := &bytes.Buffer{}
	group := c.GetParam(constant.GROUP_KEY, "")
	if group != "" {
		buf.WriteString(group)
		buf.WriteString("/")
	}

	buf.WriteString(intf)

	version := c.GetParam(constant.VERSION_KEY, "")
	if version != "" && version != "0.0.0" {
		buf.WriteString(":")
		buf.WriteString(version)
	}

	return buf.String()
}

func (c URL) Context() context.Context {
	return c.ctx
}

func (c URL) Service() string {
	service := c.GetParam(constant.INTERFACE_KEY, strings.TrimPrefix(c.Path, "/"))
	if service != "" {
		return service
	} else if c.SubURL != nil {
		service = c.GetParam(constant.INTERFACE_KEY, strings.TrimPrefix(c.Path, "/"))
		if service != "" { //if url.path is "" then return suburl's path, special for registry Url
			return service
		}
	}
	return ""
}

func (c *URL) AddParam(key string, value string) {
	c.paramsLock.Lock()
	c.Params.Add(key, value)
	c.paramsLock.Unlock()
}

func (c URL) GetParam(s string, d string) string {
	var r string
	c.paramsLock.RLock()
	if r = c.Params.Get(s); len(r) == 0 {
		r = d
	}
	c.paramsLock.RUnlock()
	return r
}
func (c URL) GetParamAndDecoded(key string) (string, error) {
	ruleDec, err := base64.URLEncoding.DecodeString(c.GetParam(key, ""))
	value := string(ruleDec)
	return value, err
}

func (c URL) GetRawParam(key string) string {
	switch key {
	case "protocol":
		return c.Protocol
	case "username":
		return c.Username
	case "host":
		return strings.Split(c.Location, ":")[0]
	case "password":
		return c.Password
	case "port":
		return c.Port
	case "path":
		return c.Path
	default:
		return c.Params.Get(key)
	}
}

// GetParamBool
func (c URL) GetParamBool(s string, d bool) bool {

	var r bool
	var err error
	if r, err = strconv.ParseBool(c.Params.Get(s)); err != nil {
		return d
	}
	return r
}

func (c URL) GetParamInt(s string, d int64) int64 {
	var r int
	var err error
	if r, err = strconv.Atoi(c.Params.Get(s)); r == 0 || err != nil {
		return d
	}
	return int64(r)
}

func (c URL) GetMethodParamInt(method string, key string, d int64) int64 {
	var r int
	var err error
	if r, err = strconv.Atoi(c.Params.Get("methods." + method + "." + key)); r == 0 || err != nil {
		return d
	}
	return int64(r)
}

func (c URL) GetMethodParamInt64(method string, key string, d int64) int64 {
	r := c.GetMethodParamInt(method, key, math.MinInt64)
	if r == math.MinInt64 {
		return c.GetParamInt(key, d)
	}

	return r
}

func (c URL) GetMethodParam(method string, key string, d string) string {
	var r string
	if r = c.Params.Get("methods." + method + "." + key); r == "" {
		r = d
	}
	return r
}

// ToMap transfer URL to Map
func (c URL) ToMap() map[string]string {

	paramsMap := make(map[string]string)

	for k, v := range c.Params {
		paramsMap[k] = v[0]
	}
	if c.Protocol != "" {
		paramsMap["protocol"] = c.Protocol
	}
	if c.Username != "" {
		paramsMap["username"] = c.Username
	}
	if c.Password != "" {
		paramsMap["password"] = c.Password
	}
	if c.Location != "" {
		paramsMap["host"] = strings.Split(c.Location, ":")[0]
		var port string
		if strings.Contains(c.Location, ":") {
			port = strings.Split(c.Location, ":")[1]
		} else {
			port = "0"
		}
		paramsMap["port"] = port
	}
	if c.Protocol != "" {
		paramsMap["protocol"] = c.Protocol
	}
	if c.Path != "" {
		paramsMap["path"] = c.Path
	}
	if len(paramsMap) == 0 {
		return nil
	}
	return paramsMap
}

// configuration  > reference config >service config
//  in this function we should merge the reference local url config into the service url from registry.
//TODO configuration merge, in the future , the configuration center's config should merge too.
func MergeUrl(serviceUrl URL, referenceUrl *URL) URL {
	mergedUrl := serviceUrl
	var methodConfigMergeFcn = []func(method string){}
	//iterator the referenceUrl if serviceUrl not have the key ,merge in

	for k, v := range referenceUrl.Params {
		if _, ok := mergedUrl.Params[k]; !ok {
			mergedUrl.Params.Set(k, v[0])
		}
	}
	//loadBalance strategy config
	if v := referenceUrl.Params.Get(constant.LOADBALANCE_KEY); v != "" {
		mergedUrl.Params.Set(constant.LOADBALANCE_KEY, v)
	}
	methodConfigMergeFcn = append(methodConfigMergeFcn, func(method string) {
		if v := referenceUrl.Params.Get(method + "." + constant.LOADBALANCE_KEY); v != "" {
			mergedUrl.Params.Set(method+"."+constant.LOADBALANCE_KEY, v)
		}
	})

	//cluster strategy config
	if v := referenceUrl.Params.Get(constant.CLUSTER_KEY); v != "" {
		mergedUrl.Params.Set(constant.CLUSTER_KEY, v)
	}
	methodConfigMergeFcn = append(methodConfigMergeFcn, func(method string) {
		if v := referenceUrl.Params.Get(method + "." + constant.CLUSTER_KEY); v != "" {
			mergedUrl.Params.Set(method+"."+constant.CLUSTER_KEY, v)
		}
	})

	//remote timestamp
	if v := serviceUrl.Params.Get(constant.TIMESTAMP_KEY); v != "" {
		mergedUrl.Params.Set(constant.REMOTE_TIMESTAMP_KEY, v)
		mergedUrl.Params.Set(constant.TIMESTAMP_KEY, referenceUrl.Params.Get(constant.TIMESTAMP_KEY))
	}

	//finally execute methodConfigMergeFcn
	for _, method := range referenceUrl.Methods {
		for _, fcn := range methodConfigMergeFcn {
			fcn("methods." + method)
		}
	}

	return mergedUrl
}
