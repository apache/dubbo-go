package directory

import (
	"fmt"

	"strings"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

import "github.com/dubbo/dubbo-go/config"

//////////////////////////////////////////
// registry array
// should be returned by registry ,will be used by client & waiting to selector
//////////////////////////////////////////

var (
	ErrServiceArrayEmpty   = jerrors.New("registryArray empty")
	ErrServiceArrayTimeout = jerrors.New("registryArray timeout")
)

type ServiceArray struct {
	arr   []config.URL
	birth time.Time
	idx   int64
}

func NewServiceArray(arr []config.URL) *ServiceArray {
	return &ServiceArray{
		arr:   arr,
		birth: time.Now(),
	}
}

func (s *ServiceArray) GetIdx() *int64 {
	return &s.idx
}

func (s *ServiceArray) GetSize() int64 {
	return int64(len(s.arr))
}

func (s *ServiceArray) GetService(i int64) config.URL {
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

func (s *ServiceArray) add(url config.URL, ttl time.Duration) {
	s.arr = append(s.arr, url)
	s.birth = time.Now().Add(ttl)
}

func (s *ServiceArray) del(url config.URL, ttl time.Duration) {
	for i, svc := range s.arr {
		if svc.PrimitiveURL == url.PrimitiveURL {
			s.arr = append(s.arr[:i], s.arr[i+1:]...)
			s.birth = time.Now().Add(ttl)
			break
		}
	}
}
