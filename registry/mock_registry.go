package registry

import (
	"go.uber.org/atomic"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common"
)

type MockRegistry struct {
	listener  *listener
	destroyed *atomic.Bool
}

func NewMockRegistry(url *common.URL) (Registry, error) {
	registry := &MockRegistry{
		destroyed: atomic.NewBool(false),
	}
	listener := &listener{count: 0, registry: registry, listenChan: make(chan *ServiceEvent)}
	registry.listener = listener
	return registry, nil
}
func (*MockRegistry) Register(url common.URL) error {
	return nil
}

func (r *MockRegistry) Destroy() {
	if r.destroyed.CAS(false, true) {
	}
}
func (r *MockRegistry) IsAvailable() bool {
	return !r.destroyed.Load()
}
func (r *MockRegistry) GetUrl() common.URL {
	return common.URL{}
}

func (r *MockRegistry) Subscribe(common.URL) (Listener, error) {
	return r.listener, nil
}

type listener struct {
	count      int64
	registry   *MockRegistry
	listenChan chan *ServiceEvent
}

func (l *listener) Next() (*ServiceEvent, error) {
	select {
	case e := <-l.listenChan:
		return e, nil
	}
}

func (*listener) Close() {

}

func (r *MockRegistry) MockEvent(event *ServiceEvent) {
	r.listener.listenChan <- event
}
