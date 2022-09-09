package cpu

import (
	"fmt"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestStat(t *testing.T) {
	time.Sleep(time.Second * 2)
	var i Info
	u := CpuUsage()
	i = GetInfo()
	fmt.Printf("cpu:: %+v\n", stats)
	assert.NotZero(t, u)
	assert.NotZero(t, i.Frequency)
	assert.NotZero(t, i.Quota)

	select {}
}
