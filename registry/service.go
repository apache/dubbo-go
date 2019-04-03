package registry

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

//////////////////////////////////////////////
// service config
//////////////////////////////////////////////

type ServiceConfig interface {
	Key() string
	String() string
	ServiceEqual(url *ServiceURL) bool
}

type DefaultServiceConfig struct {
	Protocol string `required:"true",default:"dubbo"  yaml:"protocol"  json:"protocol,omitempty"`
	Service  string `required:"true"  yaml:"service"  json:"service,omitempty"`
	Group    string `yaml:"group" json:"group,omitempty"`
	Version  string `yaml:"version" json:"version,omitempty"`
}

func (c DefaultServiceConfig) Key() string {
	return fmt.Sprintf("%s@%s", c.Service, c.Protocol)
}

func (c DefaultServiceConfig) String() string {
	return fmt.Sprintf("%s@%s-%s-%s", c.Service, c.Protocol, c.Group, c.Version)
}

func (c DefaultServiceConfig) ServiceEqual(url *ServiceURL) bool {
	if c.Protocol != url.Protocol {
		return false
	}

	if c.Service != url.Query.Get("interface") {
		return false
	}

	if c.Group != url.Group {
		return false
	}

	if c.Version != url.Version {
		return false
	}

	return true
}

type ProviderServiceConfig struct {
	DefaultServiceConfig
	Path    string `yaml:"path" json:"path,omitempty"`
	Methods string `yaml:"methods" json:"methods,omitempty"`
}

//////////////////////////////////////////
// service url
//////////////////////////////////////////

type ServiceURL struct {
	Protocol     string
	Location     string // ip+port
	Path         string // like  /com.ikurento.dubbo.UserProvider3
	Ip           string
	Port         string
	Timeout      time.Duration
	Version      string
	Group        string
	Query        url.Values
	Weight       int32
	PrimitiveURL string
}

func NewServiceURL(urlString string) (*ServiceURL, error) {
	var (
		err          error
		rawUrlString string
		serviceUrl   *url.URL
		s            = &ServiceURL{}
	)

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

func (s ServiceURL) String() string {
	return fmt.Sprintf(
		"ServiceURL{Protocol:%s, Location:%s, Path:%s, Ip:%s, Port:%s, "+
			"Timeout:%s, Version:%s, Group:%s, Weight:%d, Query:%+v}",
		s.Protocol, s.Location, s.Path, s.Ip, s.Port,
		s.Timeout, s.Version, s.Group, s.Weight, s.Query)
}

func (s *ServiceURL) ServiceConfig() DefaultServiceConfig {
	interfaceName := s.Query.Get("interface")
	return DefaultServiceConfig{
		Protocol: s.Protocol,
		Service:  interfaceName,
		Group:    s.Group,
		Version:  s.Version,
	}
}

func (s *ServiceURL) CheckMethod(method string) bool {
	var (
		methodArray []string
	)

	methodArray = strings.Split(s.Query.Get("methods"), ",")
	for _, m := range methodArray {
		if m == method {
			return true
		}
	}

	return false
}
