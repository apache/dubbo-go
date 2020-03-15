package yaml

import (
	"path/filepath"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalYMLConfig(t *testing.T) {
	conPath, err := filepath.Abs("./testdata/config.yml")
	assert.NoError(t, err)
	c := &Config{}
	_, err = UnmarshalYMLConfig(conPath, c)
	assert.NoError(t, err)
	assert.Equal(t, "strTest", c.StrTest)
	assert.Equal(t, 11, c.IntTest)
	assert.Equal(t, false, c.BooleanTest)
	assert.Equal(t, "childStrTest", c.ChildConfig.StrTest)
}

func TestUnmarshalYMLConfig_Error(t *testing.T) {
	c := &Config{}
	_, err := UnmarshalYMLConfig("./testdata/config", c)
	assert.Error(t, err)
	_, err = UnmarshalYMLConfig("", c)
	assert.Error(t, err)
}

type Config struct {
	StrTest     string      `yaml:"strTest" default:"default" json:"strTest,omitempty" property:"strTest"`
	IntTest     int         `default:"109"  yaml:"intTest" json:"intTest,omitempty" property:"intTest"`
	BooleanTest bool        `yaml:"booleanTest" default:"true" json:"booleanTest,omitempty"`
	ChildConfig ChildConfig `yaml:"child" json:"child,omitempty"`
}

type ChildConfig struct {
	StrTest string `default:"strTest" default:"default" yaml:"strTest"  json:"strTest,omitempty"`
}
