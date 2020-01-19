package protocol

import (
	"context"
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
)

func TestBeginCount(t *testing.T) {
	defer destroy()

	url, _ := common.NewURL(context.TODO(), "dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	BeginCount(url, "test")
	urlStatus := GetURLStatus(url)
	methodStatus := GetMethodStatus(url, "test")
	methodStatus1 := GetMethodStatus(url, "test1")
	assert.Equal(t, int32(1), methodStatus.active)
	assert.Equal(t, int32(1), urlStatus.active)
	assert.Equal(t, int32(0), methodStatus1.active)

}

func TestEndCount(t *testing.T) {
	defer destroy()

	url, _ := common.NewURL(context.TODO(), "dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	EndCount(url, "test", 100, true)
	urlStatus := GetURLStatus(url)
	methodStatus := GetMethodStatus(url, "test")
	assert.Equal(t, int32(-1), methodStatus.active)
	assert.Equal(t, int32(-1), urlStatus.active)
	assert.Equal(t, int32(1), methodStatus.total)
	assert.Equal(t, int32(1), urlStatus.total)
}

func TestGetMethodStatus(t *testing.T) {
	defer destroy()

	url, _ := common.NewURL(context.TODO(), "dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	status := GetMethodStatus(url, "test")
	assert.NotNil(t, status)
	assert.Equal(t, int32(0), status.total)
}

func TestGetUrlStatus(t *testing.T) {
	defer destroy()

	url, _ := common.NewURL(context.TODO(), "dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	status := GetURLStatus(url)
	assert.NotNil(t, status)
	assert.Equal(t, int32(0), status.total)
}

func Test_beginCount0(t *testing.T) {
	defer destroy()

	url, _ := common.NewURL(context.TODO(), "dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	status := GetURLStatus(url)
	beginCount0(status)
	assert.Equal(t, int32(1), status.active)
}

func Test_All(t *testing.T) {
	defer destroy()

	url, _ := common.NewURL(context.TODO(), "dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	request(url, "test", 100, false, true)
	urlStatus := GetURLStatus(url)
	methodStatus := GetMethodStatus(url, "test")
	assert.Equal(t, int32(1), methodStatus.total)
	assert.Equal(t, int32(1), urlStatus.total)
	assert.Equal(t, int32(0), methodStatus.active)
	assert.Equal(t, int32(0), urlStatus.active)
	assert.Equal(t, int32(0), methodStatus.failed)
	assert.Equal(t, int32(0), urlStatus.failed)
	assert.Equal(t, int32(0), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(0), urlStatus.successiveRequestFailureCount)
	assert.Equal(t, int64(100), methodStatus.totalElapsed)
	assert.Equal(t, int64(100), urlStatus.totalElapsed)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	request(url, "test", 100, false, false)
	assert.Equal(t, int32(6), methodStatus.total)
	assert.Equal(t, int32(6), urlStatus.total)
	assert.Equal(t, int32(5), methodStatus.failed)
	assert.Equal(t, int32(5), urlStatus.failed)
	assert.Equal(t, int32(5), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(5), urlStatus.successiveRequestFailureCount)
	assert.Equal(t, int64(600), methodStatus.totalElapsed)
	assert.Equal(t, int64(600), urlStatus.totalElapsed)
	assert.Equal(t, int64(500), methodStatus.failedElapsed)
	assert.Equal(t, int64(500), urlStatus.failedElapsed)

	request(url, "test", 100, false, true)
	assert.Equal(t, int32(0), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(0), urlStatus.successiveRequestFailureCount)

	request(url, "test", 200, false, false)
	request(url, "test", 200, false, false)
	assert.Equal(t, int32(2), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(2), urlStatus.successiveRequestFailureCount)
	assert.Equal(t, int64(200), methodStatus.maxElapsed)
	assert.Equal(t, int64(200), urlStatus.maxElapsed)

	request(url, "test1", 200, false, false)
	request(url, "test1", 200, false, false)
	request(url, "test1", 200, false, false)
	assert.Equal(t, int32(5), urlStatus.successiveRequestFailureCount)
	methodStatus1 := GetMethodStatus(url, "test1")
	assert.Equal(t, int32(2), methodStatus.successiveRequestFailureCount)
	assert.Equal(t, int32(3), methodStatus1.successiveRequestFailureCount)

}

func request(url common.URL, method string, elapsed int64, active, succeeded bool) {
	BeginCount(url, method)
	if !active {
		EndCount(url, method, elapsed, succeeded)
	}
}

func TestCurrentTimeMillis(t *testing.T) {
	defer destroy()
	c := CurrentTimeMillis()
	assert.NotNil(t, c)
	str := strconv.FormatInt(c, 10)
	i, _ := strconv.ParseInt(str, 10, 64)
	assert.Equal(t, c, i)
}

func destroy() {
	delete1 := func(key interface{}, value interface{}) bool {
		methodStatistics.Delete(key)
		return true
	}
	methodStatistics.Range(delete1)
	delete2 := func(key interface{}, value interface{}) bool {
		serviceStatistic.Delete(key)
		return true
	}
	serviceStatistic.Range(delete2)
}
