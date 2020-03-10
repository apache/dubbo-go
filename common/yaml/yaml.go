package yaml

import (
	"io/ioutil"
	"path"
)

import (
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// loadYMLConfig Load yml config byte from file
func LoadYMLConfig(confProFile string) ([]byte, error) {
	if len(confProFile) == 0 {
		return nil, perrors.Errorf("application configure(provider) file name is nil")
	}

	if path.Ext(confProFile) != ".yml" {
		return nil, perrors.Errorf("application configure file name{%v} suffix must be .yml", confProFile)
	}

	return ioutil.ReadFile(confProFile)
}

// unmarshalYMLConfig Load yml config byte from file , then unmarshal to object
func UnmarshalYMLConfig(confProFile string, out interface{}) error {
	confFileStream, err := LoadYMLConfig(confProFile)
	if err != nil {
		return perrors.Errorf("ioutil.ReadFile(file:%s) = error:%v", confProFile, perrors.WithStack(err))
	}
	return yaml.Unmarshal(confFileStream, out)
}
