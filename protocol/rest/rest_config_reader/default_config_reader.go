package rest_config_reader

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
	perrors "github.com/pkg/errors"
)

const (
	DEFAULT_READER = "default"
)

var (
	defaultConfigReader *DefaultConfigReader
)

func init() {
	extension.SetRestConfigReader(DEFAULT_READER, GetDefaultConfigReader)
}

type DefaultConfigReader struct {
}

func NewDefaultConfigReader() *DefaultConfigReader {
	return &DefaultConfigReader{}
}

func (dcr *DefaultConfigReader) ReadConsumerConfig() *rest_interface.RestConsumerConfig {
	confConFile := os.Getenv(constant.CONF_CONSUMER_FILE_PATH)
	if confConFile == "" {
		logger.Warnf("rest consumer configure(consumer) file name is nil")
		return nil
	}
	if path.Ext(confConFile) != ".yml" {
		logger.Warnf("rest consumer configure file name{%v} suffix must be .yml", confConFile)
		return nil
	}
	confFileStream, err := ioutil.ReadFile(confConFile)
	if err != nil {
		logger.Warnf("ioutil.ReadFile(file:%s) = error:%v", confConFile, perrors.WithStack(err))
		return nil
	}
	restConsumerConfig := &rest_interface.RestConsumerConfig{}
	err = yaml.Unmarshal(confFileStream, restConsumerConfig)
	if err != nil {
		logger.Warnf("yaml.Unmarshal() = error:%v", perrors.WithStack(err))
		return nil
	}
	return restConsumerConfig
}

func (dcr *DefaultConfigReader) ReadProviderConfig() *rest_interface.RestProviderConfig {
	confProFile := os.Getenv(constant.CONF_PROVIDER_FILE_PATH)
	if len(confProFile) == 0 {
		logger.Warnf("rest provider configure(provider) file name is nil")
		return nil
	}

	if path.Ext(confProFile) != ".yml" {
		logger.Warnf("rest provider configure file name{%v} suffix must be .yml", confProFile)
		return nil
	}
	confFileStream, err := ioutil.ReadFile(confProFile)
	if err != nil {
		logger.Warnf("ioutil.ReadFile(file:%s) = error:%v", confProFile, perrors.WithStack(err))
		return nil
	}
	restProviderConfig := &rest_interface.RestProviderConfig{}
	err = yaml.Unmarshal(confFileStream, restProviderConfig)
	if err != nil {
		logger.Warnf("yaml.Unmarshal() = error:%v", perrors.WithStack(err))
		return nil
	}

	return restProviderConfig
}

func GetDefaultConfigReader() rest_interface.RestConfigReader {
	if defaultConfigReader == nil {
		defaultConfigReader = NewDefaultConfigReader()
	}
	return defaultConfigReader
}
