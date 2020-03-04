package condition

import (
	"math"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

func TestDefaultHealthChecker_IsHealthy(t *testing.T) {
	url, _ := common.NewURL("dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	hc := NewDefaultHealthChecker(&url).(*DefaultHealthChecker)
	invoker := NewMockInvoker(url, 1)
	healthy := hc.IsHealthy(invoker)
	assert.True(t, healthy)

	url.SetParam(OUTSTANDING_REQUEST_COUNT_LIMIT_KEY, "10")
	url.SetParam(SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, "100")
	// fake the outgoing request
	for i := 0; i < 11; i++ {
		request(url, "test", 0, true, false)
	}
	hc = NewDefaultHealthChecker(&url).(*DefaultHealthChecker)
	healthy = hc.IsHealthy(invoker)
	// the outgoing request is more than OUTSTANDING_REQUEST_COUNT_LIMIT, go to unhealthy
	assert.False(t, hc.IsHealthy(invoker))

	// successive failed count is more than SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, go to unhealthy
	for i := 0; i < 11; i++ {
		request(url, "test", 0, false, false)
	}
	url.SetParam(SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, "10")
	url.SetParam(OUTSTANDING_REQUEST_COUNT_LIMIT_KEY, "1000")
	hc = NewDefaultHealthChecker(&url).(*DefaultHealthChecker)
	healthy = hc.IsHealthy(invoker)
	assert.False(t, hc.IsHealthy(invoker))

	// reset successive failed count and go to healthy
	request(url, "test", 0, false, true)
	healthy = hc.IsHealthy(invoker)
	assert.True(t, hc.IsHealthy(invoker))
}

func TestDefaultHealthChecker_getCircuitBreakerSleepWindowTime(t *testing.T) {

	url, _ := common.NewURL("dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	defaultHc := NewDefaultHealthChecker(&url).(*DefaultHealthChecker)
	// Increase the number of failed requests
	for i := 0; i < 100; i++ {
		request(url, "test", 1, false, false)
	}
	sleepWindowTime := defaultHc.getCircuitBreakerSleepWindowTime(protocol.GetURLStatus(url))
	assert.True(t, sleepWindowTime == MAX_CIRCUIT_TRIPPED_TIMEOUT)

	// Adjust the threshold size to 1000
	url.SetParam(SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, "1000")
	sleepWindowTime = NewDefaultHealthChecker(&url).(*DefaultHealthChecker).getCircuitBreakerSleepWindowTime(protocol.GetURLStatus(url))
	assert.True(t, sleepWindowTime == 0)

	url1, _ := common.NewURL("dubbo://192.168.10.11:20000/com.ikurento.user.UserProvider")
	sleepWindowTime = defaultHc.getCircuitBreakerSleepWindowTime(protocol.GetURLStatus(url1))
	assert.True(t, sleepWindowTime == 0)
	request(url1, "test", 1, false, false)
	request(url1, "test", 1, false, false)
	request(url1, "test", 1, false, false)
	request(url1, "test", 1, false, false)
	request(url1, "test", 1, false, false)
	request(url1, "test", 1, false, false)
	sleepWindowTime = defaultHc.getCircuitBreakerSleepWindowTime(protocol.GetURLStatus(url1))
	assert.True(t, sleepWindowTime > 0 && sleepWindowTime < MAX_CIRCUIT_TRIPPED_TIMEOUT)

}

func TestDefaultHealthChecker_getCircuitBreakerTimeout(t *testing.T) {
	url, _ := common.NewURL("dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	defaultHc := NewDefaultHealthChecker(&url).(*DefaultHealthChecker)
	timeout := defaultHc.getCircuitBreakerTimeout(protocol.GetURLStatus(url))
	assert.True(t, timeout == 0)
	url1, _ := common.NewURL("dubbo://192.168.10.11:20000/com.ikurento.user.UserProvider")
	request(url1, "test", 1, false, false)
	request(url1, "test", 1, false, false)
	request(url1, "test", 1, false, false)
	request(url1, "test", 1, false, false)
	request(url1, "test", 1, false, false)
	request(url1, "test", 1, false, false)
	timeout = defaultHc.getCircuitBreakerTimeout(protocol.GetURLStatus(url1))
	// timeout must after the current time
	assert.True(t, timeout > protocol.CurrentTimeMillis())

}

func TestDefaultHealthChecker_isCircuitBreakerTripped(t *testing.T) {
	url, _ := common.NewURL("dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	defaultHc := NewDefaultHealthChecker(&url).(*DefaultHealthChecker)
	status := protocol.GetURLStatus(url)
	tripped := defaultHc.isCircuitBreakerTripped(status)
	assert.False(t, tripped)
	// Increase the number of failed requests
	for i := 0; i < 100; i++ {
		request(url, "test", 1, false, false)
	}
	tripped = defaultHc.isCircuitBreakerTripped(protocol.GetURLStatus(url))
	assert.True(t, tripped)

}

func TestNewDefaultHealthChecker(t *testing.T) {
	url, _ := common.NewURL("dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	defaultHc := NewDefaultHealthChecker(&url).(*DefaultHealthChecker)
	assert.NotNil(t, defaultHc)
	assert.Equal(t, defaultHc.OutStandingRequestConutLimit, int32(math.MaxInt32))
	assert.Equal(t, defaultHc.RequestSuccessiveFailureThreshold, int32(DEFAULT_SUCCESSIVE_FAILED_REQUEST_MAX_DIFF))

	url1, _ := common.NewURL("dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	url1.SetParam(OUTSTANDING_REQUEST_COUNT_LIMIT_KEY, "10")
	url1.SetParam(SUCCESSIVE_FAILED_REQUEST_THRESHOLD_KEY, "10")
	nondefaultHc := NewDefaultHealthChecker(&url1).(*DefaultHealthChecker)
	assert.NotNil(t, nondefaultHc)
	assert.Equal(t, nondefaultHc.OutStandingRequestConutLimit, int32(10))
	assert.Equal(t, nondefaultHc.RequestSuccessiveFailureThreshold, int32(10))
}

func request(url common.URL, method string, elapsed int64, active, succeeded bool) {
	protocol.BeginCount(url, method)
	if !active {
		protocol.EndCount(url, method, elapsed, succeeded)
	}
}
