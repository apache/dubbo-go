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
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

import (
	cm "github.com/Workiva/go-datastructures/common"
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/jinzhu/copier"
	perrors "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
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
	// CompareURLEqualFunc compare two URL is equal
	compareURLEqualFunc CompareURLEqualFunc
)

func init() {
	compareURLEqualFunc = defaultCompareURLEqual
}

// nolint
type RoleType int

func (t RoleType) String() string {
	return DubboNodes[t]
}

// Role returns role by @RoleType
func (t RoleType) Role() string {
	return DubboRole[t]
}

type baseURL struct {
	Protocol string
	Location string // ip+port
	Ip       string
	Port     string

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

// URL thread-safe. but this URL should not be copied.
// we fail to define this struct to be immutable object.
// but, those method which will update the URL, including SetParam, SetParams
// are only allowed to be invoked in creating URL instance
// Please keep in mind that this struct is immutable after it has been created and initialized.
type URL struct {
	noCopy noCopy

	baseURL
	// url.Values is not safe map, add to avoid concurrent map read and map write error
	paramsLock sync.RWMutex
	params     url.Values

	Path     string // like  /com.ikurento.dubbo.UserProvider
	Username string
	Password string
	Methods  []string
	// special for registry
	SubURL *URL
}

// Option accepts URL
// Option will define a function of handling URL
type Option func(*URL)

// WithUsername sets username for URL
func WithUsername(username string) Option {
	return func(url *URL) {
		url.Username = username
	}
}

// WithPassword sets password for URL
func WithPassword(pwd string) Option {
	return func(url *URL) {
		url.Password = pwd
	}
}

// WithMethods sets methods for URL
func WithMethods(methods []string) Option {
	return func(url *URL) {
		url.Methods = methods
	}
}

// WithParams sets params for URL
func WithParams(params url.Values) Option {
	return func(url *URL) {
		url.params = params
	}
}

// WithParamsValue sets params field for URL
func WithParamsValue(key, val string) Option {
	return func(url *URL) {
		url.SetParam(key, val)
	}
}

// WithProtocol sets protocol for URL
func WithProtocol(proto string) Option {
	return func(url *URL) {
		url.Protocol = proto
	}
}

// WithIp sets ip for URL
func WithIp(ip string) Option {
	return func(url *URL) {
		url.Ip = ip
	}
}

// WithPort sets port for URL
func WithPort(port string) Option {
	return func(url *URL) {
		url.Port = port
	}
}

// WithPath sets path for URL
func WithPath(path string) Option {
	return func(url *URL) {
		url.Path = "/" + strings.TrimPrefix(path, "/")
	}
}

// WithLocation sets location for URL
func WithLocation(location string) Option {
	return func(url *URL) {
		url.Location = location
	}
}

// WithToken sets token for URL
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

// NewURLWithOptions will create a new URL with options
func NewURLWithOptions(opts ...Option) *URL {
	newURL := &URL{}
	for _, opt := range opts {
		opt(newURL)
	}
	newURL.Location = newURL.Ip + ":" + newURL.Port
	return newURL
}

// NewURL will create a new URL
// the urlString should not be empty
func NewURL(urlString string, opts ...Option) (*URL, error) {
	s := URL{baseURL: baseURL{}}
	if urlString == "" {
		return &s, nil
	}

	rawURLString, err := url.QueryUnescape(urlString)
	if err != nil {
		return &s, perrors.Errorf("URL.QueryUnescape(%s),  error{%v}", urlString, err)
	}

	// rawURLString = "//" + rawURLString
	if !strings.Contains(rawURLString, "//") {
		t := URL{baseURL: baseURL{}}
		for _, opt := range opts {
			opt(&t)
		}
		rawURLString = t.Protocol + "://" + rawURLString
	}

	serviceURL, urlParseErr := url.Parse(rawURLString)
	if urlParseErr != nil {
		return &s, perrors.Errorf("URL.Parse(URL string{%s}),  error{%v}", rawURLString, err)
	}

	s.params, err = url.ParseQuery(serviceURL.RawQuery)
	if err != nil {
		return &s, perrors.Errorf("URL.ParseQuery(raw URL string{%s}),  error{%v}", serviceURL.RawQuery, err)
	}

	s.PrimitiveURL = urlString
	s.Protocol = serviceURL.Scheme
	s.Username = serviceURL.User.Username()
	s.Password, _ = serviceURL.User.Password()
	s.Location = serviceURL.Host
	s.Path = serviceURL.Path
	if strings.Contains(s.Location, ":") {
		s.Ip, s.Port, err = net.SplitHostPort(s.Location)
		if err != nil {
			return &s, perrors.Errorf("net.SplitHostPort(URL.Host{%s}), error{%v}", s.Location, err)
		}
	}
	for _, opt := range opts {
		opt(&s)
	}
	return &s, nil
}

func MatchKey(serviceKey string, protocol string) string {
	return serviceKey + ":" + protocol
}

// Group get group
func (c *URL) Group() string {
	return c.GetParam(constant.GROUP_KEY, "")
}

// Version get group
func (c *URL) Version() string {
	return c.GetParam(constant.VERSION_KEY, "")
}

// URLEqual judge @URL and @c is equal or not.
func (c *URL) URLEqual(url *URL) bool {
	tmpC := c.Clone()
	tmpC.Ip = ""
	tmpC.Port = ""

	tmpURL := url.Clone()
	tmpURL.Ip = ""
	tmpURL.Port = ""

	cGroup := tmpC.GetParam(constant.GROUP_KEY, "")
	urlGroup := tmpURL.GetParam(constant.GROUP_KEY, "")
	cKey := tmpC.Key()
	urlKey := tmpURL.Key()

	if cGroup == constant.ANY_VALUE {
		cKey = strings.Replace(cKey, "group=*", "group="+urlGroup, 1)
	} else if urlGroup == constant.ANY_VALUE {
		urlKey = strings.Replace(urlKey, "group=*", "group="+cGroup, 1)
	}

	// 1. protocol, username, password, ip, port, service name, group, version should be equal
	if cKey != urlKey {
		return false
	}

	// 2. if URL contains enabled key, should be true, or *
	if tmpURL.GetParam(constant.ENABLED_KEY, "true") != "true" && tmpURL.GetParam(constant.ENABLED_KEY, "") != constant.ANY_VALUE {
		return false
	}

	// TODO :may need add interface key any value condition
	return isMatchCategory(tmpURL.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY), tmpC.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY))
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

//GetCacheInvokerMapKey get directory cacheInvokerMap key
func (c *URL) GetCacheInvokerMapKey() string {
	urlNew, _ := NewURL(c.PrimitiveURL)

	buildString := fmt.Sprintf("%s://%s:%s@%s:%s/?interface=%s&group=%s&version=%s&timestamp=%s",
		c.Protocol, c.Username, c.Password, c.Ip, c.Port, c.Service(), c.GetParam(constant.GROUP_KEY, ""),
		c.GetParam(constant.VERSION_KEY, ""), urlNew.GetParam(constant.TIMESTAMP_KEY, ""))
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
		if service != "" { // if URL.path is "" then return suburl's path, special for registry URL
			return service
		}
	}
	return ""
}

// AddParam will add the key-value pair
func (c *URL) AddParam(key string, value string) {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	if c.params == nil {
		c.params = url.Values{}
	}
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

// SetParam will put the key-value pair into URL
// usually it should only be invoked when you want to initialized an URL
func (c *URL) SetParam(key string, value string) {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	if c.params == nil {
		c.params = url.Values{}
	}
	c.params.Set(key, value)
}

// DelParam will delete the given key from the URL
func (c *URL) DelParam(key string) {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	if c.params != nil {
		c.params.Del(key)
	}
}

// ReplaceParams will replace the URL.params
// usually it should only be invoked when you want to modify an URL, such as MergeURL
func (c *URL) ReplaceParams(param url.Values) {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	c.params = param
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

	var r string
	if len(c.params) > 0 {
		r = c.params.Get(s)
	}
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

// SetParams will put all key-value pair into URL.
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
//  in this function we should merge the reference local URL config into the service URL from registry.
// TODO configuration merge, in the future , the configuration center's config should merge too.

// MergeURL will merge those two URL
// the result is based on serviceURL, and the key which si only contained in referenceURL
// will be added into result.
// for example, if serviceURL contains params (a1->v1, b1->v2) and referenceURL contains params(a2->v3, b1 -> v4)
// the params of result will be (a1->v1, b1->v2, a2->v3).
// You should notice that the value of b1 is v2, not v4.
// due to URL is not thread-safe, so this method is not thread-safe
func MergeURL(serviceURL *URL, referenceURL *URL) *URL {
	// After Clone, it is a new URL that there is no thread safe issue.
	mergedURL := serviceURL.Clone()
	params := mergedURL.GetParams()
	// iterator the referenceURL if serviceURL not have the key ,merge in
	// referenceURL usually will not changed. so change RangeParams to GetParams to avoid the string value copy.// Group get group
	for key, value := range referenceURL.GetParams() {
		if v := mergedURL.GetParam(key, ""); len(v) == 0 {
			if len(value) > 0 {
				params[key] = value
			}
		}
	}

	// loadBalance,cluster,retries strategy config
	methodConfigMergeFcn := mergeNormalParam(params, referenceURL, []string{constant.LOADBALANCE_KEY, constant.CLUSTER_KEY, constant.RETRIES_KEY, constant.TIMEOUT_KEY})

	// remote timestamp
	if v := serviceURL.GetParam(constant.TIMESTAMP_KEY, ""); len(v) > 0 {
		params[constant.REMOTE_TIMESTAMP_KEY] = []string{v}
		params[constant.TIMESTAMP_KEY] = []string{referenceURL.GetParam(constant.TIMESTAMP_KEY, "")}
	}

	// finally execute methodConfigMergeFcn
	for _, method := range referenceURL.Methods {
		for _, fcn := range methodConfigMergeFcn {
			fcn("methods." + method)
		}
	}
	// In this way, we will raise some performance.
	mergedURL.ReplaceParams(params)
	return mergedURL
}

// Clone will copy the URL
func (c *URL) Clone() *URL {
	newURL := &URL{}
	if err := copier.Copy(newURL, c); err != nil {
		// this is impossible
		return newURL
	}
	newURL.params = url.Values{}
	c.RangeParams(func(key, value string) bool {
		newURL.SetParam(key, value)
		return true
	})

	return newURL
}

func (c *URL) CloneExceptParams(excludeParams *gxset.HashSet) *URL {
	newURL := &URL{}
	if err := copier.Copy(newURL, c); err != nil {
		// this is impossible
		return newURL
	}
	newURL.params = url.Values{}
	c.RangeParams(func(key, value string) bool {
		if !excludeParams.Contains(key) {
			newURL.SetParam(key, value)
		}
		return true
	})
	return newURL
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

// Copy URL based on the reserved parameter's keys.
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
	if (left == nil && right != nil) || (right == nil && left != nil) {
		return false
	}
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

func mergeNormalParam(params url.Values, referenceURL *URL, paramKeys []string) []func(method string) {
	methodConfigMergeFcn := make([]func(method string), 0, len(paramKeys))
	for _, paramKey := range paramKeys {
		if v := referenceURL.GetParam(paramKey, ""); len(v) > 0 {
			params[paramKey] = []string{v}
		}
		methodConfigMergeFcn = append(methodConfigMergeFcn, func(method string) {
			if v := referenceURL.GetParam(method+"."+paramKey, ""); len(v) > 0 {
				params[method+"."+paramKey] = []string{v}
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

type CompareURLEqualFunc func(l *URL, r *URL, excludeParam ...string) bool

func defaultCompareURLEqual(l *URL, r *URL, excludeParam ...string) bool {
	return IsEquals(l, r, excludeParam...)
}

func SetCompareURLEqualFunc(f CompareURLEqualFunc) {
	compareURLEqualFunc = f
}

func GetCompareURLEqualFunc() CompareURLEqualFunc {
	return compareURLEqualFunc
}
