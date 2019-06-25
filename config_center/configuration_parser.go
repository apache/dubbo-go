package config_center

import (
	"github.com/magiconair/properties"
)
import (
	"github.com/apache/dubbo-go/common/logger"
)

type ConfigurationParser interface {
	Parse(string) (map[string]string, error)
}

//for support properties file in config center
type DefaultConfigurationParser struct{}

func (parser *DefaultConfigurationParser) Parse(content string) (map[string]string, error) {
	properties, err := properties.LoadString(content)
	if err != nil {
		logger.Errorf("Parse the content {%v} in DefaultConfigurationParser error ,error message is {%v}", content, err)
		return nil, err
	}
	return properties.Map(), nil
}
