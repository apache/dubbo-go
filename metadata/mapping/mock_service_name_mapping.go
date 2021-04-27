package mapping

import (
	gxset "github.com/dubbogo/gost/container/set"
)

type MockServiceNameMapping struct{}

func NewMockServiceNameMapping() *MockServiceNameMapping {
	return &MockServiceNameMapping{}
}

func (m *MockServiceNameMapping) Map(string, string, string, string) error {
	return nil
}

func (m *MockServiceNameMapping) Get(string, string, string, string) (*gxset.HashSet, error) {
	panic("implement me")
}
