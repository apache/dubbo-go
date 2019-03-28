package service

import (
	"fmt"
	jerrors "github.com/juju/errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

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

func (s ServiceURL) String() string {
	return fmt.Sprintf(
		"ServiceURL{Protocol:%s, Location:%s, Path:%s, Ip:%s, Port:%s, "+
			"Timeout:%s, Version:%s, Group:%s, Weight:%d, Query:%+v}",
		s.Protocol, s.Location, s.Path, s.Ip, s.Port,
		s.Timeout, s.Version, s.Group, s.Weight, s.Query)
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

func (s *ServiceURL) ServiceConfig() ServiceConfig {
	interfaceName := s.Query.Get("interface")
	return ServiceConfig{
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

