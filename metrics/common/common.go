package common

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
)

const (
	TagIp                    = "ip"
	TagPid                   = "pid"
	TagHostname              = "hostname"
	TagApplicationName       = "application_name"
	TagApplicationModule     = "application_module_id"
	TagInterfaceKey          = "interface"
	TagMethodKey             = "method"
	TagGroupKey              = "group"
	TagVersionKey            = "version"
	TagApplicationVersionKey = "application_version"
	TagKeyKey                = "key"
	TagConfigCenter          = "config_center"
	TagChangeType            = "change_type"
	TagThreadName            = "thread_pool_name"
	TagGitCommitId           = "git_commit_id"
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
	Ip string
	HostName string
}

var appLevel *ApplicationMetricLevel

func NewApplicationLevel() *ApplicationMetricLevel {
	if appLevel == nil {
		var rootConfig = config.GetRootConfig()
		appLevel = &ApplicationMetricLevel{
			ApplicationName: rootConfig.Application.Name,
			Version: rootConfig.Application.Version,
			Ip: common.GetLocalIp(),
			HostName: common.GetLocalHostName(),
			GitCommitId: "",
		}
	}
	return appLevel
}

func (m *ApplicationMetricLevel) Tags() map[string]string {
	tags := make(map[string]string)
	tags[TagIp] = m.Ip
	tags[TagHostname] = m.HostName
	tags[TagApplicationName] = m.ApplicationName
	tags[TagApplicationVersionKey] = m.Version
	tags[TagGitCommitId] = m.GitCommitId
	return tags
}

type ServiceMetricLevel struct {
	*ApplicationMetricLevel
	Interface string
}

func NewServiceMetric(interfaceName string) *ServiceMetricLevel {
	return &ServiceMetricLevel{ApplicationMetricLevel: NewApplicationLevel(), Interface: interfaceName}
}

func (m ServiceMetricLevel) Tags() map[string]string {
	tags := m.ApplicationMetricLevel.Tags()
	tags[TagInterfaceKey] = m.Interface
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
