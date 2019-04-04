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
	ServiceEqual(url *DefaultServiceURL) bool
	//your service config implements must contain properties below
	Service() string
	Protocol() string
	Version() string
	Group() string
	SetProtocol(string)
	SetService(string)
	SetVersion(string)
	SetGroup(string)
}

type ProviderServiceConfig interface {
	//your service config implements must contain properties below
	ServiceConfig
	Methods() string
	Path() string
	SetMethods(string)
	SetPath(string)
}

type DefaultServiceConfig struct {
	DProtocol string `required:"true",default:"dubbo"  yaml:"protocol"  json:"protocol,omitempty"`
	DService  string `required:"true"  yaml:"service"  json:"service,omitempty"`
	DGroup    string `yaml:"group" json:"group,omitempty"`
	DVersion  string `yaml:"version" json:"version,omitempty"`
}

func NewDefaultServiceConfig() ServiceConfig {
	return &DefaultServiceConfig{}
}

func (c *DefaultServiceConfig) Key() string {
	return fmt.Sprintf("%s@%s", c.DService, c.DProtocol)
}

func (c *DefaultServiceConfig) String() string {
	return fmt.Sprintf("%s@%s-%s-%s", c.DService, c.DProtocol, c.DGroup, c.DVersion)
}

func (c *DefaultServiceConfig) ServiceEqual(url *DefaultServiceURL) bool {
	if c.DProtocol != url.Protocol {
		return false
	}

	if c.DService != url.Query.Get("interface") {
		return false
	}

	if c.DGroup != url.Group {
		return false
	}

	if c.DVersion != url.Version {
		return false
	}

	return true
}

func (c *DefaultServiceConfig) Service() string {
	return c.DService
}

func (c *DefaultServiceConfig) Protocol() string {
	return c.DProtocol
}

func (c *DefaultServiceConfig) Version() string {
	return c.DVersion
}

func (c *DefaultServiceConfig) Group() string {
	return c.DGroup
}
func (c *DefaultServiceConfig) SetProtocol(s string) {
	c.DProtocol = s
}

func (c *DefaultServiceConfig) SetService(s string) {
	c.DService = s
}
func (c *DefaultServiceConfig) SetVersion(s string) {
	c.DVersion = s
}

func (c *DefaultServiceConfig) SetGroup(s string) {
	c.DGroup = s
}

type DefaultProviderServiceConfig struct {
	*DefaultServiceConfig
	DPath    string `yaml:"path" json:"path,omitempty"`
	DMethods string `yaml:"methods" json:"methods,omitempty"`
}

func NewDefaultProviderServiceConfig() ProviderServiceConfig {
	return &DefaultProviderServiceConfig{
		DefaultServiceConfig: NewDefaultServiceConfig().(*DefaultServiceConfig),
	}
}

func (c *DefaultProviderServiceConfig) Methods() string {
	return c.DMethods
}

func (c *DefaultProviderServiceConfig) Path() string {
	return c.DPath
}

func (c *DefaultProviderServiceConfig) SetMethods(s string) {
	c.DMethods = s
}

func (c *DefaultProviderServiceConfig) SetPath(s string) {
	c.DPath = s
}

//////////////////////////////////////////
// service url
//////////////////////////////////////////

type ServiceURL interface {
	ServiceConfig() ServiceConfig
	CheckMethod(string) bool
}

type DefaultServiceURL struct {
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

func NewDefaultServiceURL(urlString string) (*DefaultServiceURL, error) {
	var (
		err          error
		rawUrlString string
		serviceUrl   *url.URL
		s            = &DefaultServiceURL{}
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

func (s DefaultServiceURL) String() string {
	return fmt.Sprintf(
		"DefaultServiceURL{Protocol:%s, Location:%s, Path:%s, Ip:%s, Port:%s, "+
			"Timeout:%s, Version:%s, Group:%s, Weight:%d, Query:%+v}",
		s.Protocol, s.Location, s.Path, s.Ip, s.Port,
		s.Timeout, s.Version, s.Group, s.Weight, s.Query)
}

func (s *DefaultServiceURL) ServiceConfig() ServiceConfig {
	interfaceName := s.Query.Get("interface")
	return &DefaultServiceConfig{
		DProtocol: s.Protocol,
		DService:  interfaceName,
		DGroup:    s.Group,
		DVersion:  s.Version,
	}
}

func (s *DefaultServiceURL) CheckMethod(method string) bool {
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
