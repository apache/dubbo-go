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

	Weight  int32
	Methods string `yaml:"methods" json:"methods,omitempty"`

	Version string `yaml:"version" json:"version,omitempty"`
	Group   string `yaml:"group" json:"group,omitempty"`

	Username string
	Password string

	//reference only
	Cluster string
}

type method struct {
	Name    string
	Retries int
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
	return fmt.Sprintf("%s@%s-%s-%s-%s-%s", c.Service, c.Protocol, c.Group, c.Location, c.Version, c.Methods)
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
			"Timeout:%s, Version:%s, Group:%s, Weight_:%d, Params:%+v}",
		c.Protocol, c.Location, c.Path, c.Ip, c.Port,
		c.Timeout, c.Version, c.Group, c.Weight, c.Params)
}

func (c *URL) ToFullString() string {
	return fmt.Sprintf(
		"%s://%s:%s@%s:%s/%s?%s&%s&%s&%s",
		c.Protocol, c.Password, c.Username, c.Ip, c.Port, c.Path, c.Methods, c.Version, c.Group, c.Params)
}

func (c *URL) Context() context.Context {
	return c.ctx
}
