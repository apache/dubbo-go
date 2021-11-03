package capeva

type CapacityEvaluator interface {
	// Estimated is estimated capacity, which reflects the maximum requests handled by the provider.
	Estimated() int64
	UpdateEstimated(value int64)

	// Actual is actual requests on the provider.
	Actual() int64
	// UpdateActual updates by delta, for instance, if a new request comes, the delta should be 1.
	UpdateActual(delta int64)

	// NewCapacityUpdater returns a capacity updater
	NewCapacityUpdater() CapacityUpdater
}

// CapacityUpdater updates capacity evaluator.
// Each method has a stand-alone updater instance, it could be passed by the invocation.
type CapacityUpdater interface {
	// Succeed updates capacity evaluator if the invocation finish successfully.
	Succeed()
	// Failed updates capacity evaluator if the invocation finish unsuccessfully.
	Failed()
}