package customizer

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"strconv"
)

func init() {
	extension.AddCustomizers(&weightCustomizer{})
}

type weightCustomizer struct{}

func (w weightCustomizer) GetPriority() int {
	return 1
}

func (w weightCustomizer) Customize(instance registry.ServiceInstance) {
	if instance.GetServiceMetadata() != nil {
		if instance.GetServiceMetadata().GetExportedServiceURLs() != nil {
			for _, url := range instance.GetServiceMetadata().GetExportedServiceURLs() {
				instance.GetMetadata()[constant.WeightKey] =
					url.GetParam(constant.WeightKey, strconv.FormatInt(constant.DefaultWeight, 10))
			}
		}
	}
}
