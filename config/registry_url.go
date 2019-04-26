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

/////////////////////////////////
// dubbo role type
/////////////////////////////////

const (
	CONSUMER = iota
	CONFIGURATOR
	ROUTER
	PROVIDER
)

var (
	DubboNodes = [...]string{"consumers", "configurators", "routers", "providers"}
	DubboRole  = [...]string{"consumer", "", "", "provider"}
)

type DubboType int

func (t DubboType) String() string {
	return DubboNodes[t]
}

func (t DubboType) Role() string {
	return DubboRole[t]
}

type RegistryURL struct {
	baseUrl
	//store for service waiting for register
	URL URL
	//both for registry & service & reference
	Version string `yaml:"version" json:"version,omitempty"`
	Group   string `yaml:"group" json:"group,omitempty"`
	//for registry
	Username     string    `yaml:"username" json:"username,omitempty"`
	Password     string    `yaml:"password" json:"password,omitempty"`
	DubboType    DubboType `yaml:"-"`
	Organization string    `yaml:"organization" json:"organization,omitempty"`
	Name         string    `yaml:"name" json:"name,omitempty"`
	Module       string    `yaml:"module" json:"module,omitempty"`
	Owner        string    `yaml:"owner" json:"owner,omitempty"`
	Environment  string    `yaml:"environment" json:"environment,omitempty"`
	Address      string    `yaml:"address" json:"address,omitempty"`
}

func NewRegistryURL(context context.Context, urlString string) (*RegistryURL, error) {

	var (
		err          error
		rawUrlString string
		serviceUrl   *url.URL
		s            = &RegistryURL{}
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
	s.ctx = context
	return s, nil
}

func (c *RegistryURL) Key() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", c.Protocol, c.Group, c.Location, c.Version, c.DubboType)
}

func (c *RegistryURL) URLEqual(url IURL) bool {

	if c.Key() != url.Key() {
		return false
	}
	return true
}

func (c *RegistryURL) Context() context.Context {
	return c.ctx
}
