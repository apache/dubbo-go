package logger

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestInitLog(t *testing.T) {
	var (
		err  error
		path string
	)

	err = InitLog("")
	assert.EqualError(t, err, "log configure file name is nil")

	path, err = filepath.Abs("./log.xml")
	assert.NoError(t, err)
	err = InitLog(path)
	assert.EqualError(t, err, "log configure file name{"+path+"} suffix must be .yml")

	path, err = filepath.Abs("./logger.yml")
	assert.NoError(t, err)
	err = InitLog(path)
	assert.EqualError(t, err, "ioutil.ReadFile(file:"+path+") = error:open "+path+": no such file or directory")

	err = InitLog("./log.yml")
	assert.NoError(t, err)

	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")
	Debugf("%s", "debug")
	Infof("%s", "info")
	Warnf("%s", "warn")
	Errorf("%s", "error")
}
