package yaml

import (
	"io/ioutil"
	"path"
)

import (
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func UnmarshalYMLConfig(yamlFile string, out interface{}) error {
	if path.Ext(yamlFile) != ".yml" {
		return perrors.Errorf("yamlFile name{%v} suffix must be .yml", yamlFile)
	}
	confFileStream, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		return perrors.Errorf("ioutil.ReadFile(file:%s) = error:%v", yamlFile, perrors.WithStack(err))
	}
	err = yaml.Unmarshal(confFileStream, out)
	if err != nil {
		return perrors.Errorf("yaml.Unmarshal() = error:%v", perrors.WithStack(err))
	}
	return nil
}
