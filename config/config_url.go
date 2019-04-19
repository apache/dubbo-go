package config

import (
	"fmt"
	"github.com/dubbo/dubbo-go/common/extension"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

func init() {
	extension.SetDefaultURLExtension(NewDefaultConfigURL)
}

// load url config
type ConfigURL interface {
	Key() string
	String() string
	ConfigURLEqual(url ConfigURL) bool
	PrimitiveURL() string
	Query() url.Values
	Location() string
	Timeout() time.Duration
	Group() string
	Protocol() string
	Version() string
	Ip() string
	Port() string
	Path() string
	Service() string
	Methods() string

	SetProtocol(string)
	SetService(string)
	SetVersion(string)
	SetGroup(string)
	SetMethods(string)
	SetPath(string)
}

type DefaultConfigURL struct {
	Service_      string
	Protocol_     string `required:"true",default:"dubbo"  yaml:"protocol"  json:"protocol,omitempty"`
	Location_     string // ip+port
	Path_         string `yaml:"path" json:"path,omitempty"` // like  /com.ikurento.dubbo.UserProvider3
	Ip_           string
	Port_         string
	Timeout_      time.Duration
	Version_      string `yaml:"version" json:"version,omitempty"`
	Group_        string `yaml:"group" json:"group,omitempty"`
	Query_        url.Values
	Weight_       int32
	PrimitiveURL_ string
	Methods_      string `yaml:"methods" json:"methods,omitempty"`
}

func NewDefaultConfigURL(urlString string) (ConfigURL, error) {
	var (
		err          error
		rawUrlString string
		serviceUrl   *url.URL
		s            = &DefaultConfigURL{}
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

	s.Query_, err = url.ParseQuery(serviceUrl.RawQuery)
	if err != nil {
		return nil, jerrors.Errorf("url.ParseQuery(raw url string{%s}),  error{%v}", serviceUrl.RawQuery, err)
	}

	s.PrimitiveURL_ = urlString
	s.Protocol_ = serviceUrl.Scheme
	s.Location_ = serviceUrl.Host
	s.Path_ = serviceUrl.Path
	if strings.Contains(s.Location_, ":") {
		s.Ip_, s.Port_, err = net.SplitHostPort(s.Location_)
		if err != nil {
			return nil, jerrors.Errorf("net.SplitHostPort(Url.Host{%s}), error{%v}", s.Location_, err)
		}
	}
	s.Group_ = s.Query_.Get("group")
	s.Version_ = s.Query_.Get("version")
	timeoutStr := s.Query_.Get("timeout")
	if len(timeoutStr) == 0 {
		timeoutStr = s.Query_.Get("default.timeout")
	}
	if len(timeoutStr) != 0 {
		timeout, err := strconv.Atoi(timeoutStr)
		if err == nil && timeout != 0 {
			s.Timeout_ = time.Duration(timeout * 1e6) // timeout unit is millisecond
		}
	}

	return s, nil
}

func (c *DefaultConfigURL) Key() string {
	return fmt.Sprintf("%s@%s-%s-%s-%s-%s", c.Service_, c.Protocol_, c.Group_, c.Location_, c.Version_, c.Methods_)
}

func (c *DefaultConfigURL) ConfigURLEqual(url ConfigURL) bool {
	if c.Key() != url.Key() {
		return false
	}
	return true
}

func (c DefaultConfigURL) String() string {
	return fmt.Sprintf(
		"DefaultServiceURL{Protocol:%s, Location:%s, Path:%s, Ip:%s, Port:%s, "+
			"Timeout:%s, Version:%s, Group:%s, Weight_:%d, Query:%+v}",
		c.Protocol_, c.Location_, c.Path_, c.Ip_, c.Port_,
		c.Timeout_, c.Version_, c.Group_, c.Weight_, c.Query_)
}

func (c *DefaultConfigURL) Service() string {
	return c.Service_
}
func (c *DefaultConfigURL) PrimitiveURL() string {
	return c.PrimitiveURL_
}

func (c *DefaultConfigURL) Timeout() time.Duration {
	return c.Timeout_
}
func (c *DefaultConfigURL) Location() string {
	return c.Location_
}

func (c *DefaultConfigURL) Query() url.Values {
	return c.Query_
}

func (c *DefaultConfigURL) Group() string {
	return c.Group_
}

func (c *DefaultConfigURL) Protocol() string {
	return c.Protocol_
}

func (c *DefaultConfigURL) Version() string {
	return c.Version_
}

func (c *DefaultConfigURL) Ip() string {
	return c.Ip_
}

func (c *DefaultConfigURL) Port() string {
	return c.Port_
}

func (c *DefaultConfigURL) Path() string {
	return c.Path_
}

func (c *DefaultConfigURL) Methods() string {
	return c.Methods_
}

func (c *DefaultConfigURL) SetProtocol(p string) {
	c.Protocol_ = p
}

func (c *DefaultConfigURL) SetService(s string) {
	c.Service_ = s
}

func (c *DefaultConfigURL) SetVersion(v string) {
	c.Version_ = v
}

func (c *DefaultConfigURL) SetGroup(g string) {
	c.Group_ = g
}

func (c *DefaultConfigURL) SetMethods(m string) {
	c.Methods_ = m
}

func (c *DefaultConfigURL) SetPath(p string) {
	c.Path_ = p
}
