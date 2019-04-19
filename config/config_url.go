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

func init(){
	extension.SetDefaultURLExtension(NewDefaultServiceURL)
}



//////////////////////////////////////////
// service url
//////////////////////////////////////////

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
	Service()string
	Methods()string
}

type DefaultServiceURL struct {
	Service_      string
	Protocol_     string `required:"true",default:"dubbo"  yaml:"protocol"  json:"protocol,omitempty"`
	Location_     string // ip+port
	Path_         string `yaml:"path" json:"path,omitempty"`// like  /com.ikurento.dubbo.UserProvider3
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

func NewDefaultServiceURL(urlString string) (ConfigURL, error) {
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



func (c *DefaultServiceURL) Key() string {
	return fmt.Sprintf("%s@%s-%s-%s-%s-%s", c.Service_, c.Protocol_,c.Group_,c.Location_,c.Version_,c.Methods_)
}


func (c *DefaultServiceURL) ConfigURLEqual(url ConfigURL) bool {
	if c.Key() != url.Key() {
		return false
	}
	return true
}
func (s DefaultServiceURL) String() string {
	return fmt.Sprintf(
		"DefaultServiceURL{Protocol:%s, Location:%s, Path:%s, Ip:%s, Port:%s, "+
			"Timeout:%s, Version:%s, Group:%s, Weight_:%d, Query:%+v}",
		s.Protocol_, s.Location_, s.Path_, s.Ip_, s.Port_,
		s.Timeout_, s.Version_, s.Group_, s.Weight_, s.Query_)
}

func (s *DefaultServiceURL) Service() string {
	return s.Service_
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

func (c *DefaultServiceURL) Methods() string {
	return c.Methods_
}
