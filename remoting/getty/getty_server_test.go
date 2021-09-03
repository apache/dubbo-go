package getty

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitServer(t *testing.T) {
	originRootConf := config.GetRootConfig()
	rootConf := config.RootConfig{
		Protocols: map[string]*config.ProtocolConfig{
			"dubbo": {
				Name: "dubbo",
				Ip:   "127.0.0.1",
				Port: "20003",
						},
		},
	}
	config.SetRootConfig(rootConf)
	initServer("dubbo")
	config.SetRootConfig(*originRootConf)
	assert.NotNil(t, srvConf)
}
