package mock

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/registry"
	gxset "github.com/dubbogo/gost/container/set"
)

// mockMetadataReport just for test, use memory as metadata center
type mockMetadataReport struct {
	appMetadata map[string]*info.MetadataInfo // revision to info
	mapping     map[string]*gxset.HashSet     // interfaceName to app name
	listeners   map[string][]mapping.MappingListener
}

type MockMetadataFactory struct{}

func (MockMetadataFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	return &mockMetadataReport{
		appMetadata: make(map[string]*info.MetadataInfo, 0),
		mapping:     make(map[string]*gxset.HashSet, 0),
		listeners:   make(map[string][]mapping.MappingListener, 0),
	}
}

func (m *mockMetadataReport) GetAppMetadata(application, revision string) (*info.MetadataInfo, error) {
	return m.appMetadata[application+"/"+revision], nil
}

func (m *mockMetadataReport) PublishAppMetadata(application, revision string, info *info.MetadataInfo) error {
	m.appMetadata[application+"/"+revision] = info
	return nil
}

func (m *mockMetadataReport) RegisterServiceAppMapping(interfaceName, group string, application string) error {
	key := group + "/" + interfaceName
	if v, ok := m.mapping[key]; ok {
		v.Add(application)
	} else {
		m.mapping[key] = gxset.NewSet(application)
	}
	if ls, ok := m.listeners[key]; ok {
		for _, l := range ls {
			err := l.OnEvent(registry.NewServiceMappingChangedEvent(interfaceName, m.mapping[key]))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *mockMetadataReport) GetServiceAppMapping(interfaceName, group string, l mapping.MappingListener) (*gxset.HashSet, error) {
	key := group + "/" + interfaceName
	if v, ok := m.mapping[key]; ok {
		return v, nil
	}
	if ls, ok := m.listeners[key]; ok {
		ls = append(ls, l)
	} else {
		m.listeners[key] = []mapping.MappingListener{l}
	}
	return gxset.NewSet(), nil
}

func (m *mockMetadataReport) RemoveServiceAppMappingListener(interfaceName, group string) error {
	key := group + "/" + interfaceName
	delete(m.listeners, key)
	return nil
}
