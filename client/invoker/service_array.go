package invoker

import (
	"fmt"
	"strings"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/service"
)

//////////////////////////////////////////
// service array
// should be returned by registry ,will be used by client & waiting to selector
//////////////////////////////////////////

var (
	ErrServiceArrayEmpty   = jerrors.New("serviceArray empty")
	ErrServiceArrayTimeout = jerrors.New("serviceArray timeout")
)

type ServiceArray struct {
	arr   []*service.ServiceURL
	birth time.Time
	idx   int64
}

func newServiceArray(arr []*service.ServiceURL) *ServiceArray {
	return &ServiceArray{
		arr:   arr,
		birth: time.Now(),
	}
}

func (s *ServiceArray) GetIdx() *int64 {
	return &s.idx
}
func (s *ServiceArray) GetSize() int {
	return len(s.arr)
}
func (s *ServiceArray) GetService(i int) *service.ServiceURL {
	return s.arr[i]
}

func (s *ServiceArray) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("birth:%s, idx:%d, arr len:%d, arr:{", s.birth, s.idx, len(s.arr)))
	for i := range s.arr {
		builder.WriteString(fmt.Sprintf("%d:%s, ", i, s.arr[i]))
	}
	builder.WriteString("}")

	return builder.String()
}

func (s *ServiceArray) add(service *service.ServiceURL, ttl time.Duration) {
	s.arr = append(s.arr, service)
	s.birth = time.Now().Add(ttl)
}

func (s *ServiceArray) del(service *service.ServiceURL, ttl time.Duration) {
	for i, svc := range s.arr {
		if svc.PrimitiveURL == service.PrimitiveURL {
			s.arr = append(s.arr[:i], s.arr[i+1:]...)
			s.birth = time.Now().Add(ttl)
			break
		}
	}
}
