package config

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	os.Setenv(constant.CONFIG_FILE_ENV_KEY, "../config/testdata/config/app/application.yaml")
	Load()
}
