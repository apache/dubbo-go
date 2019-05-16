package dubbo

import (
	"testing"
	"time"
)

import (
	"github.com/dubbogo/hessian2"
	"github.com/stretchr/testify/assert"
)

func TestDubboPackage_MarshalAndUnmarshal(t *testing.T) {
	pkg := &DubboPackage{}
	pkg.Body = []interface{}{"a"}
	pkg.Header.Type = hessian.PackageHeartbeat
	pkg.Header.SerialID = byte(S_Dubbo)
	pkg.Header.ID = 10086

	// heartbeat
	data, err := pkg.Marshal()
	assert.NoError(t, err)

	pkgres := &DubboPackage{}
	pkgres.Body = []interface{}{}
	err = pkgres.Unmarshal(data)
	assert.NoError(t, err)
	assert.Equal(t, hessian.PackageHeartbeat|hessian.PackageRequest|hessian.PackageRequest_TwoWay, pkgres.Header.Type)
	assert.Equal(t, byte(S_Dubbo), pkgres.Header.SerialID)
	assert.Equal(t, int64(10086), pkgres.Header.ID)
	assert.Equal(t, 0, len(pkgres.Body.([]interface{})))

	// request
	pkg.Header.Type = hessian.PackageRequest
	pkg.Service.Interface = "Service"
	pkg.Service.Target = "Service"
	pkg.Service.Version = "2.6"
	pkg.Service.Method = "Method"
	pkg.Service.Timeout = time.Second
	data, err = pkg.Marshal()
	assert.NoError(t, err)

	pkgres = &DubboPackage{}
	pkgres.Body = make([]interface{}, 7)
	err = pkgres.Unmarshal(data)
	assert.NoError(t, err)
	assert.Equal(t, hessian.PackageRequest|hessian.PackageRequest_TwoWay, pkgres.Header.Type)
	assert.Equal(t, byte(S_Dubbo), pkgres.Header.SerialID)
	assert.Equal(t, int64(10086), pkgres.Header.ID)
	assert.Equal(t, "2.5.4", pkgres.Body.([]interface{})[0])
	assert.Equal(t, "Service", pkgres.Body.([]interface{})[1])
	assert.Equal(t, "2.6", pkgres.Body.([]interface{})[2])
	assert.Equal(t, "Method", pkgres.Body.([]interface{})[3])
	assert.Equal(t, "Ljava/lang/String;", pkgres.Body.([]interface{})[4])
	assert.Equal(t, []interface{}{"a"}, pkgres.Body.([]interface{})[5])
	assert.Equal(t, map[interface{}]interface{}{"interface": "Service", "path": "", "timeout": "1000"}, pkgres.Body.([]interface{})[6])
}
