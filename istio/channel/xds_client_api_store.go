package channel

import (
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"sync"
)

type ResponseInfo struct {
	ResponseNonce string
	VersionInfo   string
	ResourceNames []string
}

type ApiStore struct {
	mutex     sync.Mutex
	responses map[string]*ResponseInfo
	rdsMutex  sync.Mutex
}

const (
	EnvoyListener = resource.ListenerType
	EnvoyCluster  = resource.ClusterType
	EnvoyEndpoint = resource.EndpointType
	EnvoyRoute    = resource.RouteType
)

func NewApiStore() *ApiStore {
	return &ApiStore{
		responses: map[string]*ResponseInfo{},
	}
}

func (s *ApiStore) Store(typeUrl string, info *ResponseInfo) {
	s.mutex.Lock()
	s.responses[typeUrl] = info
	s.mutex.Unlock()
}

func (s *ApiStore) SetResourceNames(typeUrl string, ResourceNames []string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	info, ok := s.responses[typeUrl]
	if ok {
		info.ResourceNames = ResourceNames
		s.responses[typeUrl] = info
		return
	}
	s.responses[typeUrl] = &ResponseInfo{
		ResourceNames: ResourceNames,
	}
}

func (s *ApiStore) Find(typeUrl string) *ResponseInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	info, ok := s.responses[typeUrl]
	if !ok {
		return &ResponseInfo{
			ResourceNames: []string{},
		}
	}
	return info
}
