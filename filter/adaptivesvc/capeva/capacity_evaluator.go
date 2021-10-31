package capeva

type CapacityEvaluator interface {
	// Estimated is estimated capacity, which reflects the maximum requests handled by the provider
	Estimated() int64
	UpdateEstimated(value int64) error

	// Actual is actual requests on the provider
	Actual() int64
	UpdateActual(value int64) error
}