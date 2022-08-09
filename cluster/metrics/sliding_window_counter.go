package metrics

import (
	"fmt"
	"sync"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// SlidingWindowCounter is a policy for ring window based on time duration.
// SlidingWindowCounter moves bucket offset with time duration.
// e.g. If the last point is appended one bucket duration ago,
// SlidingWindowCounter will increment current offset.
type SlidingWindowCounter struct {
	size           int
	mu             sync.Mutex
	buckets        []int64
	count          int64
	offset         int
	bucketDuration time.Duration
	lastAppendTime time.Time
}

// SlidingWindowCounterOpts contains the arguments for creating SlidingWindowCounter.
type SlidingWindowCounterOpts struct {
	Size           int
	BucketDuration time.Duration
}

// NewSlidingWindowCounter creates a new SlidingWindowCounter based on the given window and SlidingWindowCounterOpts.
func NewSlidingWindowCounter(opts SlidingWindowCounterOpts) *SlidingWindowCounter {
	buckets := make([]int64, opts.Size)

	return &SlidingWindowCounter{
		size:           opts.Size,
		offset:         0,
		buckets:        buckets,
		bucketDuration: opts.BucketDuration,
		lastAppendTime: time.Now(),
	}
}

func (c *SlidingWindowCounter) timespan() int {
	v := int(time.Since(c.lastAppendTime) / c.bucketDuration)
	if v > -1 { // maybe time backwards
		return v
	}
	return c.size
}

func (c *SlidingWindowCounter) Add(_ int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	//move offset
	timespan := c.timespan()
	if timespan > 0 {
		start := (c.offset + 1) % c.size
		end := (c.offset + timespan) % c.size
		// reset the expired buckets
		c.ResetBuckets(start, timespan)
		c.offset = end
		c.lastAppendTime = c.lastAppendTime.Add(time.Duration(timespan * int(c.bucketDuration)))
	}

	c.buckets[c.offset]++
	c.count++
}

func (c *SlidingWindowCounter) Value() int64 {
	return c.count
}

// ResetBucket empties the bucket based on the given offset.
func (c *SlidingWindowCounter) ResetBucket(offset int) {
	c.count -= c.buckets[offset%c.size]
	c.buckets[offset%c.size] = 0
}

// ResetBuckets empties the buckets based on the given offsets.
func (c *SlidingWindowCounter) ResetBuckets(offset int, count int) {
	if count > c.size {
		count = c.size
	}
	for i := 0; i < count; i++ {
		c.ResetBucket(offset + i)
	}
}

var SlidingWindowCounterMetrics Metrics

func init() {
	SlidingWindowCounterMetrics = newSlidingWindowCounterMetrics()
}

var _ Metrics = (*slidingWindowCounterMetrics)(nil)

type slidingWindowCounterMetrics struct {
	opts    SlidingWindowCounterOpts
	metrics sync.Map
}

func newSlidingWindowCounterMetrics() *slidingWindowCounterMetrics {
	return &slidingWindowCounterMetrics{
		opts: SlidingWindowCounterOpts{
			Size:           10,
			BucketDuration: 20000000,
		},
	}
}

func (m *slidingWindowCounterMetrics) GetMethodMetrics(url *common.URL, methodName, key string) (interface{}, error) {
	metricsKey := fmt.Sprintf("%s.%s.%s.%s", getInstanceKey(url), getInvokerKey(url), methodName, key)
	if metrics, ok := m.metrics.Load(metricsKey); ok {
		return metrics.(*SlidingWindowCounter).Value(), nil
	}
	return int64(0), ErrMetricsNotFound
}

func (m *slidingWindowCounterMetrics) SetMethodMetrics(url *common.URL, methodName, key string, _ interface{}) error {
	metricsKey := fmt.Sprintf("%s.%s.%s.%s", getInstanceKey(url), getInvokerKey(url), methodName, key)
	if metrics, ok := m.metrics.Load(metricsKey); ok {
		metrics.(*SlidingWindowCounter).Add(1)
	} else {
		metrics, _ = m.metrics.LoadOrStore(metricsKey, NewSlidingWindowCounter(m.opts))
		metrics.(*SlidingWindowCounter).Add(1)
	}
	return nil
}

func (m *slidingWindowCounterMetrics) GetInvokerMetrics(url *common.URL, key string) (interface{}, error) {
	panic("implement me")
}

func (m *slidingWindowCounterMetrics) SetInvokerMetrics(url *common.URL, key string, value interface{}) error {
	panic("implement me")
}

func (m *slidingWindowCounterMetrics) GetInstanceMetrics(url *common.URL, key string) (interface{}, error) {
	panic("implement me")
}

func (m *slidingWindowCounterMetrics) SetInstanceMetrics(url *common.URL, key string, value interface{}) error {
	panic("implement me")
}
