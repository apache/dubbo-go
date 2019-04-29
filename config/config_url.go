package config

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

type IURL interface {
	Key() string
	URLEqual(IURL) bool
	Context() context.Context
	GetParam(string, string) string
	GetParamInt(string, int64) int64
	String() string
}

type baseUrl struct {
	Protocol     string `required:"true"  yaml:"protocol"  json:"protocol,omitempty"`
	Location     string // ip+port
	Ip           string
	Port         string
	Timeout      time.Duration
	Params       url.Values
	PrimitiveURL string
	ctx          context.Context
}

type URL struct {
	baseUrl
	Service string

	Path string `yaml:"path" json:"path,omitempty"` // like  /com.ikurento.dubbo.UserProvider3

	//Weight  int32
	Version string `yaml:"version" json:"version,omitempty"`
	Group   string `yaml:"group" json:"group,omitempty"`

	Username string
	Password string
	Methods  []string
}

func NewURL(ctx context.Context, urlString string) (*URL, error) {

	var (
		err          error
		rawUrlString string
		serviceUrl   *url.URL
		s            = &URL{baseUrl: baseUrl{ctx: ctx}}
	)

	// new a null instance
	if urlString == "" {
		return s, nil
	}

	rawUrlString, err = url.QueryUnescape(urlString)
	if err != nil {
		return nil, jerrors.Errorf("url.QueryUnescape(%s),  error{%v}", urlString, err)
	}

	serviceUrl, err = url.Parse(rawUrlString)
	if err != nil {
		return nil, jerrors.Errorf("url.Parse(url string{%s}),  error{%v}", rawUrlString, err)
	}

	s.Params, err = url.ParseQuery(serviceUrl.RawQuery)
	if err != nil {
		return nil, jerrors.Errorf("url.ParseQuery(raw url string{%s}),  error{%v}", serviceUrl.RawQuery, err)
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
			return nil, jerrors.Errorf("net.SplitHostPort(Url.Host{%s}), error{%v}", s.Location, err)
		}
	}
	s.Group = s.Params.Get("group")
	s.Version = s.Params.Get("version")

	timeoutStr := s.Params.Get("timeout")
	if len(timeoutStr) == 0 {
		timeoutStr = s.Params.Get("default.timeout")
	}
	if len(timeoutStr) != 0 {
		timeout, err := strconv.Atoi(timeoutStr)
		if err == nil && timeout != 0 {
			s.Timeout = time.Duration(timeout * 1e6) // timeout unit is millisecond
		}
	}

	return s, nil
}

func (c *URL) Key() string {
	return fmt.Sprintf("%s@%s-%s-%s-%s", c.Service, c.Protocol, c.Group, c.Location, c.Version)
}

func (c *URL) URLEqual(url IURL) bool {

	if c.Key() != url.Key() {
		return false
	}
	return true
}
func (c URL) String() string {
	return fmt.Sprintf(
		"DefaultServiceURL{Protocol:%s, Location:%s, Path:%s, Ip:%s, Port:%s, "+
			"Timeout:%s, Version:%s, Group:%s,  Params:%+v}",
		c.Protocol, c.Location, c.Path, c.Ip, c.Port,
		c.Timeout, c.Version, c.Group, c.Params)
}

func (c *URL) ToFullString() string {
	buildString := fmt.Sprintf(
		"%s://%s:%s@%s:%s/%s?verison=%s&group=%s",
		c.Protocol, c.Username, c.Password, c.Ip, c.Port, c.Path, c.Version, c.Group)
	for k, v := range c.Params {
		buildString += "&" + k + "=" + v[0]
	}
	return buildString
}

func (c *URL) Context() context.Context {
	return c.ctx
}

func (c *URL) GetParam(s string, d string) string {
	var r string
	if r = c.Params.Get(s); r == "" {
		r = d
	}
	return r
}

func (c *URL) GetParamInt(s string, d int64) int64 {
	var r int
	var err error
	if r, err = strconv.Atoi(c.Params.Get(s)); r == 0 || err != nil {
		return d
	}
	return int64(r)
}

func (c *URL) GetMethodParamInt(method string, key string, d int64) int64 {
	var r int
	var err error
	if r, err = strconv.Atoi(c.Params.Get("methods." + method + "." + key)); r == 0 || err != nil {
		return d
	}
	return int64(r)
}

func (c *URL) GetMethodParam(method string, key string, d string) string {
	var r string
	if r = c.Params.Get(c.Params.Get("methods." + method + "." + key)); r == "" {
		r = d
	}
	return r
}
