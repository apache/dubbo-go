package registry

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	jerrors "github.com/juju/errors"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

//////////////////////////////////////////
// load balancer mode
//////////////////////////////////////////

// Mode defines the algorithm of selecting a provider from cluster
type Mode int

const (
	SM_BEGIN Mode = iota
	SM_Random
	SM_RoundRobin
	SM_END
)

var modeStrings = [...]string{
	"Begin",
	"Random",
	"RoundRobin",
	"End",
}

func (s Mode) String() string {
	if SM_BEGIN < s && s < SM_END {
		return modeStrings[s]
	}

	return ""
}

//////////////////////////////////////////
// service url event type
//////////////////////////////////////////

type ServiceURLEventType int

const (
	ServiceURLAdd = iota
	ServiceURLDel
)

var serviceURLEventTypeStrings = [...]string{
	"add service url",
	"delete service url",
}

func (t ServiceURLEventType) String() string {
	return serviceURLEventTypeStrings[t]
}

//////////////////////////////////////////
// service url event
//////////////////////////////////////////

type ServiceURLEvent struct {
	Action  ServiceURLEventType
	Service *ServiceURL
}

func (e ServiceURLEvent) String() string {
	return fmt.Sprintf("ServiceURLEvent{Action{%s}, Service{%s}}", e.Action.String(), e.Service)
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

//////////////////////////////////////////
// service array
//////////////////////////////////////////

var (
	ErrServiceArrayEmpty   = jerrors.New("serviceArray empty")
	ErrServiceArrayTimeout = jerrors.New("serviceArray timeout")
)

type ServiceArray struct {
	Arr   []*ServiceURL
	Birth time.Time
	Idx   int64
}

func NewServiceArray(arr []*ServiceURL) *ServiceArray {
	return &ServiceArray{
		Arr:   arr,
		Birth: time.Now(),
	}
}

func (s *ServiceArray) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("birth:%s, idx:%d, arr len:%d, arr:{", s.Birth, s.Idx, len(s.Arr)))
	for i := range s.Arr {
		builder.WriteString(fmt.Sprintf("%d:%s, ", i, s.Arr[i]))
	}
	builder.WriteString("}")

	return builder.String()
}

func (s *ServiceArray) Select(ID int64, mode Mode, ttl time.Duration) (*ServiceURL, error) {
	arrSize := len(s.Arr)
	if arrSize == 0 {
		return nil, ErrServiceArrayEmpty
	}

	if ttl != 0 && time.Since(s.Birth) > ttl {
		return nil, ErrServiceArrayTimeout
	}

	idx := atomic.AddInt64(&s.Idx, 1)
	switch mode {
	case SM_RoundRobin:
		idx = (ID + idx) % int64(arrSize)
	default: // random
		idx = ((int64)(rand.Int()) + ID) % int64(arrSize)
	}

	return s.Arr[idx], nil
}

func (s *ServiceArray) Add(service *ServiceURL, ttl time.Duration) {
	s.Arr = append(s.Arr, service)
	s.Birth = time.Now().Add(ttl)
}

func (s *ServiceArray) Del(service *ServiceURL, ttl time.Duration) {
	for i, svc := range s.Arr {
		if svc.PrimitiveURL == service.PrimitiveURL {
			s.Arr = append(s.Arr[:i], s.Arr[i+1:]...)
			s.Birth = time.Now().Add(ttl)
			break
		}
	}
}
