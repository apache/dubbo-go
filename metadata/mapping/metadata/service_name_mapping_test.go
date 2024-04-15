package metadata

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetNameMappingInstance(t *testing.T) {
	ins := GetNameMappingInstance()
	assert.NotNil(t, ins)
}

func TestNoReportInstance(t *testing.T) {
	ins := GetNameMappingInstance()
	lis := &listener{}
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
	)
	_, err := ins.Get(serviceUrl, lis)
	assert.NotNil(t, err, "test Get no report instance")
	err = ins.Map(serviceUrl)
	assert.NotNil(t, err, "test Map with no report instance")
	err = ins.Remove(serviceUrl)
	assert.NotNil(t, err, "test Remove with no report instance")
}

func TestServiceNameMappingGet(t *testing.T) {
	ins := GetNameMappingInstance()
	lis := &listener{}
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
	)
	_, err := initMock()
	assert.Nil(t, err)
	apps, err := ins.Get(serviceUrl, lis)
	assert.True(t, apps.Empty(), "test with report instance and no app")
	err = ins.Map(serviceUrl)
	assert.Nil(t, err)
	apps, err = ins.Get(serviceUrl, lis)
	assert.True(t, !apps.Empty(), "test with report instance and app")
}

func TestServiceNameMappingMap(t *testing.T) {
	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
	)
	reportIns, err := initMock()
	assert.Nil(t, err)
	err = ins.Map(serviceUrl)
	assert.Nil(t, err, "test with report instance")
	assert.Equal(t, reportIns.(*mockMetadataReport).app, serviceUrl.GetParam(constant.ApplicationKey, ""))
	assert.Equal(t, reportIns.(*mockMetadataReport).interfaceName, serviceUrl.Interface())
	assert.Equal(t, reportIns.(*mockMetadataReport).group, DefaultGroup)
	reportIns.(*mockMetadataReport).thr = true
	err = ins.Map(serviceUrl)
	assert.NotNil(t, err, "test mapping error")
}

func TestServiceNameMappingRemove(t *testing.T) {
	ins := GetNameMappingInstance()
	serviceUrl := common.NewURLWithOptions(
		common.WithInterface("org.apache.dubbo.samples.proto.GreetService"),
		common.WithParamsValue(constant.ApplicationKey, "dubbo"),
	)
	reportIns, err := initMock()
	err = ins.Map(serviceUrl)
	assert.Nil(t, err)
	err = ins.Remove(serviceUrl)
	assert.Nil(t, err)
	assert.Nil(t, reportIns.(*mockMetadataReport).listener)
	assert.Equal(t, reportIns.(*mockMetadataReport).group, "")
	assert.Equal(t, reportIns.(*mockMetadataReport).interfaceName, "")
}

func initMock() (report.MetadataReport, error) {
	metadataReport := &mockMetadataReport{mapping: make(map[string]string, 0)}
	extension.SetMetadataReportFactory("mock", func() report.MetadataReportFactory {
		return metadataReport
	})
	opts := metadata.NewReportOptions(
		metadata.WithProtocol("mock"),
		metadata.WithAddress("127.0.0.1"),
	)
	err := opts.Init()
	return metadataReport, err
}

type listener struct {
}

func (l listener) OnEvent(e observer.Event) error {
	return nil
}

func (l listener) Stop() {
}

type mockMetadataReport struct {
	app           string
	group         string
	revision      string
	interfaceName string
	info          *info.MetadataInfo
	mapping       map[string]string
	listener      mapping.MappingListener
	thr           bool // test register map error
}

func (m *mockMetadataReport) CreateMetadataReport(url *common.URL) report.MetadataReport {
	return m
}

func (m *mockMetadataReport) GetAppMetadata(application, revision string) (*info.MetadataInfo, error) {
	m.app = application
	m.revision = revision
	return m.info, nil
}

func (m *mockMetadataReport) PublishAppMetadata(application, revision string, info *info.MetadataInfo) error {
	m.app = application
	m.revision = revision
	m.info = info
	return nil
}

func (m *mockMetadataReport) RegisterServiceAppMapping(interfaceName, group string, application string) error {
	if m.thr {
		return errors.New("write mapping error")
	}
	m.app = application
	m.group = group
	m.interfaceName = interfaceName
	if m.mapping == nil {
		m.mapping = make(map[string]string, 0)
	}
	m.mapping[interfaceName] = application
	return nil
}

func (m *mockMetadataReport) GetServiceAppMapping(interfaceName, group string, l mapping.MappingListener) (*gxset.HashSet, error) {
	m.interfaceName = interfaceName
	m.group = group
	set := gxset.NewSet()
	if v, ok := m.mapping[interfaceName]; ok {
		set.Add(v)
	}
	m.listener = l
	return set, nil
}

func (m *mockMetadataReport) RemoveServiceAppMappingListener(interfaceName, group string) error {
	m.interfaceName = ""
	m.group = ""
	m.listener = nil
	return nil
}
