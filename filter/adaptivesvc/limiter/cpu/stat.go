package cpu

import (
	"fmt"
	"time"
)

import (
	"go.uber.org/atomic"
)

const (
	interval time.Duration = time.Millisecond * 500
)

var (
	stats CPU
	usage = atomic.NewUint64(0)
)

// CPU is cpu stat usage.
type CPU interface {
	Usage() (u uint64, e error)
	Info() Info
}

func init() {
	var (
		err error
	)
	stats, err = newCgroupCPU()
	if err != nil {
		panic(fmt.Sprintf("cgroup cpu init failed! err:=%v", err))
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			<-ticker.C
			u, err := stats.Usage()
			if err == nil && u != 0 {
				usage.Store(u)
			}
		}
	}()
}

// Info cpu info.
type Info struct {
	Frequency uint64
	Quota     float64
}

// CpuUsage read cpu stat.
func CpuUsage() uint64 {
	return usage.Load()
}

// GetInfo get cpu info.
func GetInfo() Info {
	return stats.Info()
}
