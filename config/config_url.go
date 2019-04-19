package config

import (
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

type baseUrl struct {
	Protocol     string `required:"true"  yaml:"protocol"  json:"protocol,omitempty"`
	Location     string // ip+port
	Ip           string
	Port         string
	Timeout      time.Duration
	Query        url.Values
	PrimitiveURL string
}

type ConfigURL struct {
	baseUrl
	Service string

	Path string `yaml:"path" json:"path,omitempty"` // like  /com.ikurento.dubbo.UserProvider3

	Version string `yaml:"version" json:"version,omitempty"`
	Group   string `yaml:"group" json:"group,omitempty"`

	Weight   int32
	Methods  string `yaml:"methods" json:"methods,omitempty"`
	Username string
	Password string
}

func NewConfigURL(urlString string) (*ConfigURL, error) {

	var (
		err          error
		rawUrlString string
		serviceUrl   *url.URL
		s            = &ConfigURL{}
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

	s.Query, err = url.ParseQuery(serviceUrl.RawQuery)
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
	s.Group = s.Query.Get("group")
	s.Version = s.Query.Get("version")
	timeoutStr := s.Query.Get("timeout")
	if len(timeoutStr) == 0 {
		timeoutStr = s.Query.Get("default.timeout")
	}
	if len(timeoutStr) != 0 {
		timeout, err := strconv.Atoi(timeoutStr)
		if err == nil && timeout != 0 {
			s.Timeout = time.Duration(timeout * 1e6) // timeout unit is millisecond
		}
	}

	return s, nil
}

func (c *ConfigURL) Key() string {
	return fmt.Sprintf("%s@%s-%s-%s-%s-%s", c.Service, c.Protocol, c.Group, c.Location, c.Version, c.Methods)
}

func (c *ConfigURL) ConfigURLEqual(url ConfigURL) bool {

	if c.Key() != url.Key() {
		return false
	}
	return true
}
func (s ConfigURL) String() string {
	return fmt.Sprintf(
		"DefaultServiceURL{Protocol:%s, Location:%s, Path:%s, Ip:%s, Port:%s, "+
			"Timeout:%s, Version:%s, Group:%s, Weight_:%d, Query:%+v}",
		s.Protocol, s.Location, s.Path, s.Ip, s.Port,
		s.Timeout, s.Version, s.Group, s.Weight, s.Query)
}
