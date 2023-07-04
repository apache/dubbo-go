package metrics

var registries = make(map[string]func(*ReporterConfig) MetricRegistry)
var collectors = make([]CollectorFunc, 0)
var registry MetricRegistry

// CollectorFunc 各个指标处理模块扩展
type CollectorFunc func(MetricRegistry, *ReporterConfig)

// Init 整个 Metrics 模块初始化入口
func Init(c *ReporterConfig) {
	// config.extention = prometheus
	regFunc, ok := registries["config.extention"]
	if !ok {
		regFunc, _ = registries["prometheus"] // default
	}
	registry = regFunc(c)
	for _, v := range collectors {
		v(registry, c)
	}
	registry.Export()
}

// SetRegistry 扩展其他数据容器，暴露方式，内置 Prometheus 实现
func SetRegistry(name string, v func(*ReporterConfig) MetricRegistry) {
	registries[name] = v
}

// AddCollector 扩展指标收集器，例如 metadata、耗时、配置中心等
func AddCollector(name string, fun func(MetricRegistry, *ReporterConfig)) {
	collectors = append(collectors, fun)
}

// MetricRegistry 数据指标容器，指标计算、指标暴露、聚合
type MetricRegistry interface {
	Counter(*MetricId) CounterMetric     // add or update a counter
	Gauge(*MetricId) GaugeMetric         // add or update a gauge
	Histogram(*MetricId) HistogramMetric // add a metric num to a histogram
	Summary(*MetricId) SummaryMetric     // add a metric num to a summary
	Export() // 数据暴露， 如 Prometheus 是 http 暴露
	// GetMetrics() []*MetricSample // 获取所有指标数据
	// GetMetricsString() (string, error) // 如需复用端口则加一下这个接口
}

// 组合暴露方式，参考 micrometer CompositeMeterRegistry
//type CompositeRegistry struct {
//	rs []MetricRegistry
//}

// Type 指标类型，暂定和 micrometer 一致
type Type uint8

const (
	Counter Type = iota
	Gauge
	LongTaskTimer
	Timer
	DistributionSummary
	Other
)

// MetricId
// # HELP dubbo_metadata_store_provider_succeed_total Succeed Store Provider Metadata
// # TYPE dubbo_metadata_store_provider_succeed_total gauge
// dubbo_metadata_store_provider_succeed_total{application_name="provider",hostname="localhost",interface="org.example.DemoService",ip="10.252.156.213",} 1.0
// 除值以外的其他属性
type MetricId struct {
	Name string
	Desc string
	Tags map[string]string
	Type Type
}

func (m *MetricId) TagKeys() []string {
	keys := make([]string, 0, len(m.Tags))
	for k := range m.Tags {
		keys = append(keys, k)
	}
	return keys
}

// MetricSample 一个指标的完整定义，包含值，这是指标的最终呈现，不是中间值(如 summary，histogram 他们统计完后会导出为一组 MetricSample)
type MetricSample struct {
	*MetricId
	value float64
}

// CounterMetric 指标抽象接口
type CounterMetric interface {
	Inc()
	Add(float64)
}

// GaugeMetric 指标抽象接口
type GaugeMetric interface {
	Set(float64)
	// Inc()
	// Dec()
	// Add(float64)
	// Sub(float64)
}

// HistogramMetric 指标抽象接口
type HistogramMetric interface {
	Record(float64)
}

// SummaryMetric 指标抽象接口
type SummaryMetric interface {
	Record(float64)
}

// StatesMetrics 综合指标，包括总数、成功数，失败数，调用 MetricsRegistry 实现最终暴露
type StatesMetrics interface {
	Success()
	AddSuccess(float64)
	Fail()
	AddFailed(float64)
}

func NewStatesMetrics(total *MetricId, succ *MetricId, fail *MetricId) StatesMetrics {
	return &DefaultStatesMetric{total: total, succ: succ, fail: fail, r: registry}
}

// TimeMetrics 综合指标, 包括 min(Gauge)、max(Gauge)、avg(Gauge)、sum(Gauge)、last(Gauge)，调用 MetricRegistry 实现最终暴露
// 参见 dubbo-java org.apache.dubbo.metrics.aggregate.TimeWindowAggregator 类实现
type TimeMetrics interface {
	Record(float64)
}

// NewTimeMetrics init and write all data to registry 
func NewTimeMetrics(min *MetricId, avg *MetricId, max *MetricId, last *MetricId, sum *MetricId) {

}

type DefaultStatesMetric struct {
	r     MetricRegistry
	total *MetricId
	succ  *MetricId
	fail  *MetricId
}

func (c DefaultStatesMetric) Success() {
	c.r.Counter(c.total).Inc()
	c.r.Counter(c.succ).Inc()
}

func (c DefaultStatesMetric) AddSuccess(v float64) {
	c.r.Counter(c.total).Add(v)
	c.r.Counter(c.succ).Add(v)
}

func (c DefaultStatesMetric) Fail() {
	c.r.Counter(c.total).Inc()
	c.r.Counter(c.fail).Inc()
}

func (c DefaultStatesMetric) AddFailed(v float64) {
	c.r.Counter(c.total).Add(v)
	c.r.Counter(c.fail).Add(v)
}
