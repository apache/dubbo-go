package capeva

type CapacityEvaluator interface {
	// EstimatedCapacity is estimated capacity, which reflects the maximum requests handled by the provider.
	EstimatedCapacity() int64

	// ActualCapacity is actual requests on the provider.
	ActualCapacity() int64

	// NewCapacityUpdater returns a capacity updater
	NewCapacityUpdater() CapacityUpdater
}
