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
	"math"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	cm "github.com/Workiva/go-datastructures/common"

	gxset "github.com/dubbogo/gost/container/set"

	"github.com/google/uuid"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// dubbo role type constant
const (
	CONSUMER = iota
	CONFIGURATOR
	ROUTER
	PROVIDER
)

// PROTOCOL is the canonical key for the protocol in URL maps/params.
//
// Deprecated: use constant.ProtocolKey instead. Kept as an alias to preserve
// backward compatibility for downstream users.
const PROTOCOL = constant.ProtocolKey

var (
	DubboNodes          = [...]string{"consumers", "configurators", "routers", "providers"} // Dubbo service node
	DubboRole           = [...]string{"consumer", "", "routers", "provider"}                // Dubbo service role
	compareURLEqualFunc CompareURLEqualFunc                                                 // function to compare two URL is equal
)

func init() {
	compareURLEqualFunc = defaultCompareURLEqual
}

// RoleType defines the role of the node in Dubbo service topology.
// The values are indexes for arrays like DubboNodes and DubboRole.
type RoleType int

func (t RoleType) String() string {
	return DubboNodes[t]
}

// Role returns role by @RoleType
func (t RoleType) Role() string {
	return DubboRole[t]
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

	Protocol string
	Location string // ip+port
	Ip       string
	Port     string

	PrimitiveURL string
	primitiveTS  string // primitiveTS caches the original provider timestamp so GetCacheInvokerMapKey stays stable when configurator refreshes rebuild invoker URLs.
	// url.Values is not safe map, add to avoid concurrent map read and map write error
	paramsLock sync.RWMutex
	params     url.Values

	Path     string // like  /com.ikurento.dubbo.UserProvider
	Username string
	Password string
	Methods  []string

	attributesLock sync.RWMutex
	// attributes should not be transported
	attributes map[string]any `hessian:"-"`
	// special for registry
	SubURL *URL
}

// JavaClassName POJO for URL
func (c *URL) JavaClassName() string {
	return "org.apache.dubbo.common.URL"
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

// WithParams deep copy the params in the argument into params of the target URL
func WithParams(params url.Values) Option {
	return func(url *URL) {
		url.SetParams(params)
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

// WithInterface sets interface param for URL
func WithInterface(v string) Option {
	return func(url *URL) {
		url.SetParam(constant.InterfaceKey, v)
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
				u, _ := uuid.NewUUID()
				value = u.String()
			}
			url.SetParam(constant.TokenKey, value)
		}
	}
}

// WithAttribute sets attribute for URL
func WithAttribute(key string, attribute any) Option {
	return func(url *URL) {
		if url.attributes == nil {
			url.attributes = make(map[string]any)
		}
		url.attributes[key] = attribute
	}
}

// WithWeight sets weight for URL
func WithWeight(weight int64) Option {
	return func(url *URL) {
		if weight > 0 {
			url.SetParam(constant.WeightKey, strconv.FormatInt(weight, 10))
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
	s := URL{}
	if urlString == "" {
		return &s, nil
	}

	// zookeeper, nacos, metadata address with multi addr trim space
	urlStringTrim := func(s string) string {
		parts := strings.Split(s, ",")
		var nonEmptyParts []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				nonEmptyParts = append(nonEmptyParts, trimmed)
			}
		}
		return strings.Join(nonEmptyParts, ",")
	}
	urlString = urlStringTrim(urlString)

	rawURLString, err := url.QueryUnescape(urlString)
	if err != nil {
		return &s, perrors.Errorf("URL.QueryUnescape(%s),  error{%v}", urlString, err)
	}

	// rawURLString = "//" + rawURLString
	if !strings.Contains(rawURLString, "//") {
		t := URL{}
		for _, opt := range opts {
			opt(&t)
		}
		rawURLString = t.Protocol + "://" + rawURLString
	}

	serviceURL, urlParseErr := url.Parse(rawURLString)
	if urlParseErr != nil {
		return &s, perrors.Errorf("URL.Parse(URL string{%s}),  error{%v}", rawURLString, urlParseErr)
	}

	s.params, err = url.ParseQuery(serviceURL.RawQuery)
	if err != nil {
		return &s, perrors.Errorf("URL.ParseQuery(raw URL string{%s}),  error{%v}", serviceURL.RawQuery, err)
	}

	s.PrimitiveURL = urlString
	s.primitiveTS = s.params.Get(constant.TimestampKey)
	s.Protocol = serviceURL.Scheme
	s.Username = serviceURL.User.Username()
	s.Password, _ = serviceURL.User.Password()
	s.Location = serviceURL.Host
	s.Path = serviceURL.Path
	for _, location := range strings.Split(s.Location, ",") {
		location = strings.Trim(location, " ")
		if strings.Contains(location, ":") {
			s.Ip, s.Port, err = net.SplitHostPort(location)
			if err != nil {
				return &s, perrors.Errorf("net.SplitHostPort(url.Host{%s}), error{%v}", s.Location, err)
			}
			break
		}
	}
	for _, opt := range opts {
		opt(&s)
	}
	if s.params.Get(constant.RegistryGroupKey) != "" {
		s.PrimitiveURL = strings.Join([]string{s.PrimitiveURL, s.params.Get(constant.RegistryGroupKey)}, constant.PathSeparator)
	}
	return &s, nil
}

func MatchKey(serviceKey string, protocol string) string {
	return serviceKey + ":" + protocol
}

// Group get group
func (c *URL) Group() string {
	return c.GetParam(constant.GroupKey, "")
}

// Interface get interface
func (c *URL) Interface() string {
	return c.GetParam(constant.InterfaceKey, "")
}

// Version get group
func (c *URL) Version() string {
	return c.GetParam(constant.VersionKey, "")
}

// Address with format "ip:port"
func (c *URL) Address() string {
	if c.Port == "" {
		return c.Ip
	}
	return c.Ip + ":" + c.Port
}

// URLEqual judge @URL and @c is equal or not.
func (c *URL) URLEqual(url *URL) bool {
	tmpC := c.Clone()
	tmpC.Ip = ""
	tmpC.Port = ""

	tmpURL := url.Clone()
	tmpURL.Ip = ""
	tmpURL.Port = ""

	cGroup := tmpC.GetParam(constant.GroupKey, "")
	urlGroup := tmpURL.GetParam(constant.GroupKey, "")
	cKey := tmpC.Key()
	urlKey := tmpURL.Key()

	if cGroup == constant.AnyValue {
		cKey = strings.Replace(cKey, "group=*", "group="+urlGroup, 1)
	} else if urlGroup == constant.AnyValue {
		urlKey = strings.Replace(urlKey, "group=*", "group="+cGroup, 1)
	}

	// 1. protocol, username, password, ip, port, service name, group, version should be equal
	if cKey != urlKey {
		return false
	}

	// 2. if URL contains enabled key, should be true, or *
	if tmpURL.GetParam(constant.EnabledKey, "true") != "true" && tmpURL.GetParam(constant.EnabledKey, "") != constant.AnyValue {
		return false
	}

	// TODO :may need add interface key any value condition
	return isMatchCategory(tmpURL.GetParam(constant.CategoryKey, constant.DefaultCategory), tmpC.GetParam(constant.CategoryKey, constant.DefaultCategory))
}

func isMatchCategory(category1 string, category2 string) bool {
	if len(category2) == 0 {
		return category1 == constant.DefaultCategory
	} else if strings.Contains(category2, constant.AnyValue) {
		return true
	} else if strings.Contains(category2, constant.RemoveValuePrefix) {
		return !strings.Contains(category2, constant.RemoveValuePrefix+category1)
	} else {
		return strings.Contains(category2, category1)
	}
}

func (c *URL) String() string {
	c.paramsLock.RLock()
	defer c.paramsLock.RUnlock()

	encodedParams := c.params.Encode()
	var buf strings.Builder

	size := len(c.Protocol) + len("://") + len(c.Ip) + len(":") + len(c.Port) + len(c.Path)
	if len(c.Username) != 0 || len(c.Password) != 0 {
		size += len(c.Username) + len(":") + len(c.Password) + len("@")
	}
	if encodedParams != "" {
		size += len("?") + len(encodedParams)
	}

	buf.Grow(size)
	buf.WriteString(c.Protocol)
	buf.WriteString("://")
	if len(c.Username) != 0 || len(c.Password) != 0 {
		buf.WriteString(c.Username)
		buf.WriteString(":")
		buf.WriteString(c.Password)
		buf.WriteString("@")
	}
	buf.WriteString(c.Ip)
	buf.WriteString(":")
	buf.WriteString(c.Port)
	buf.WriteString(c.Path)
	if encodedParams != "" {
		buf.WriteByte('?')
		buf.WriteString(encodedParams)
	}
	return buf.String()
}

// Key gets key
func (c *URL) Key() string {
	var buf strings.Builder
	service := c.Service()
	group := c.GetParam(constant.GroupKey, "")
	version := c.GetParam(constant.VersionKey, "")

	size := len(c.Protocol) + len("://") + len(c.Username) +
		len(":") + len(c.Password) + len("@") +
		len(c.Ip) + len(":") + len(c.Port) +
		len("/?interface=") + len(service) +
		len("&group=") + len(group) +
		len("&version=") + len(version)

	buf.Grow(size)
	buf.WriteString(c.Protocol)
	buf.WriteString("://")
	buf.WriteString(c.Username)
	buf.WriteString(":")
	buf.WriteString(c.Password)
	buf.WriteString("@")
	buf.WriteString(c.Ip)
	buf.WriteString(":")
	buf.WriteString(c.Port)
	buf.WriteString("/?interface=")
	buf.WriteString(service)
	buf.WriteString("&group=")
	buf.WriteString(group)
	buf.WriteString("&version=")
	buf.WriteString(version)

	return buf.String()
}

// GetCacheInvokerMapKey get directory cacheInvokerMap key
func (c *URL) GetCacheInvokerMapKey() string {
	var buf strings.Builder
	service := c.Service()
	group := c.GetParam(constant.GroupKey, "")
	version := c.GetParam(constant.VersionKey, "")
	meshClusterID := c.GetParam(constant.MeshClusterIDKey, "")

	size := len(c.Protocol) + len("://") + len(c.Username) +
		len(":") + len(c.Password) + len("@") +
		len(c.Ip) + len(":") + len(c.Port) +
		len("/?interface=") + len(service) +
		len("&group=") + len(group) +
		len("&version=") + len(version) +
		len("&timestamp=") + len(c.primitiveTS) +
		len("&meshClusterID=") + len(meshClusterID)

	buf.Grow(size)
	buf.WriteString(c.Protocol)
	buf.WriteString("://")
	buf.WriteString(c.Username)
	buf.WriteString(":")
	buf.WriteString(c.Password)
	buf.WriteString("@")
	buf.WriteString(c.Ip)
	buf.WriteString(":")
	buf.WriteString(c.Port)
	buf.WriteString("/?interface=")
	buf.WriteString(service)
	buf.WriteString("&group=")
	buf.WriteString(group)
	buf.WriteString("&version=")
	buf.WriteString(version)
	buf.WriteString("&timestamp=")
	buf.WriteString(c.primitiveTS)
	buf.WriteString("&meshClusterID=")
	buf.WriteString(meshClusterID)

	return buf.String()
}

// ServiceKey gets a unique key of a service.
func (c *URL) ServiceKey() string {
	return ServiceKey(
		c.GetParam(constant.InterfaceKey, strings.TrimPrefix(c.Path, constant.PathSeparator)),
		c.GetParam(constant.GroupKey, ""), c.GetParam(constant.VersionKey, ""),
	)
}

func ServiceKey(intf string, group string, version string) string {
	if intf == "" {
		return ""
	}
	var buf strings.Builder
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

// ParseServiceKey gets interface, group and version from service key
func ParseServiceKey(serviceKey string) (string, string, string) {
	var (
		group   string
		version string
	)
	if serviceKey == "" {
		return "", "", ""
	}
	// get group if it exists
	sepIndex := strings.Index(serviceKey, constant.PathSeparator)
	if sepIndex != -1 {
		group = serviceKey[:sepIndex]
		serviceKey = serviceKey[sepIndex+1:]
	}
	// get version if it exists
	sepIndex = strings.LastIndex(serviceKey, constant.KeySeparator)
	if sepIndex != -1 {
		version = serviceKey[sepIndex+1:]
		serviceKey = serviceKey[:sepIndex]
	}

	return serviceKey, group, version
}

// IsAnyCondition judges if is any condition
func IsAnyCondition(intf, group, version string, serviceURL *URL) bool {
	matchCondition := func(pattern, actual string) bool {
		return pattern == constant.AnyValue || pattern == actual
	}

	return matchCondition(intf, serviceURL.Service()) &&
		matchCondition(group, serviceURL.Group()) &&
		matchCondition(version, serviceURL.Version())
}

// ColonSeparatedKey
// The format is "{interface}:[version]:[group]"
func (c *URL) ColonSeparatedKey() string {
	intf := c.GetParam(constant.InterfaceKey, strings.TrimPrefix(c.Path, "/"))
	if intf == "" {
		return ""
	}
	var buf strings.Builder
	buf.WriteString(intf)
	buf.WriteString(":")
	version := c.GetParam(constant.VersionKey, "")
	if version != "" && version != "0.0.0" {
		buf.WriteString(version)
	}
	group := c.GetParam(constant.GroupKey, "")
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
	service := c.GetParam(constant.InterfaceKey, strings.TrimPrefix(c.Path, "/"))
	if service != "" {
		return service
	} else if c.SubURL != nil {
		service = c.SubURL.GetParam(constant.InterfaceKey, strings.TrimPrefix(c.Path, "/"))
		if service != "" { // if URL.path is "" then return suburl's path, special for registry URL
			return service
		}
	}
	return ""
}

// AppendParam appends the key-value pair without replacing existing values.
func (c *URL) AppendParam(key, value string) {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	if c.params == nil {
		c.params = url.Values{}
	}
	c.params.Add(key, value)
}

// AddParam will add the key-value pair.
// Deprecated: use SetParam to replace an existing value or AppendParam to preserve multiple values.
func (c *URL) AddParam(key, value string) {
	c.AppendParam(key, value)
}

// AddParamAvoidNil will add key-value pair.
// Deprecated: use AppendParam instead.
func (c *URL) AddParamAvoidNil(key, value string) {
	c.AppendParam(key, value)
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

// CompareAndSwapParam will set the key-value pair into URL when the current value equals the expected value.
// It returns true if the value was set successfully, false otherwise.
// This is a thread-safe compare-and-swap operation.
func (c *URL) CompareAndSwapParam(key string, expected string, newValue string) bool {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	if c.params == nil {
		c.params = url.Values{}
	}
	if c.params.Get(key) == expected {
		c.params.Set(key, newValue)
		return true
	}
	return false
}

func (c *URL) SetAttribute(key string, value any) {
	c.attributesLock.Lock()
	defer c.attributesLock.Unlock()
	if c.attributes == nil {
		c.attributes = make(map[string]any)
	}
	c.attributes[key] = value
}

func (c *URL) GetAttribute(key string) (any, bool) {
	c.attributesLock.RLock()
	defer c.attributesLock.RUnlock()
	r, ok := c.attributes[key]
	return r, ok
}

func (c *URL) DeleteAttribute(key string) {
	c.attributesLock.Lock()
	defer c.attributesLock.Unlock()
	if c.attributes != nil {
		delete(c.attributes, key)
	}
}

func (c *URL) ClearAttributes() {
	c.attributesLock.Lock()
	defer c.attributesLock.Unlock()
	c.attributes = make(map[string]any)
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

// GetNonDefaultParam gets value by key, return nil,false if no value found mapping to the key
func (c *URL) GetNonDefaultParam(s string) (string, bool) {
	c.paramsLock.RLock()
	defer c.paramsLock.RUnlock()

	var r string
	if len(c.params) > 0 {
		r = c.params.Get(s)
	}

	return r, r != ""
}

// GetParams gets values.
// Deprecated: use CopyParams instead.
func (c *URL) GetParams() url.Values {
	return c.CopyParams()
}

func copyURLValues(src url.Values) url.Values {
	if src == nil {
		return nil
	}

	dst := make(url.Values, len(src))
	for k, vs := range src {
		copied := make([]string, len(vs))
		copy(copied, vs)
		dst[k] = copied
	}

	return dst
}

func valuesHasNonDefaultParam(values url.Values, key string) bool {
	if len(values) == 0 {
		return false
	}
	return values.Get(key) != ""
}

// CopyParams returns a deep copy of params.
func (c *URL) CopyParams() url.Values {
	c.paramsLock.RLock()
	defer c.paramsLock.RUnlock()

	return copyURLValues(c.params)
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
	case constant.ProtocolKey:
		return c.Protocol
	case constant.UsernameKey:
		return c.Username
	case constant.HostKey:
		host, _ := parseLocation(c.Location)
		return host
	case constant.PasswordKey:
		return c.Password
	case constant.PortKey:
		return c.Port
	case constant.PathKey:
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

// SetParams copies all key-value pairs into URL.
// 1. if there already has same key, the value will be override
// 2. this method acquires a write lock and deep-copies the provided values to avoid data races
// 3. think twice when you want to invoke this method
func (c *URL) SetParams(m url.Values) {
	c.paramsLock.Lock()
	defer c.paramsLock.Unlock()
	if c.params == nil {
		c.params = url.Values{}
	}
	for k, vs := range m {
		if len(vs) == 0 {
			delete(c.params, k)
			continue
		}
		copied := make([]string, len(vs))
		copy(copied, vs)
		c.params[k] = copied
	}
}

// ToMap transfer URL to Map
func (c *URL) ToMap() map[string]string {
	// Pre-calculate capacity to avoid map growth
	c.paramsLock.RLock()
	paramCount := len(c.params)
	c.paramsLock.RUnlock()

	// Count non-empty scalar fields
	capacity := paramCount
	if c.Protocol != "" {
		capacity++
	}
	if c.Username != "" {
		capacity++
	}
	if c.Password != "" {
		capacity++
	}
	if c.Location != "" {
		capacity += 2 // host + port
	}
	if c.Path != "" {
		capacity++
	}

	if capacity == 0 {
		return nil
	}

	paramsMap := make(map[string]string, capacity)

	c.RangeParams(
		func(key, value string) bool {
			paramsMap[key] = value
			return true
		},
	)

	if c.Protocol != "" {
		paramsMap[constant.ProtocolKey] = c.Protocol
	}
	if c.Username != "" {
		paramsMap[constant.UsernameKey] = c.Username
	}
	if c.Password != "" {
		paramsMap[constant.PasswordKey] = c.Password
	}
	if c.Location != "" {
		host, port := parseLocation(c.Location)
		paramsMap[constant.HostKey] = host
		paramsMap[constant.PortKey] = port
	}
	if c.Path != "" {
		paramsMap[constant.PathKey] = c.Path
	}
	return paramsMap
}

// configuration  > reference config >service config
//  in this function we should merge the reference local URL config into the service URL from registry.
// TODO configuration merge, in the future , the configuration center's config should merge too.

// MergeURL will merge those two URL
// the result is based on c, and the key which si only contained in anotherUrl
// will be added into result.
// for example, if c contains params (a1->v1, b1->v2) and anotherUrl contains params(a2->v3, b1 -> v4)
// the params of result will be (a1->v1, b1->v2, a2->v3).
// You should notice that the value of b1 is v2, not v4
// except constant.LoadbalanceKey, constant.ClusterKey, constant.RetriesKey, constant.TimeoutKey.
// due to URL is not thread-safe, so this method is not thread-safe
func (c *URL) MergeURL(anotherUrl *URL) *URL {
	// After Clone, it is a new URL that there is no thread safe issue.
	mergedURL := c.Clone()
	params := mergedURL.params
	if params == nil {
		params = url.Values{}
		mergedURL.params = params
	}
	baseTimestamp := params.Get(constant.TimestampKey)
	baseHasTimestamp := baseTimestamp != ""

	func() {
		anotherUrl.paramsLock.RLock()
		defer anotherUrl.paramsLock.RUnlock()

		// Merge params from anotherUrl under one read lock.
		for key, value := range anotherUrl.params {
			if !valuesHasNonDefaultParam(params, key) {
				if len(value) > 0 {
					params[key] = make([]string, len(value))
					copy(params[key], value)
				}
			}
		}

		// remote timestamp
		if !baseHasTimestamp {
			params[constant.RemoteTimestampKey] = []string{baseTimestamp}
			params[constant.TimestampKey] = []string{anotherUrl.params.Get(constant.TimestampKey)}
		}

		// finally execute methodConfigMergeFcn
		mergedURL.Methods = make([]string, len(anotherUrl.Methods))
		for i, method := range anotherUrl.Methods {
			for _, paramKey := range []string{constant.LoadbalanceKey, constant.ClusterKey, constant.RetriesKey, constant.TimeoutKey} {
				if v := anotherUrl.params.Get(paramKey); len(v) > 0 {
					params[paramKey] = []string{v}
				}

				methodsKey := "methods." + method + "." + paramKey
				// if len(mergedURL.GetParam(methodsKey, "")) == 0 {
				if v := anotherUrl.params.Get(methodsKey); len(v) > 0 {
					params[methodsKey] = []string{v}
				}
				// }
				mergedURL.Methods[i] = method
			}
		}
	}()

	// merge attributes
	anotherUrl.RangeAttributes(func(attrK string, attrV any) bool {
		if _, ok := mergedURL.GetAttribute(attrK); !ok {
			mergedURL.SetAttribute(attrK, attrV)
		}
		return true
	})
	// In this way, we will raise some performance.
	mergedURL.ReplaceParams(params)
	return mergedURL
}

// CloneWithFilter - Clone the URL with parameter filtering
// excludeParams: the set of parameters to exclude from the cloned URL
// reserveParams: the set of parameters to retain in the cloned URL
func (c *URL) CloneWithFilter(excludeParams *gxset.HashSet, reserveParams []string) *URL {
	newURL := c.newURLForClone()
	newURL.params = c.copyFilteredParams(newURL.params, excludeParams, reserveParams)

	// Copy attributes
	c.RangeAttributes(
		func(key string, value any) bool {
			newURL.SetAttribute(key, value)
			return true
		},
	)

	// Copy SubURL if it exists
	if c.SubURL != nil {
		newURL.SubURL = c.SubURL.Clone()
	}

	return newURL
}

func (c *URL) newURLForClone() *URL {
	return &URL{
		Protocol:     c.Protocol,
		Location:     c.Location,
		Ip:           c.Ip,
		Port:         c.Port,
		PrimitiveURL: c.PrimitiveURL,
		primitiveTS:  c.primitiveTS,
		Path:         c.Path,
		Username:     c.Username,
		Password:     c.Password,
		Methods:      append(make([]string, 0), c.Methods...),
		attributes:   make(map[string]any),
		params:       url.Values{},
	}
}

func (c *URL) copyFilteredParams(params url.Values, excludeParams *gxset.HashSet, reserveParams []string) url.Values {
	c.paramsLock.RLock()
	defer c.paramsLock.RUnlock()

	if excludeParams == nil && len(reserveParams) == 0 {
		if len(c.params) > 8 {
			copiedParams := copyURLValues(c.params)
			if copiedParams != nil {
				return copiedParams
			}
		}
		for key, values := range c.params {
			copied := make([]string, len(values))
			copy(copied, values)
			params[key] = copied
		}
		return params
	}

	if capacity := paramsCapacityForClone(len(c.params), excludeParams, reserveParams); capacity > 8 {
		params = make(url.Values, capacity)
	}

	for key, values := range c.params {
		if shouldCopyParam(key, excludeParams, reserveParams) {
			copied := make([]string, len(values))
			copy(copied, values)
			params[key] = copied
		}
	}

	return params
}

func paramsCapacityForClone(paramsLen int, excludeParams *gxset.HashSet, reserveParams []string) int {
	if len(reserveParams) > 0 && len(reserveParams) < paramsLen {
		paramsLen = len(reserveParams)
	}
	if excludeParams != nil {
		paramsLen -= excludeParams.Size()
	}
	if paramsLen < 0 {
		return 0
	}
	return paramsLen
}

func shouldCopyParam(key string, excludeParams *gxset.HashSet, reserveParams []string) bool {
	if excludeParams != nil && excludeParams.Contains(key) {
		return false
	}
	return len(reserveParams) == 0 || slices.Contains(reserveParams, key)
}

// Clone will copy the URL
func (c *URL) Clone() *URL {
	return c.CloneWithFilter(nil, nil)
}

func (c *URL) RangeAttributes(f func(key string, value any) bool) {
	c.attributesLock.RLock()
	defer c.attributesLock.RUnlock()
	for k, v := range c.attributes {
		if !f(k, v) {
			break
		}
	}
}

func (c *URL) CloneExceptParams(excludeParams *gxset.HashSet) *URL {
	return c.CloneWithFilter(excludeParams, nil)
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

// CloneWithParams Copy URL based on the reserved parameter's keys.
func (c *URL) CloneWithParams(reserveParams []string) *URL {
	return c.CloneWithFilter(nil, reserveParams)
}

// keySet is a set of URL parameter keys that should be ignored during comparison.
type keySet map[string]struct{}

func newKeySet(keys []string) keySet {
	set := make(keySet, len(keys))
	for _, k := range keys {
		set[k] = struct{}{}
	}
	return set
}

func (s keySet) contains(key string) bool {
	_, ok := s[key]
	return ok
}

// reservedKeyList are the keys that ToMap materializes from scalar fields (or the
// "host:port" Location), where a non-empty field overrides the same-named param.
var reservedKeyList = []string{
	constant.ProtocolKey,
	constant.UsernameKey,
	constant.PasswordKey,
	constant.HostKey,
	constant.PortKey,
	constant.PathKey,
}

// reservedKeys is reservedKeyList as a set. equalParams skips these keys so that
// equalReservedKeys can compare them with ToMap's field-overrides-param precedence.
var reservedKeys = newKeySet(reservedKeyList)

// IsEquals compares if two URLs equals with each other. Excludes are all parameter keys which should ignored.
func IsEquals(left *URL, right *URL, excludes ...string) bool {
	if (left == nil && right != nil) || (right == nil && left != nil) {
		return false
	}
	if left.Ip != right.Ip || left.Port != right.Port {
		return false
	}

	excluded := newKeySet(excludes)
	return equalReservedKeys(left, right, excluded) &&
		equalParams(left, right, excluded)
}

// rawParam reports the first value of a param and whether the key is present in
// the params map (an empty-valued key still counts as present, matching ToMap).
func (c *URL) rawParam(key string) (string, bool) {
	c.paramsLock.RLock()
	defer c.paramsLock.RUnlock()

	if len(c.params) == 0 {
		return "", false
	}
	vs, ok := c.params[key]
	if !ok || len(vs) == 0 {
		return "", false
	}
	return vs[0], true
}

// effectiveReserved returns the flattened value of a reserved key and whether it
// is present, matching ToMap's precedence: a non-empty scalar field (or a
// non-empty Location for host/port) overrides the same-named param; otherwise the
// param, if any, is used.
func effectiveReserved(u *URL, key string) (string, bool) {
	switch key {
	case constant.ProtocolKey:
		if u.Protocol != "" {
			return u.Protocol, true
		}
	case constant.UsernameKey:
		if u.Username != "" {
			return u.Username, true
		}
	case constant.PasswordKey:
		if u.Password != "" {
			return u.Password, true
		}
	case constant.PathKey:
		if u.Path != "" {
			return u.Path, true
		}
	case constant.HostKey:
		if u.Location != "" {
			host, _ := parseLocation(u.Location)
			return host, true
		}
	case constant.PortKey:
		if u.Location != "" {
			_, port := parseLocation(u.Location)
			return port, true
		}
	}
	return u.rawParam(key)
}

// equalReservedKeys compares the reserved keys (protocol/username/password/host/
// port/path) of two URLs using ToMap precedence, skipping any excluded key.
func equalReservedKeys(left, right *URL, excluded keySet) bool {
	for _, key := range reservedKeyList {
		if excluded.contains(key) {
			continue
		}
		lv, lok := effectiveReserved(left, key)
		rv, rok := effectiveReserved(right, key)
		if lok != rok || lv != rv {
			return false
		}
	}
	return true
}

// equalParams compares the non-reserved params of two URLs, skipping excluded keys.
// Reserved keys are handled by equalScalars/equalLocation, so they are skipped here
// to preserve ToMap's "field overrides same-named param" precedence. It avoids
// materializing both maps by snapshotting only the left side and streaming the right.
func equalParams(left, right *URL, excluded keySet) bool {
	leftParams := make(map[string]string)
	left.RangeParams(func(key, value string) bool {
		if !reservedKeys.contains(key) && !excluded.contains(key) {
			leftParams[key] = value
		}
		return true
	})

	rightCount := 0
	matched := true
	right.RangeParams(func(key, value string) bool {
		if reservedKeys.contains(key) || excluded.contains(key) {
			return true
		}
		rightCount++
		if lv, ok := leftParams[key]; !ok || lv != value {
			matched = false
			return false
		}
		return true
	})

	return matched && rightCount == len(leftParams)
}

// parseLocation splits "host:port" into host and port. The port is the segment
// between the first and second ':' (equivalent to strings.Split(location, ":")[1]),
// preserving the historical behavior for locations that contain extra ':'.
func parseLocation(location string) (host, port string) {
	if location == "" {
		return "", "0"
	}
	idx := strings.IndexByte(location, ':')
	if idx < 0 {
		return location, "0"
	}
	host = location[:idx]
	rest := location[idx+1:]
	if j := strings.IndexByte(rest, ':'); j >= 0 {
		return host, rest[:j]
	}
	return host, rest
}

// URLSlice will be used to sort URL instance
// Instances will be order by URL.String()
type URLSlice []*URL

// Len returns the number of URLs in the slice.
func (s URLSlice) Len() int {
	return len(s)
}

// Less reports whether the URL at index i should sort before the URL at index j.
func (s URLSlice) Less(i, j int) bool {
	return s[i].String() < s[j].String()
}

// Swap swaps the URLs at indexes i and j.
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

// GetParamDuration get duration if param is invalid or missing default value will return 3s
func (c *URL) GetParamDuration(s string, d string) time.Duration {
	if t, err := time.ParseDuration(c.GetParam(s, d)); err == nil {
		return t
	}
	return 3 * time.Second
}

func GetSubscribeName(url *URL) string {
	var buffer bytes.Buffer

	buffer.Write([]byte(DubboNodes[PROVIDER]))
	appendParam(&buffer, url, constant.InterfaceKey)
	appendParam(&buffer, url, constant.VersionKey)
	appendParam(&buffer, url, constant.GroupKey)
	return buffer.String()
}

func appendParam(target *bytes.Buffer, url *URL, key string) {
	value := url.GetParam(key, "")
	target.Write([]byte(constant.NacosServiceNameSeparator))
	if strings.TrimSpace(value) != "" {
		target.Write([]byte(value))
	}
}
