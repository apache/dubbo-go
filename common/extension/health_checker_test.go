package extension

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

func TestGetHealthChecker(t *testing.T) {
	SethealthChecker("mock", newMockhealthCheck)
	checker := GetHealthChecker("mock", common.NewURLWithOptions())
	assert.NotNil(t, checker)
}

type mockHealthChecker struct {
}

func (m mockHealthChecker) IsHealthy(invoker protocol.Invoker) bool {
	return true
}

func newMockhealthCheck(url *common.URL) router.HealthChecker {
	return &mockHealthChecker{}
}
