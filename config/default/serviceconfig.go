package _default

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
)

func init() {
	extension.SetServiceConfig("default", GetServiceConfig)
	extension.SetURL("default", GetURL)
}

type DefaultServiceConfig struct {
	Protocol_ string `required:"true",default:"dubbo"  yaml:"protocol"  json:"protocol,omitempty"`
	Service_  string `required:"true"  yaml:"service"  json:"service,omitempty"`
	Group_    string `yaml:"group" json:"group,omitempty"`
	Version_  string `yaml:"version" json:"version,omitempty"`
}

func NewDefaultServiceConfig() DefaultServiceConfig {
	return DefaultServiceConfig{}
}

func (c *DefaultServiceConfig) Key() string {
	return fmt.Sprintf("%s@%s", c.Service_, c.Protocol_)
}

func (c *DefaultServiceConfig) String() string {
	return fmt.Sprintf("%s@%s-%s-%s", c.Service_, c.Protocol_, c.Group_, c.Version_)
}

func (c *DefaultServiceConfig) ServiceEqual(url config.URL) bool {
	if c.Protocol_ != url.Protocol() {
		return false
	}

	if c.Service_ != url.Query().Get("interface") {
		return false
	}

	if c.Group_ != url.Group() {
		return false
	}

	if c.Version_ != url.Version() {
		return false
	}

	return true
}

func (c *DefaultServiceConfig) Service() string {
	return c.Service_
}

func (c *DefaultServiceConfig) Protocol() string {
	return c.Protocol_
}

func (c *DefaultServiceConfig) Version() string {
	return c.Version_
}

func (c *DefaultServiceConfig) Group() string {
	return c.Group_
}
func (c *DefaultServiceConfig) SetProtocol(s string) {
	c.Protocol_ = s
}

func (c *DefaultServiceConfig) SetService(s string) {
	c.Service_ = s
}
func (c *DefaultServiceConfig) SetVersion(s string) {
	c.Version_ = s
}

func (c *DefaultServiceConfig) SetGroup(s string) {
	c.Group_ = s
}

func (c *DefaultServiceConfig) Export() {
	//todo:export
}

func GetServiceConfig() config.ServiceConfig {
	s := NewDefaultServiceConfig()
	return &s
}

/////////////////////////////////////
// url
/////////////////////////////////////

type DefaultServiceURL struct {
	Protocol_     string
	Location_     string // ip+port
	Path_         string // like  /com.ikurento.dubbo.UserProvider3
	Ip_           string
	Port_         string
	Timeout_      time.Duration
	Version_      string
	Group_        string
	Query_        url.Values
	Weight_       int32
	PrimitiveURL_ string
}

func NewDefaultServiceURL(urlString string) (config.URL, error) {
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

func (s DefaultServiceURL) String() string {
	return fmt.Sprintf(
		"DefaultServiceURL{Protocol:%s, Location:%s, Path:%s, Ip:%s, Port:%s, "+
			"Timeout:%s, Version:%s, Group:%s, Weight_:%d, Query:%+v}",
		s.Protocol_, s.Location_, s.Path_, s.Ip_, s.Port_,
		s.Timeout_, s.Version_, s.Group_, s.Weight_, s.Query_)
}

func (s *DefaultServiceURL) CheckMethod(method string) bool {
	var (
		methodArray []string
	)

	methodArray = strings.Split(s.Query_.Get("methods"), ",")
	for _, m := range methodArray {
		if m == method {
			return true
		}
	}

	return false
}

func (s *DefaultServiceURL) PrimitiveURL() string {
	return s.PrimitiveURL_
}

func (s *DefaultServiceURL) Timeout() time.Duration {
	return s.Timeout_
}
func (s *DefaultServiceURL) Location() string {
	return s.Location_
}

func (s *DefaultServiceURL) Query() url.Values {
	return s.Query_
}

func (s *DefaultServiceURL) Group() string {
	return s.Group_
}

func (s *DefaultServiceURL) Protocol() string {
	return s.Protocol_
}

func (s *DefaultServiceURL) Version() string {
	return s.Version_
}

func (s *DefaultServiceURL) Ip() string {
	return s.Ip_
}

func (s *DefaultServiceURL) Port() string {
	return s.Port_
}

func (s *DefaultServiceURL) Path() string {
	return s.Path_
}

func GetURL(urlString string) config.URL {
	url, err := NewDefaultServiceURL(urlString)
	if err != nil {
		log.Error(jerrors.Trace(err))
	}
	return url
}
