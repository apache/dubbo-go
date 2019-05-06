package registry

import (
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/tevino/abool"
)

type MockRegistry struct {
	listener  *listener
	destroyed *abool.AtomicBool
}

func NewMockRegistry() *MockRegistry {
	registry := &MockRegistry{
		destroyed: abool.NewBool(false),
	}
	listener := &listener{count: 0, registry: registry, listenChan: make(chan *ServiceEvent)}
	registry.listener = listener
	return registry
}
func (*MockRegistry) Register(url config.URL) error {
	return nil
}

func (r *MockRegistry) Destroy() {
	if r.destroyed.SetToIf(false, true) {
	}
}
func (r *MockRegistry) IsAvailable() bool {
	return !r.destroyed.IsSet()
}
func (r *MockRegistry) GetUrl() config.URL {
	return config.URL{}
}

func (r *MockRegistry) Subscribe(config.URL) (Listener, error) {
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
