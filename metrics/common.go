package metrics

import "dubbo.apache.org/dubbo-go/v3/common"

const (
	TagIp                    = "ip"
	TagPid                   = "pid"
	TagHostname              = "hostname"
	TagApplicationName       = "application.Name"
	TagApplicationModule     = "application.module.id"
	TagInterfaceKey          = "interface"
	TagMethodKey             = "method"
	TagGroupKey              = "group"
	TagVersionKey            = "Version"
	TagApplicationVersionKey = "application.Version"
	TagKeyKey                = "key"
	TagConfigCenter          = "config.center"
	TagChangeType            = "change.type"
	TagThreadName            = "thread.pool.Name"
	TagGitCommitId           = "git.commit.id"
)

type MetricKey struct {
	Name string
	Desc string
}

func NewMetricKey(name string, desc string) *MetricKey {
	return &MetricKey{Name: name, Desc: desc}
}

type MetricLevel interface {
	Tags() map[string]string
}

type ApplicationMetricLevel struct {
	ApplicationName string
	Version         string
	GitCommitId     string
}

func NewApplicationMetric(application string, version string) *ApplicationMetricLevel {
	return &ApplicationMetricLevel{
		ApplicationName: application,
		Version:         version,
	}
}

func (m ApplicationMetricLevel) Tags() map[string]string {
	tags := make(map[string]string)
	tags[TagIp] = common.GetLocalIp()
	tags[TagHostname] = common.GetLocalHostName()
	tags[TagApplicationName] = m.ApplicationName
	tags[TagApplicationVersionKey] = m.Version
	tags[TagGitCommitId] = m.GitCommitId
	return tags
}

type ServiceMetricLevel struct {
	*ApplicationMetricLevel
	InterfaceName string
}

func NewServiceMetric(application string, appVersion string, interfaceName string) *ServiceMetricLevel {
	return &ServiceMetricLevel{ApplicationMetricLevel: NewApplicationMetric(application, appVersion), InterfaceName: interfaceName}
}

func (m ServiceMetricLevel) Tags() map[string]string {
	tags := m.ApplicationMetricLevel.Tags()
	tags[TagInterfaceKey] = m.InterfaceName
	return tags
}

type MethodMetricLevel struct {
	*ServiceMetricLevel
	Method  string
	Group   string
	Version string
}

func (m MethodMetricLevel) Tags() map[string]string {
	tags := m.ServiceMetricLevel.Tags()
	tags[TagMethodKey] = m.Method
	tags[TagGroupKey] = m.Group
	tags[TagVersionKey] = m.Version
	return tags
}
