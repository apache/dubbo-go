package registry

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

import (
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
	Version      string
	Group        string
	Query        url.Values
	Weight       int32
	PrimitiveURL string
}

func (s ServiceURL) String() string {
	return fmt.Sprintf(
		"ServiceURL{Protocol:%s, Location:%s, Path:%s, Ip:%s, Port:%s, "+
			"Version:%s, Group:%s, Weight:%d, Query:%+v}",
		s.Protocol, s.Location, s.Path, s.Ip, s.Port,
		s.Version, s.Group, s.Weight, s.Query)
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

type serviceArray struct {
	arr   []*ServiceURL
	birth time.Time
	idx   int64
}

func newServiceArray(arr []*ServiceURL) *serviceArray {
	return &serviceArray{
		arr:   arr,
		birth: time.Now(),
	}
}

func (s *serviceArray) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("birth:%s, idx:%d, arr len:%d, arr:{", s.birth, s.idx, len(s.arr)))
	for i := range s.arr {
		builder.WriteString(fmt.Sprintf("%d:%s, ", i, s.arr[i]))
	}
	builder.WriteString("}")

	return builder.String()
}

func (s *serviceArray) Select(ID int64, mode Mode, ttl time.Duration) (*ServiceURL, error) {
	arrSize := len(s.arr)
	if arrSize == 0 {
		return nil, ErrServiceArrayEmpty
	}

	if ttl != 0 && time.Since(s.birth) > ttl {
		return nil, ErrServiceArrayTimeout
	}

	idx := atomic.AddInt64(&s.idx, 1)
	switch mode {
	case SM_RoundRobin:
		idx = (ID + idx) % int64(arrSize)
	default: // random
		idx = ((int64)(rand.Int()) + ID) % int64(arrSize)
	}

	return s.arr[idx], nil
}

func (s *serviceArray) Add(service *ServiceURL, ttl time.Duration) {
	s.arr = append(s.arr, service)
	s.birth = time.Now().Add(ttl)
}

func (s *serviceArray) Del(service *ServiceURL, ttl time.Duration) {
	for i, svc := range s.arr {
		if svc.PrimitiveURL == service.PrimitiveURL {
			s.arr = append(s.arr[:i], s.arr[i+1:]...)
			s.birth = time.Now().Add(ttl)
			break
		}
	}
}
