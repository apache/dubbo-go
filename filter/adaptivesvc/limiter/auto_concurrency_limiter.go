package limiter

import (
	"math"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter/cpu"
)

var (
	gCPU  *atomic.Uint64 = atomic.NewUint64(0)
	decay                = 0.95
	_     Limiter        = (*AutoConcurrency)(nil)
	_     Updater        = (*AutoConcurrencyUpdater)(nil)
)

func init() {
	go cpuproc()
}

// cpu = cpuᵗ⁻¹ * decay + cpuᵗ * (1 - decay)
func cpuproc() {
	ticker := time.NewTicker(time.Millisecond * 500) // same to cpu sample rate
	defer func() {
		ticker.Stop()
		if err := recover(); err != nil {
			go cpuproc()
		}
	}()

	for range ticker.C {
		usage := cpu.CpuUsage()
		prevCPU := gCPU.Load()
		curCPU := uint64(float64(prevCPU)*decay + float64(usage)*(1.0-decay))
		logger.Debugf("current cpu usage: %d", curCPU)
		gCPU.Store(curCPU)
	}
}

func (l *AutoConcurrency) CpuUsage() uint64 {
	return gCPU.Load()
}

type AutoConcurrency struct {
	sync.RWMutex
	alpha          float64 //explore ratio TODO: it should be adjusted heuristically
	emaFactor      float64
	CPUThreshold   uint64
	noLoadLatency  float64 //duration
	maxQPS         float64
	maxConcurrency uint64

	// metrics of the current round
	avgLatency *slidingWindow

	inflight *atomic.Uint64

	prevDropTime *atomic.Duration
}

func NewAutoConcurrencyLimiter() *AutoConcurrency {
	return &AutoConcurrency{
		alpha:          0.8,
		emaFactor:      0.75,
		noLoadLatency:  0,
		maxQPS:         0,
		maxConcurrency: 8,
		CPUThreshold:   600,
		avgLatency: newSlidingWindow(slidingWindowOpts{
			Size:           20,
			BucketDuration: 50000000,
		}),
		inflight:     atomic.NewUint64(0),
		prevDropTime: atomic.NewDuration(0),
	}
}

func (l *AutoConcurrency) updateNoLoadLatency(latency float64) {
	if l.noLoadLatency <= 0 {
		l.noLoadLatency = latency
	} else if latency < l.noLoadLatency {
		l.noLoadLatency = latency*l.emaFactor + l.noLoadLatency*(1-l.emaFactor)
	}
}

func (l *AutoConcurrency) updateQPS(qps float64) {
	if l.maxQPS <= qps {
		l.maxQPS = qps
	} else {
		l.maxQPS = qps*l.emaFactor + l.maxQPS*(1-l.emaFactor)
	}
}

func (l *AutoConcurrency) updateMaxConcurrency(v uint64) {
	if l.maxConcurrency <= v {
		l.maxConcurrency = v
	} else {
		l.maxConcurrency = uint64(float64(v)*l.emaFactor + float64(l.maxConcurrency)*(1-l.emaFactor))
	}
}

func (l *AutoConcurrency) Inflight() uint64 {
	return l.inflight.Load()
}

func (l *AutoConcurrency) Remaining() uint64 {
	return l.maxConcurrency - l.inflight.Load()
}

func (l *AutoConcurrency) Acquire() (Updater, error) {
	now := time.Now()
	if l.inflight.Inc() > l.maxConcurrency {
		drop := false
		if l.CpuUsage() >= l.CPUThreshold {
			drop = true
		}
		prevDrop := l.prevDropTime.Load()
		//if prevDrop != 0 {
		//	// already started drop, return directly
		//	drop = true
		//}
		if time.Duration(now.UnixNano())-prevDrop <= time.Second {
			drop = true
		}
		if drop {
			l.inflight.Dec()
			// store start drop time
			l.prevDropTime.Store(time.Duration(now.Unix()))
			return nil, ErrReachLimitation
		}
	}

	l.prevDropTime.Store(time.Duration(0))
	u := &AutoConcurrencyUpdater{
		startTime: now,
		limiter:   l,
	}
	return u, nil
}

type AutoConcurrencyUpdater struct {
	startTime time.Time
	limiter   *AutoConcurrency
}

func (u *AutoConcurrencyUpdater) DoUpdate() error {
	defer func() {
		u.limiter.inflight.Dec()
	}()
	latency := float64(time.Now().UnixNano()-u.startTime.UnixNano()) / 1e6
	u.limiter.avgLatency.Add(latency)
	u.limiter.Lock()
	defer u.limiter.Unlock()
	reqPerMilliseconds := float64(u.limiter.avgLatency.MaxCount()) / float64(u.limiter.avgLatency.BucketDuration())
	u.limiter.updateNoLoadLatency(latency)
	u.limiter.updateQPS(reqPerMilliseconds)
	nextMaxConcurrency := u.limiter.maxQPS * ((1 + u.limiter.alpha) * u.limiter.noLoadLatency)
	u.limiter.updateMaxConcurrency(uint64(math.Ceil(nextMaxConcurrency)))
	logger.Debugf("[Auto Concurrency Limiter] QPMilli: %v, NoLoadLatency: %f, avgLatency: %f, MaxConcurrency: %d",
		reqPerMilliseconds, u.limiter.noLoadLatency, u.limiter.avgLatency.avg, u.limiter.maxConcurrency)
	return nil
}

// slidingWindow is a policy for ring window based on time duration.
// slidingWindow moves bucket offset with time duration.
// e.g. If the last point is appended one bucket duration ago,
// slidingWindow will increment current offset.
type slidingWindow struct {
	size           int
	mu             sync.Mutex
	buckets        []bucket //sum+cnt
	count          int64
	avg            float64
	sum            float64
	offset         int
	bucketDuration time.Duration
	lastAppendTime time.Time
}

type bucket struct {
	cnt int64
	sum float64
}

// SlidingWindowAvgOpts contains the arguments for creating SlidingWindowCounter.
type slidingWindowOpts struct {
	Size           int
	BucketDuration time.Duration
}

// NewSlidingWindowAvg creates a new SlidingWindowCounter based on the given window and SlidingWindowCounterOpts.
func newSlidingWindow(opts slidingWindowOpts) *slidingWindow {
	buckets := make([]bucket, opts.Size)

	return &slidingWindow{
		size:           opts.Size,
		offset:         0,
		buckets:        buckets,
		bucketDuration: opts.BucketDuration,
		lastAppendTime: time.Now(),
	}
}

func (w *slidingWindow) timespan() int {
	v := int(time.Since(w.lastAppendTime) / w.bucketDuration)
	if v > -1 { // maybe time backwards
		return v
	}
	return w.size
}

func (w *slidingWindow) Add(v float64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	//move offset
	timespan := w.timespan()
	if timespan > 0 {
		start := (w.offset + 1) % w.size
		end := (w.offset + timespan) % w.size
		// reset the expired buckets
		w.ResetBuckets(start, timespan)
		w.offset = end
		w.lastAppendTime = w.lastAppendTime.Add(time.Duration(timespan * int(w.bucketDuration)))
	}

	w.buckets[w.offset].cnt++
	w.buckets[w.offset].sum += v

	w.sum += v
	w.count++
	w.avg = w.sum / float64(w.count)
}

func (w *slidingWindow) Value() float64 {
	return w.avg
}

func (w *slidingWindow) Count() int64 {
	return w.count
}

func (w *slidingWindow) MaxCount() int64 {
	max := int64(0)
	for _, b := range w.buckets {
		if max < b.cnt {
			max = b.cnt
		}
	}
	return max
}

func (w *slidingWindow) Avg() float64 {
	return w.avg
}

func (w *slidingWindow) BucketDuration() int64 {
	return w.bucketDuration.Milliseconds()
}

func (w *slidingWindow) TimespanMilliseconds() int64 {
	return w.bucketDuration.Milliseconds() * int64(w.size)
}

// ResetBucket empties the bucket based on the given offset.
func (w *slidingWindow) ResetBucket(offset int) {
	w.sum -= w.buckets[offset%w.size].sum
	w.count -= w.buckets[offset%w.size].cnt
	w.avg = w.sum / float64(w.count)

	w.buckets[offset%w.size].cnt = 0
	w.buckets[offset%w.size].sum = 0
}

// ResetBuckets empties the buckets based on the given offsets.
func (w *slidingWindow) ResetBuckets(offset int, count int) {
	if count > w.size {
		count = w.size
	}
	for i := 0; i < count; i++ {
		w.ResetBucket(offset + i)
	}
}
