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
	"encoding/base64"
	"fmt"
	cm "github.com/Workiva/go-datastructures/common"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/jinzhu/copier"
	perrors "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

// ///////////////////////////////
// dubbo role type
// ///////////////////////////////

// role constant
const (
	// CONSUMER is consumer role
	CONSUMER = iota
	// CONFIGURATOR is configurator role
	CONFIGURATOR
	// ROUTER is router role
	ROUTER
	// PROVIDER is provider role
	PROVIDER
	PROTOCOL = "protocol"
)

var (
	// DubboNodes Dubbo service node
	DubboNodes = [...]string{"consumers", "configurators", "routers", "providers"}
	// DubboRole Dubbo service role
	DubboRole = [...]string{"consumer", "", "routers", "provider"}
)

// nolint
type RoleType int

func (t RoleType) String() string {
	return DubboNodes[t]
}

// Role returns role by @RoleType
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
	params       url.Values
	PrimitiveURL string
}

// noCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// URL thread-safe. but this url should not be copied.
// we fail to define this struct to be immutable object.
// but, those method which will update the URL, including SetParam, SetParams
// are only allowed to be invoked in creating URL instance
// Please keep in mind that this struct is immutable after it has been created and initialized.
type URL struct {
	noCopy noCopy

	baseUrl
	Path     string // like  /com.ikurento.dubbo.UserProvider
	Username string
	Password string
	Methods  []string
	// special for registry
	SubURL *URL
}

// Option accepts url
// Option will define a function of handling URL
type Option func(*URL)

// WithUsername sets username for url
func WithUsername(username string) Option {
	return func(url *URL) {
		url.Username = username
	}
}

// WithPassword sets password for url
func WithPassword(pwd string) Option {
	return func(url *URL) {
		url.Password = pwd
	}
}

// WithMethods sets methods for url
func WithMethods(methods []string) Option {
	return func(url *URL) {
		url.Methods = methods
	}
}

// WithParams sets params for url
func WithParams(params url.Values) Option {
	return func(url *URL) {
		url.params = params
	}
}

// WithParamsValue sets params field for url
func WithParamsValue(key, val string) Option {
	return func(url *URL) {
		url.SetParam(key, val)
	}
}

// WithProtocol sets protocol for url
func WithProtocol(proto string) Option {
	return func(url *URL) {
		url.Protocol = proto
	}
}

// WithIp sets ip for url
func WithIp(ip string) Option {
	return func(url *URL) {
		url.Ip = ip
	}
}

// WithPort sets port for url
func WithPort(port string) Option {
	return func(url *URL) {
		url.Port = port
	}
}

// WithPath sets path for url
func WithPath(path string) Option {
	return func(url *URL) {
		url.Path = "/" + strings.TrimPrefix(path, "/")
	}
}

// WithLocation sets location for url
func WithLocation(location string) Option {
	return func(url *URL) {
		url.Location = location
	}
}

// WithToken sets token for url
func WithToken(token string) Option {
	return func(url *URL) {
		if len(token) > 0 {
			value := token
			if strings.ToLower(token) == "true" || strings.ToLower(token) == "default" {
				u, err := uuid.NewV4()
				if err != nil {
					logger.Errorf("could not generator UUID: %v", err)
					return
				}
				value = u.String()
			}
			url.SetParam(constant.TOKEN_KEY, value)
		}
	}
}

// NewURLWithOptions will create a new url with options
func NewURLWithOptions(opts ...Option) *URL {
	newUrl := &URL{}
	for _, opt := range opts {
		opt(newUrl)
	}
	newUrl.Location = newUrl.Ip + ":" + newUrl.Port
	return newUrl
}

// NewURL will create a new url
// the urlString should not be empty
func NewURL(urlString string, opts ...Option) (*URL, error) {
	s := URL{baseUrl: baseUrl{}}
	if urlString == "" {
		return &s, nil
	}

	rawUrlString, err := url.QueryUnescape(urlString)
	if err != nil {
		return &s, perrors.Errorf("url.QueryUnescape(%s),  error{%v}", urlString, err)
	}

	// rawUrlString = "//" + rawUrlString
	if !strings.Contains(rawUrlString, "//") {
		t := URL{baseUrl: baseUrl{}}
		for _, opt := range opts {
			opt(&t)
		}
		rawUrlString = t.Protocol + "://" + rawUrlString
	}

	serviceUrl, urlParseErr := url.Parse(rawUrlString)
	if urlParseErr != nil {
		return &s, perrors.Errorf("url.Parse(url string{%s}),  error{%v}", rawUrlString, err)
	}

	s.params, err = url.ParseQuery(serviceUrl.RawQuery)
	if err != nil {
		return &s, perrors.Errorf("url.ParseQuery(raw url string{%s}),  error{%v}", serviceUrl.RawQuery, err)
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
			return &s, perrors.Errorf("net.SplitHostPort(url.Host{%s}), error{%v}", s.Location, err)
		}
	}
	for _, opt := range opts {
		opt(&s)
	}
	return &s, nil
}

// URLEqual judge @url and @c is equal or not.
func (c *URL) URLEqual(url *URL) bool {
	tmpC := c.Clone()
	tmpC.Ip = ""
	tmpC.Port = ""

	tmpUrl := url.Clone()
	tmpUrl.Ip = ""
	tmpUrl.Port = ""

	cGroup := tmpC.GetParam(constant.GROUP_KEY, "")
	urlGroup := tmpUrl.GetParam(constant.GROUP_KEY, "")
	cKey := tmpC.Key()
	urlKey := tmpUrl.Key()

	if cGroup == constant.ANY_VALUE {
		cKey = strings.Replace(cKey, "group=*", "group="+urlGroup, 1)
	} else if urlGroup == constant.ANY_VALUE {
		urlKey = strings.Replace(urlKey, "group=*", "group="+cGroup, 1)
	}

	// 1. protocol, username, password, ip, port, service name, group, version should be equal
	if cKey != urlKey {
		return false
	}

	// 2. if url contains enabled key, should be true, or *
	if tmpUrl.GetParam(constant.ENABLED_KEY, "true") != "true" && tmpUrl.GetParam(constant.ENABLED_KEY, "") != constant.ANY_VALUE {
		return false
	}

	// TODO :may need add interface key any value condition
	return isMatchCategory(tmpUrl.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY), tmpC.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY))
}

func isMatchCategory(category1 string, category2 string) bool {
	if len(category2) == 0 {
		return category1 == constant.DEFAULT_CATEGORY
	} else if strings.Contains(category2, constant.ANY_VALUE) {
		return true
	} else if strings.Contains(category2, constant.REMOVE_VALUE_PREFIX) {
		return !strings.Contains(category2, constant.REMOVE_VALUE_PREFIX+category1)
	} else {
		return strings.Contains(category2, category1)
	}
}

func (c *URL) String() string {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	var buf strings.Builder
	if len(c.Username) == 0 && len(c.Password) == 0 {
		buf.WriteString(fmt.Sprintf("%s://%s:%s%s?", c.Protocol, c.Ip, c.Port, c.Path))
	} else {
		buf.WriteString(fmt.Sprintf("%s://%s:%s@%s:%s%s?", c.Protocol, c.Username, c.Password, c.Ip, c.Port, c.Path))
	}
	buf.WriteString(c.params.Encode())
	return buf.String()
}

// Key gets key
func (c *URL) Key() string {
	buildString := fmt.Sprintf("%s://%s:%s@%s:%s/?interface=%s&group=%s&version=%s",
		c.Protocol, c.Username, c.Password, c.Ip, c.Port, c.Service(), c.GetParam(constant.GROUP_KEY, ""), c.GetParam(constant.VERSION_KEY, ""))
	return buildString
}

// ServiceKey gets a unique key of a service.
func (c *URL) ServiceKey() string {
	return ServiceKey(c.GetParam(constant.INTERFACE_KEY, strings.TrimPrefix(c.Path, "/")),
		c.GetParam(constant.GROUP_KEY, ""), c.GetParam(constant.VERSION_KEY, ""))
}

func ServiceKey(intf string, group string, version string) string {
	if intf == "" {
		return ""
	}
	buf := &bytes.Buffer{}
	if group != "" {
		buf.WriteString(group)
		buf.WriteString("/")
	}

	buf.WriteString(intf)

	if version != "" && version != "0.0.0" {
		buf.WriteString(":")
		buf.WriteString(version)
	}

	return buf.String()
}

// ColonSeparatedKey
// The format is "{interface}:[version]:[group]"
func (c *URL) ColonSeparatedKey() string {
	intf := c.GetParam(constant.INTERFACE_KEY, strings.TrimPrefix(c.Path, "/"))
	if intf == "" {
		return ""
	}
	var buf strings.Builder
	buf.WriteString(intf)
	buf.WriteString(":")
	version := c.GetParam(constant.VERSION_KEY, "")
	if version != "" && version != "0.0.0" {
		buf.WriteString(version)
	}
	group := c.GetParam(constant.GROUP_KEY, "")
	buf.WriteString(":")
	if group != "" {
		buf.WriteString(group)
	}
	return buf.String()
}

// EncodedServiceKey encode the service key
func (c *URL) EncodedServiceKey() string {
	serviceKey := c.ServiceKey()
	return strings.Replace(serviceKey, "/", "*", 1)
}

// Service gets service
func (c *URL) Service() string {
	service := c.GetParam(constant.INTERFACE_KEY, strings.TrimPrefix(c.Path, "/"))
	if service != "" {
		return service
	} else if c.SubURL != nil {
		service = c.SubURL.GetParam(constant.INTERFACE_KEY, strings.TrimPrefix(c.Path, "/"))
		if service != "" { // if url.path is "" then return suburl's path, special for registry url
			return service
		}
	}
	return ""
}

// AddParam will add the key-value pair
func (c *URL) AddParam(key string, value string) {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	c.params.Add(key, value)
}

// AddParamAvoidNil will add key-value pair
func (c *URL) AddParamAvoidNil(key string, value string) {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	if c.params == nil {
		c.params = url.Values{}
	}
	c.params.Add(key, value)
}

// SetParam will put the key-value pair into url
// usually it should only be invoked when you want to initialized an url
func (c *URL) SetParam(key string, value string) {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	c.params.Set(key, value)
}

// RangeParams will iterate the params
func (c *URL) RangeParams(f func(key, value string) bool) {
	c.paramsLock.RLock()
	defer c.paramsLock.RUnlock()
	for k, v := range c.params {
		if !f(k, v[0]) {
			break
		}
	}
}

// GetParam gets value by key
func (c *URL) GetParam(s string, d string) string {
	c.paramsLock.RLock()
	defer c.paramsLock.RUnlock()
	r := c.params.Get(s)
	if len(r) == 0 {
		r = d
	}
	return r
}

// GetParams gets values
func (c *URL) GetParams() url.Values {
	return c.params
}

// GetParamAndDecoded gets values and decode
func (c *URL) GetParamAndDecoded(key string) (string, error) {
	ruleDec, err := base64.URLEncoding.DecodeString(c.GetParam(key, ""))
	value := string(ruleDec)
	return value, err
}

// GetRawParam gets raw param
func (c *URL) GetRawParam(key string) string {
	switch key {
	case PROTOCOL:
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
		return c.GetParam(key, "")
	}
}

// GetParamBool judge whether @key exists or not
func (c *URL) GetParamBool(key string, d bool) bool {
	r, err := strconv.ParseBool(c.GetParam(key, ""))
	if err != nil {
		return d
	}
	return r
}

// GetParamInt gets int64 value by @key
func (c *URL) GetParamInt(key string, d int64) int64 {
	r, err := strconv.ParseInt(c.GetParam(key, ""), 10, 64)
	if err != nil {
		return d
	}
	return r
}

// GetParamInt32 gets int32 value by @key
func (c *URL) GetParamInt32(key string, d int32) int32 {
	r, err := strconv.ParseInt(c.GetParam(key, ""), 10, 32)
	if err != nil {
		return d
	}
	return int32(r)
}

// GetParamByIntValue gets int value by @key
func (c *URL) GetParamByIntValue(key string, d int) int {
	r, err := strconv.ParseInt(c.GetParam(key, ""), 10, 0)
	if err != nil {
		return d
	}
	return int(r)
}

// GetMethodParamInt gets int method param
func (c *URL) GetMethodParamInt(method string, key string, d int64) int64 {
	r, err := strconv.ParseInt(c.GetParam("methods."+method+"."+key, ""), 10, 64)
	if err != nil {
		return d
	}
	return r
}

// GetMethodParamIntValue gets int method param
func (c *URL) GetMethodParamIntValue(method string, key string, d int) int {
	r, err := strconv.ParseInt(c.GetParam("methods."+method+"."+key, ""), 10, 0)
	if err != nil {
		return d
	}
	return int(r)
}

// GetMethodParamInt64 gets int64 method param
func (c *URL) GetMethodParamInt64(method string, key string, d int64) int64 {
	r := c.GetMethodParamInt(method, key, math.MinInt64)
	if r == math.MinInt64 {
		return c.GetParamInt(key, d)
	}
	return r
}

// GetMethodParam gets method param
func (c *URL) GetMethodParam(method string, key string, d string) string {
	r := c.GetParam("methods."+method+"."+key, "")
	if r == "" {
		r = d
	}
	return r
}

// GetMethodParamBool judge whether @method param exists or not
func (c *URL) GetMethodParamBool(method string, key string, d bool) bool {
	r := c.GetParamBool("methods."+method+"."+key, d)
	return r
}

// SetParams will put all key-value pair into url.
// 1. if there already has same key, the value will be override
// 2. it's not thread safe
// 3. think twice when you want to invoke this method
func (c *URL) SetParams(m url.Values) {
	for k := range m {
		c.SetParam(k, m.Get(k))
	}
}

// ToMap transfer URL to Map
func (c *URL) ToMap() map[string]string {
	paramsMap := make(map[string]string)

	c.RangeParams(func(key, value string) bool {
		paramsMap[key] = value
		return true
	})

	if c.Protocol != "" {
		paramsMap[PROTOCOL] = c.Protocol
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
		paramsMap[PROTOCOL] = c.Protocol
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
// TODO configuration merge, in the future , the configuration center's config should merge too.

// MergeUrl will merge those two url
// the result is based on serviceUrl, and the key which si only contained in referenceUrl
// will be added into result.
// for example, if serviceUrl contains params (a1->v1, b1->v2) and referenceUrl contains params(a2->v3, b1 -> v4)
// the params of result will be (a1->v1, b1->v2, a2->v3).
// You should notice that the value of b1 is v2, not v4.
// due to URL is not thread-safe, so this method is not thread-safe
func MergeUrl(serviceUrl *URL, referenceUrl *URL) *URL {
	mergedUrl := serviceUrl.Clone()

	// iterator the referenceUrl if serviceUrl not have the key ,merge in
	referenceUrl.RangeParams(func(key, value string) bool {
		if v := mergedUrl.GetParam(key, ""); len(v) == 0 {
			mergedUrl.SetParam(key, value)
		}
		return true
	})
	// loadBalance,cluster,retries strategy config
	methodConfigMergeFcn := mergeNormalParam(mergedUrl, referenceUrl, []string{constant.LOADBALANCE_KEY, constant.CLUSTER_KEY, constant.RETRIES_KEY, constant.TIMEOUT_KEY})

	// remote timestamp
	if v := serviceUrl.GetParam(constant.TIMESTAMP_KEY, ""); len(v) > 0 {
		mergedUrl.SetParam(constant.REMOTE_TIMESTAMP_KEY, v)
		mergedUrl.SetParam(constant.TIMESTAMP_KEY, referenceUrl.GetParam(constant.TIMESTAMP_KEY, ""))
	}

	// finally execute methodConfigMergeFcn
	for _, method := range referenceUrl.Methods {
		for _, fcn := range methodConfigMergeFcn {
			fcn("methods." + method)
		}
	}

	return mergedUrl
}

// Clone will copy the url
func (c *URL) Clone() *URL {
	newUrl := &URL{}
	copier.Copy(newUrl, c)
	newUrl.params = url.Values{}
	c.RangeParams(func(key, value string) bool {
		newUrl.SetParam(key, value)
		return true
	})
	return newUrl
}

func (c *URL) CloneExceptParams(excludeParams *gxset.HashSet) *URL {
	newUrl := &URL{}
	copier.Copy(newUrl, c)
	newUrl.params = url.Values{}
	c.RangeParams(func(key, value string) bool {
		if !excludeParams.Contains(key) {
			newUrl.SetParam(key, value)
		}
		return true
	})
	return newUrl
}

func (c *URL) Compare(comp cm.Comparator) int {
	a := c.String()
	b := comp.(*URL).String()
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

// Copy url based on the reserved parameter's keys.
func (c *URL) CloneWithParams(reserveParams []string) *URL {
	params := url.Values{}
	for _, reserveParam := range reserveParams {
		v := c.GetParam(reserveParam, "")
		if len(v) != 0 {
			params.Set(reserveParam, v)
		}
	}

	return NewURLWithOptions(
		WithProtocol(c.Protocol),
		WithUsername(c.Username),
		WithPassword(c.Password),
		WithIp(c.Ip),
		WithPort(c.Port),
		WithPath(c.Path),
		WithMethods(c.Methods),
		WithParams(params),
	)
}

// IsEquals compares if two URLs equals with each other. Excludes are all parameter keys which should ignored.
func IsEquals(left *URL, right *URL, excludes ...string) bool {
	if left.Ip != right.Ip || left.Port != right.Port {
		return false
	}

	leftMap := left.ToMap()
	rightMap := right.ToMap()
	for _, exclude := range excludes {
		delete(leftMap, exclude)
		delete(rightMap, exclude)
	}

	if len(leftMap) != len(rightMap) {
		return false
	}

	for lk, lv := range leftMap {
		if rv, ok := rightMap[lk]; !ok {
			return false
		} else if lv != rv {
			return false
		}
	}

	return true
}

func mergeNormalParam(mergedUrl *URL, referenceUrl *URL, paramKeys []string) []func(method string) {
	methodConfigMergeFcn := make([]func(method string), 0, len(paramKeys))
	for _, paramKey := range paramKeys {
		if v := referenceUrl.GetParam(paramKey, ""); len(v) > 0 {
			mergedUrl.SetParam(paramKey, v)
		}
		methodConfigMergeFcn = append(methodConfigMergeFcn, func(method string) {
			if v := referenceUrl.GetParam(method+"."+paramKey, ""); len(v) > 0 {
				mergedUrl.SetParam(method+"."+paramKey, v)
			}
		})
	}
	return methodConfigMergeFcn
}

// URLSlice will be used to sort URL instance
// Instances will be order by URL.String()
type URLSlice []*URL

// nolint
func (s URLSlice) Len() int {
	return len(s)
}

// nolint
func (s URLSlice) Less(i, j int) bool {
	return s[i].String() < s[j].String()
}

// nolint
func (s URLSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
